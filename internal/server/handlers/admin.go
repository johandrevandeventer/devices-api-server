package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	serverutils "github.com/johandrevandeventer/devices-api-server/internal/server/utils"
	devicesdb "github.com/johandrevandeventer/devices-api-server/pkg/db"
	"github.com/johandrevandeventer/devices-api-server/pkg/db/models"
	"gorm.io/gorm"
)

// Route: GenerateAdminToken (Admin Only)
func GenerateAdminTokenHandler(c *gin.Context) {
	userID := serverutils.GenerateID()

	// Generate the JWT token
	token, err := serverutils.GenerateJWT(userID, "Admin", "admin", "ADMIN", false)
	if err != nil {
		serverutils.WriteError(c, http.StatusInternalServerError, "Failed to generate token", err.Error())
		return
	}

	serverutils.WriteJSON(c, http.StatusOK, "Token generated successfully", token)
}

// Route: GenerateJWTToken (Admin Only)
func GenerateTokenHandler(c *gin.Context) {
	// Get data off request body
	var body struct {
		CustomerID string `json:"customer_id"`
		Action     string `json:"action"`
	}
	if err := c.BindJSON(&body); err != nil {
		serverutils.WriteError(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate the customer_id field
	if body.CustomerID == "" {
		serverutils.WriteError(c, http.StatusBadRequest, "Invalid request body", "Customer ID field is required")
		return
	}

	if !serverutils.IsValidUUID(body.CustomerID) {
		serverutils.WriteError(c, http.StatusBadRequest, "Invalid request body", "Invalid Customer ID")
		return
	}

	// Validate the action field
	if body.Action == "" {
		serverutils.WriteError(c, http.StatusBadRequest, "Invalid request body", "Action field is required")
		return
	}

	// Check if the action is allowed
	if !serverutils.IsValidAction(body.Action) {
		serverutils.WriteError(c, http.StatusBadRequest, "Invalid request body", "Action not allowed")
		return
	}

	// Get the database instance
	bmsDB, err := devicesdb.GetDB()
	if err != nil {
		serverutils.WriteError(c, http.StatusInternalServerError, "Failed to get database instance", err.Error())
		return
	}

	// Check if the customer exists
	var customer models.Customer
	result := bmsDB.DB.First(&customer, "id = ?", body.CustomerID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, http.StatusNotFound, "Customer not found", "Customer does not exist")
		return
	} else if result.Error != nil {
		serverutils.WriteError(c, http.StatusInternalServerError, "Database error", result.Error.Error())
		return
	}

	// Generate the JWT token
	token, err := serverutils.GenerateJWT(body.CustomerID, customer.Name, "user", body.Action, false)
	if err != nil {
		serverutils.WriteError(c, http.StatusInternalServerError, "Failed to generate token", err.Error())
		return
	}

	// Create the AuthToken record
	authToken := models.AuthToken{
		CustomerID: customer.ID,
		Action:     body.Action,
		Token:      token,
	}

	// Save the AuthToken to the database
	if err := bmsDB.DB.Create(&authToken).Error; err != nil {
		serverutils.WriteError(c, http.StatusInternalServerError, "Failed to save token", err.Error())
		return
	}

	// Preload the Customer details
	if err := bmsDB.DB.Preload("Customer").First(&authToken, authToken.ID).Error; err != nil {
		serverutils.WriteError(c, http.StatusInternalServerError, "Failed to fetch token details", err.Error())
		return
	}

	// Return the response with the AuthToken and preloaded Customer details
	serverutils.WriteJSON(c, http.StatusOK, "Token generated successfully", authToken)
}
