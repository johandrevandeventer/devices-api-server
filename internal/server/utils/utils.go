package serverutils

import (
	"errors"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	devicesdb "github.com/johandrevandeventer/devices-api-server/pkg/db"
	"github.com/johandrevandeventer/logging"
	"go.uber.org/zap"
)

// Response structure for JSON responses.
type Response struct {
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Claims represents the structure of the JWT claims for the admin route.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"user_name"`
	Role     string `json:"role"`
	Action   string `json:"action"`
	Issuer   string `json:"issuer"`
	IssuedAt int64  `json:"issued_at"`
	jwt.RegisteredClaims
}

// WriteJSON sends a JSON response with the provided status code, message, and data.
func WriteJSON(c *gin.Context, status int, message string, data any) {
	response := Response{
		Status:  status,
		Message: message,
		Data:    data,
	}

	c.JSON(status, response)
}

// WriteError sends an error response with a status code and logs the error.
func WriteError(c *gin.Context, status int, message, errMsg string) {
	response := Response{
		Status:  status,
		Message: message,
		Error:   errMsg,
	}

	c.JSON(status, response)

	// Log the error
	logger := logging.GetLogger("api-server")
	logger.Error(response.Message, zap.String("error", errMsg))
}

// GenerateID generates a new UUID
func GenerateID() string {
	return uuid.New().String() // Example: "550e8400-e29b-41d4-a716-446655440000"
}

// GenerateJWT generates a new JWT token for a user
func GenerateJWT(userID, username, role, action string, expire bool) (string, error) {
	if !IsValidUUID(userID) {
		return "", errors.New("invalid user ID")
	}

	if !IsValidString(username) {
		return "", errors.New("invalid username")
	}

	if !IsValidRole(role) {
		return "", errors.New("invalid role")
	}

	if !IsValidAction(action) {
		return "", errors.New("invalid action")
	}

	// Create the claims
	claims := Claims{
		UserID:           userID,
		Username:         username,
		Role:             role,
		Action:           action,
		Issuer:           "Rubicon BMS",
		IssuedAt:         time.Now().Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			// ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Token expires in 24 hours
		},
	}

	if expire {
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(24 * time.Hour * 30))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	JwtSecret := os.Getenv("DEVICES_SERVER_JWT_SECRET")
	if JwtSecret == "" {
		return "", errors.New("DEVICES_SERVER_JWT_SECRET is not set")
	}

	return token.SignedString([]byte(JwtSecret))
}

// ValidateJWT validates the JWT token and extracts claims
func ValidateJWT(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}

		JWTSecret := os.Getenv("DEVICES_SERVER_JWT_SECRET")
		if JWTSecret == "" {
			return nil, errors.New("DEVICES_SERVER_JWT_SECRET is not set")
		}

		return []byte(JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// Helper function to get database instance
// GetDBInstance returns the database instance or handles the error.
func GetDBInstance(c *gin.Context) (*devicesdb.BMS_DB, bool) {
	bmsDB, err := devicesdb.GetDB()
	if err != nil {
		WriteError(c, http.StatusInternalServerError, "Database error", err.Error())
		return nil, false
	}
	return bmsDB, true
}

// IsValidUUID checks if a string is a valid UUID.
func IsValidUUID(s string) bool {
	if _, err := uuid.Parse(s); err != nil {
		return false
	}
	return true
}

// IsValidString checks if a string is valid.
func IsValidString(s string) bool {
	stringRegex := `^[a-zA-Z0-9_ ]{3,20}$`
	return regexp.MustCompile(stringRegex).MatchString(s)
}

// IsValidRole checks if a role is valid.
func IsValidRole(role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsValidAction checks if an action is valid.
func IsValidAction(action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}
