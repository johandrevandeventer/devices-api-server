package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/johandrevandeventer/devices-api-server/internal/config"
	"github.com/johandrevandeventer/devices-api-server/internal/flags"
	"github.com/johandrevandeventer/devices-api-server/internal/server"
	coreutils "github.com/johandrevandeventer/devices-api-server/utils"
	"github.com/johandrevandeventer/persist"
	"go.uber.org/zap"
)

var (
	tmpFilePath            string
	stopFileFilePath       string
	connectionsLogFilePath string

	startTime time.Time
	endTime   time.Time
)

// NewEngine creates a new Engine instance
func NewEngine(cfg *config.Config, logger *zap.Logger, statePersister *persist.FilePersister) *Engine {
	tmpFilePath = cfg.App.Runtime.TmpDir
	stopFileFilePath = cfg.App.Runtime.StopFileFilepath
	connectionsLogFilePath = cfg.App.Runtime.ConnectionsLogFilePath

	return &Engine{
		cfg:            cfg,
		logger:         logger,
		statePersister: statePersister,
		stopFileChan:   make(chan struct{}), // Initialize stop file channel
	}
}

// Run starts the Engine
func (e *Engine) Run(ctx context.Context) {
	e.ctx = ctx
	defer e.Cleanup()

	e.logger.Info("Starting application")

	connectionsLogFilePathDir := filepath.Dir(connectionsLogFilePath)

	err := coreutils.CreateTmpDir(tmpFilePath)
	if err != nil {
		e.logger.Error("Failed to create tmp directory", zap.Error(err))
	}

	e.verboseDebug("Creating tmp directory", zap.String("path", filepath.ToSlash(tmpFilePath)))
	e.verboseDebug("Creating connections directory", zap.String("path", filepath.ToSlash(connectionsLogFilePathDir)))

	startTime = time.Now()

	e.statePersister.Set("app", map[string]any{})
	e.statePersister.Set("app.status", "running")
	e.statePersister.Set("app.name", e.cfg.System.AppName)
	e.statePersister.Set("app.version", e.cfg.System.AppVersion)
	e.statePersister.Set("app.release_date", e.cfg.System.ReleaseDate)
	e.statePersister.Set("app.environment", flags.FlagEnvironment)
	e.statePersister.Set("app.start_time", startTime.Format(time.RFC3339))

	coreutils.WriteToLogFile(connectionsLogFilePath, fmt.Sprintf("%s: App started\n", startTime.Format(time.RFC3339)))

	e.start()

	// Main Engine logic
	<-e.ctx.Done()
}

func (e *Engine) start() {
	e.WatchStopFile(stopFileFilePath)

	server := server.NewApiServer()

	go server.Start()

	coreutils.WriteToLogFile(connectionsLogFilePath, fmt.Sprintf("%s: Server started\n", time.Now().Format(time.RFC3339)))

	e.statePersister.Set("app.server", map[string]any{})
	e.statePersister.Set("app.server.status", "running")
}

// Cleanup performs cleanup operations
func (e *Engine) Cleanup() {
	// Perform Cleanup
	e.verboseDebug("Cleaning up")
	defer e.verboseDebug("Cleanup complete")

	// Delete the `tmp` directory if it exists
	response, err := coreutils.CleanTmpDir(tmpFilePath)
	if err != nil {
		e.logger.Error("Failed to clean tmp directory", zap.Error(err))
	}

	if response != "" {
		e.verboseInfo(response)
	}
}

// Stop stops the Engine
func (e *Engine) Stop() {
	endTime = time.Now()

	duration := endTime.Sub(startTime)

	coreutils.WriteToLogFile(connectionsLogFilePath, fmt.Sprintf("%s: App stopped\n", endTime.Format(time.RFC3339)))
	coreutils.WriteToLogFile(connectionsLogFilePath, fmt.Sprintf("%s: Server stopped\n", endTime.Format(time.RFC3339)))
	e.logger.Info("Stopping application")

	e.statePersister.Set("app.status", "stopped")
	e.statePersister.Set("app.end_time", endTime.Format(time.RFC3339))
	e.statePersister.Set("app.duration", duration.String())
}

// WatchStopFile watches for the presence of a stop file and closes the stop file channel when the file is detected
func (e *Engine) WatchStopFile(stopFileFilePath string) {
	go func() {
		ticker := time.NewTicker(1 * time.Second) // Polling interval
		defer ticker.Stop()

		for {
			select {
			case <-e.stopFileChan: // Stop watching if channel is closed
				return
			default:
				if _, err := os.Stat(stopFileFilePath); err == nil {
					close(e.stopFileChan) // Signal stop file detection
					return
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()
}

// StopFileDetected returns a channel that is closed when the stop file is detected
func (e *Engine) StopFileDetected() <-chan struct{} {
	return e.stopFileChan
}
