package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	serverutils "github.com/johandrevandeventer/devices-api-server/internal/server/utils"
	devicesdb "github.com/johandrevandeventer/devices-api-server/pkg/db"
	"github.com/johandrevandeventer/devices-api-server/pkg/db/models"
)

// Route: Authenticate
// Authenticate a user from the request body using JWT
func AuthenticateHandler(c *gin.Context) {
	// Get data off request body
	var body struct {
		Token string `json:"token"`
	}
	if err := c.BindJSON(&body); err != nil {
		serverutils.WriteError(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate the token field
	if body.Token == "" {
		serverutils.WriteError(c, http.StatusBadRequest, "Invalid request body", "Token field is required")
		return
	}

	// Validate the JWT token
	claims, err := serverutils.ValidateJWT(body.Token)
	if err != nil {
		serverutils.WriteError(c, http.StatusUnauthorized, "Invalid token", err.Error())
		return
	}

	// Get database instance
	bmsDB, err := devicesdb.GetDB()
	if err != nil {
		serverutils.WriteError(c, http.StatusInternalServerError, "Failed to get database instance", err.Error())
		return
	}

	role := claims["role"].(string)
	if role != "admin" {
		// See if the token exists in the database
		var token models.AuthToken
		bmsDB.DB.First(&token, "token = ?", body.Token)
		if token.Token == "" {
			serverutils.WriteError(c, http.StatusUnauthorized, "Invalid token", "Token not found")
			return
		}
	}

	// Set the claims to the cookie
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("Authorization", body.Token, 3600*24, "", "", false, true)

	serverutils.WriteJSON(c, http.StatusOK, "Token validated", nil)
}
