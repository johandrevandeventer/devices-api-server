package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/johandrevandeventer/devices-api-server/internal/server/handlers"
	serverutils "github.com/johandrevandeventer/devices-api-server/internal/server/utils"
	"github.com/johandrevandeventer/logging"
	"go.uber.org/zap"
)

// APIServer structure for the API server.
type APIServer struct {
	listenAddr string
	logger     *zap.Logger
}

// Custom writer to redirect logs
type zapRedirectWriter struct {
	logger *zap.Logger
}

func (w zapRedirectWriter) Write(p []byte) (n int, err error) {
	trimmedMessage := strings.TrimSpace(string(p))
	w.logger.Debug(trimmedMessage)
	return len(p), nil
}

func NewApiServer() *APIServer {
	logger := logging.GetLogger("api-server")

	port := os.Getenv("DEVICES_SERVER_PORT")
	if port == "" {
		logger.Fatal("PORT environment variable is not set")
	}

	return &APIServer{
		listenAddr: fmt.Sprintf(":%s", port),
		logger:     logger,
	}
}

// Start the API server
func (s *APIServer) Start() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = zapRedirectWriter{logger: s.logger}      // Redirects Gin debug logs
	gin.DefaultErrorWriter = zapRedirectWriter{logger: s.logger} // Redirects Gin error logs

	r := gin.New()

	// Middleware
	r.Use(loggingMiddleware(s.logger))
	r.Use(gin.Recovery())

	// Handle 404 (Not Found)
	r.NoRoute(notFoundHandler())

	// Handle 405 (Method Not Allowed)
	r.NoMethod(methodNotAllowedHandler())

	// Setup the routes
	s.setupRoutes(r)

	// Start the server with HTTPS
	certFile := "server.crt"
	keyFile := "server.key"

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		s.logger.Fatal("Certificate file not found", zap.String("certFile", certFile))
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		s.logger.Fatal("Private key file not found", zap.String("keyFile", keyFile))
	}

	// Create a custom HTTP server with TLS
	server := &http.Server{
		Addr:     s.listenAddr,
		Handler:  r,
		ErrorLog: zap.NewStdLog(s.logger), // Redirect server logs to zap logger
	}

	s.logger.Info("Starting HTTPS server", zap.String("port", s.listenAddr))
	if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
		s.logger.Fatal("Failed to start HTTPS server", zap.Error(err))
	}
}

// Setup the routes
func (s *APIServer) setupRoutes(r *gin.Engine) {
	adminSecret := os.Getenv("DEVICES_SERVER_ADMIN_SECRET")
	if adminSecret == "" {
		s.logger.Fatal("DEVICES_SERVER_ADMIN_SECRET environment variable is not set")
	}

	r.GET("/health", handlers.HealthHandler)

	adminGroup := r.Group("/admin")
	adminGroup.Use(AdminMiddleware(adminSecret))
	{
		adminGroup.POST("/generate-admin-token", handlers.GenerateAdminTokenHandler)
		adminGroup.POST("/generate-token", handlers.GenerateTokenHandler)
	}

	// Authenticate
	r.POST("/authenticate", handlers.AuthenticateHandler)

	protectedGroup := r.Group("")
	protectedGroup.Use(AuthMiddleware)
	{
		// Customer routes
		protectedGroup.POST("/customers", AdminOnlyMiddleware, handlers.CustomerCreate)
		protectedGroup.GET("/customers", AdminOnlyMiddleware, handlers.CustomerFetchAll)
		protectedGroup.GET("/customers/:customer_id", handlers.CustomerFetchByID)
		protectedGroup.PUT("/customers/:customer_id", AdminOnlyMiddleware, handlers.CustomerUpdate)
		protectedGroup.DELETE("/customers/:customer_id", AdminOnlyMiddleware, handlers.CustomerDelete)

		// Site routes
		protectedGroup.POST("/customers/:customer_id/sites", AdminOnlyMiddleware, handlers.SiteCreate)
		protectedGroup.GET("/customers/:customer_id/sites", handlers.SiteFetchByCustomerID)
		protectedGroup.GET("/sites", AdminOnlyMiddleware, handlers.SiteFetchAll)
		protectedGroup.GET("/sites/:site_id", handlers.SiteFetchByID)
		protectedGroup.PUT("/sites/:site_id", AdminOnlyMiddleware, handlers.SiteUpdate)
		protectedGroup.DELETE("/sites/:site_id", AdminOnlyMiddleware, handlers.SiteDelete)

		// Device routes
		protectedGroup.POST("/customers/:customer_id/sites/:site_id/devices", AdminOnlyMiddleware, handlers.DeviceCreate)
		protectedGroup.GET("/devices", handlers.DeviceFetchAll)
		protectedGroup.GET("/customers/:customer_id/devices", handlers.DeviceFetchByCustomerID)
		protectedGroup.GET("/sites/:site_id/devices", handlers.DeviceFetchBySiteID)
		protectedGroup.GET("/devices/:device_serial_number", handlers.DeviceFetchBySerialNumber)
		protectedGroup.PUT("/devices/:device_serial_number", AdminOnlyMiddleware, handlers.DeviceUpdate)
		protectedGroup.DELETE("/devices/:device_serial_number", AdminOnlyMiddleware, handlers.DeviceDelete)
	}
}

// methodNotAllowedHandler handles unsupported HTTP methods.
func methodNotAllowedHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		msg := fmt.Sprintf("Method Not Allowed: (%s) - '%s'", c.Request.Method, c.Request.RequestURI)
		err := fmt.Sprintf("(%d) Method not allowed", http.StatusMethodNotAllowed)
		serverutils.WriteError(c, http.StatusMethodNotAllowed, msg, err)
	}
}

// notFoundHandler handles unknown routes.
func notFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// msg := "Route Not Found: " + c.Request.RequestURI
		msg := fmt.Sprintf("Route Not Found: (%s) - '%s'", c.Request.Method, c.Request.RequestURI)
		err := fmt.Sprintf("(%d) Route not found", http.StatusNotFound)
		serverutils.WriteError(c, http.StatusNotFound, msg, err)
	}
}
