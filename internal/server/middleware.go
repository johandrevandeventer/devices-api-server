package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	serverutils "github.com/johandrevandeventer/devices-api-server/internal/server/utils"
	devicesdb "github.com/johandrevandeventer/devices-api-server/pkg/db"
	"github.com/johandrevandeventer/devices-api-server/pkg/db/models"
	"go.uber.org/zap"
)

// loggingMiddleware logs HTTP requests with response status and duration
func loggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log request details
		statusCode := c.Writer.Status()
		logEntry := logger.Info
		if statusCode >= 400 {
			logEntry = logger.Warn
		}

		logEntry("Request completed",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("remoteAddr", c.ClientIP()),
			zap.Int("statusCode", statusCode),
			zap.Duration("duration", time.Since(start)),
		)
	}
}

// AdminMiddleware is a Gin middleware to check for a valid admin secret
func AdminMiddleware(adminSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the "Admin-Secret" header from the request
		secret := c.GetHeader("Admin-Secret")

		// Check if the secret matches the expected admin secret
		if secret != adminSecret {
			// If the secret is invalid, return a 401 Unauthorized response
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Invalid admin secret",
			})
			c.Abort() // Stop further processing of the request
			return
		}

		// If the secret is valid, proceed to the next handler
		c.Next()
	}
}

func AdminOnlyMiddleware(c *gin.Context) {
	role := c.GetString("role")
	if role != "admin" {
		serverutils.WriteError(c, 403, "Unauthorized", "Only admins can perform this action")
		c.Abort()
		return
	}
	c.Next()
}

// AuthMiddleware is a Gin middleware to check for a valid JWT token
func AuthMiddleware(c *gin.Context) {
	// Get the cookie off request
	tokenString, err := c.Cookie("Authorization")
	if err != nil {
		serverutils.WriteError(c, http.StatusUnauthorized, "Unauthorized", "Please authenticate first")
		c.Abort()
		return
	}

	// Validate the JWT token
	claims, err := serverutils.ValidateJWT(tokenString)
	if err != nil {
		serverutils.WriteError(c, http.StatusUnauthorized, "Unauthorized", "Invalid token")
		c.Abort()
		return
	}

	// Get database instance
	bmsDB, err := devicesdb.GetDB()
	if err != nil {
		serverutils.WriteError(c, http.StatusInternalServerError, "Failed to get database instance", err.Error())
		c.Abort()
		return
	}

	role := claims["role"].(string)
	if role != "admin" {
		var token models.AuthToken
		bmsDB.DB.First(&token, "customer_id = ? and action = ?", claims["user_id"], claims["action"])
		if token.Token == "" {
			serverutils.WriteError(c, http.StatusUnauthorized, "Unauthorized", "Token not found")
			c.Abort()
			return
		}
	}

	// Set the claims to the context
	c.Set("customer_id", claims["user_id"])
	c.Set("role", claims["role"])
	c.Set("action", claims["action"])
}
