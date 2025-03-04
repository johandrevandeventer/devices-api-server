package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/johandrevandeventer/devices-api-server/cmd"
	"github.com/johandrevandeventer/devices-api-server/initializers"
	"github.com/johandrevandeventer/devices-api-server/internal/config"
	"github.com/johandrevandeventer/devices-api-server/internal/engine"
	"github.com/johandrevandeventer/logging"
	"github.com/johandrevandeventer/splashscreen"
	"go.uber.org/zap"
)

func main() {
	var wg sync.WaitGroup

	// Increase WaitGroup counter
	wg.Add(1)

	splashscreen.PrintSplashScreen()

	cmd.Execute()

	initializers.LoadEnvVariable()
	initializers.InitConfig()
	cfg := config.GetConfig()

	initializers.InitLogger(cfg)

	initializers.InitDB()

	logger := logging.GetLogger("main")

	statePersister, err := initializers.InitPersist(cfg)
	if err != nil {
		logger.Error("Failed to initialize the state persister", zap.Error(err))
		os.Exit(1)
	}

	// Graceful shutdown handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	svc := engine.NewEngine(cfg, logger, statePersister)

	// Goroutine to handle stop signals or stop file detection
	go func() {
		defer wg.Done() // Ensure the WaitGroup counter is decremented

		select {
		case <-ctx.Done(): // Handle system interrupt (e.g., Ctrl+C)
			logger.Warn("Received signal to stop the application")
		case <-svc.StopFileDetected(): // Stop file detected by Engine
			logger.Warn("Stop file detected, shutting down application")
		}

		// Ensure application cleanup and shutdown
		svc.Stop() // Stop the engine
		stop()     // Cancel the context
	}()

	defer func() {
		if r := recover(); r != nil {
			logger.Error("recovered from panic", zap.Any("panic", r))
		}
	}()

	svc.Run(ctx)

	// Wait for goroutine to complete before exiting
	wg.Wait()
}
