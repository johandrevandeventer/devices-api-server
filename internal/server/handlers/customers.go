package handlers

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	serverutils "github.com/johandrevandeventer/devices-api-server/internal/server/utils"
	devicesdb "github.com/johandrevandeventer/devices-api-server/pkg/db"
	"github.com/johandrevandeventer/devices-api-server/pkg/db/models"
	"gorm.io/gorm"
)

type CustomerResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type CustomerRequest struct {
	Name string `json:"name"`
}

// Create a new customer or restore a soft-deleted one
func CustomerCreate(c *gin.Context) {
	var body CustomerRequest

	if err := c.BindJSON(&body); err != nil || body.Name == "" {
		serverutils.WriteError(c, 400, "Invalid request body", "Name field is required")
		return
	}

	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	customer, err := FetchCustomerByName(bmsDB, body.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 500, "Failed to fetch customer", err.Error())
		return
	}

	if customer == nil {
		// Create new customer
		newCustomer := models.Customer{Name: body.Name}
		if err := bmsDB.DB.Create(&newCustomer).Error; err != nil {
			serverutils.WriteError(c, 500, "Failed to create customer", err.Error())
			return
		}
		serverutils.WriteJSON(c, 201, "Customer created", CustomerResponse{ID: newCustomer.ID, Name: newCustomer.Name})
		return
	}

	// Restore soft-deleted customer
	if customer.DeletedAt.Valid {
		now := time.Now()
		customer.DeletedAt = gorm.DeletedAt{}
		customer.CreatedAt, customer.UpdatedAt = now, now

		if err := bmsDB.DB.Unscoped().Save(&customer).Error; err != nil {
			serverutils.WriteError(c, 500, "Failed to restore customer", err.Error())
			return
		}
		serverutils.WriteJSON(c, 200, "Customer restored", CustomerResponse{ID: customer.ID, Name: customer.Name})
		return
	}

	serverutils.WriteError(c, 400, "Customer already exists", "A customer with this name already exists")
}

// Get all customers
func CustomerFetchAll(c *gin.Context) {
	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	var customers []models.Customer
	if err := bmsDB.DB.Find(&customers).Error; err != nil {
		serverutils.WriteError(c, 500, "Failed to fetch customers", err.Error())
		return
	}

	customerResponses := make([]CustomerResponse, len(customers))
	for i, customer := range customers {
		customerResponses[i] = CustomerResponse{ID: customer.ID, Name: customer.Name}
	}

	serverutils.WriteJSON(c, 200, "Customers fetched", customerResponses)
}

// Get a customer by ID
func CustomerFetchByID(c *gin.Context) {
	id := c.Param("customer_id")
	fmt.Println("ID: ", id)
	if !serverutils.IsValidUUID(id) {
		serverutils.WriteError(c, 400, "Invalid customer ID", "Invalid UUID format")
		return
	}

	role := c.GetString("role")
	requesterID := c.GetString("customer_id")
	if role != "admin" && requesterID != id {
		serverutils.WriteError(c, 403, "Forbidden", "Unauthorized access")
		return
	}

	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	customer, err := FetchCustomerByID(bmsDB, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Customer not found", "No customer found with the given ID")
		return
	} else if err != nil {
		serverutils.WriteError(c, 500, "Failed to fetch customer", err.Error())
		return
	}

	serverutils.WriteJSON(c, 200, "Customer fetched", CustomerResponse{ID: customer.ID, Name: customer.Name})
}

// Update a customer by ID
func CustomerUpdate(c *gin.Context) {
	role := c.GetString("role")
	if role != "admin" {
		serverutils.WriteError(c, 403, "Forbidden", "Unauthorized access")
		return
	}

	id := c.Param("customer_id")
	if !serverutils.IsValidUUID(id) {
		serverutils.WriteError(c, 400, "Invalid customer ID", "Invalid UUID format")
		return
	}

	var body CustomerRequest
	if err := c.BindJSON(&body); err != nil || body.Name == "" {
		serverutils.WriteError(c, 400, "Invalid request body", "Name field is required")
		return
	}

	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	customer, err := FetchCustomerByID(bmsDB, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Customer not found", "No customer found with the given ID")
		return
	} else if err != nil {
		serverutils.WriteError(c, 500, "Failed to fetch customer", err.Error())
		return
	}

	if err := bmsDB.DB.Model(customer).Update("name", body.Name).Error; err != nil {
		serverutils.WriteError(c, 500, "Failed to update customer", err.Error())
		return
	}

	serverutils.WriteJSON(c, 200, "Customer updated", CustomerResponse{ID: customer.ID, Name: body.Name})
}

// Delete a customer by ID
func CustomerDelete(c *gin.Context) {
	role := c.GetString("role")
	if role != "admin" {
		serverutils.WriteError(c, 403, "Forbidden", "Unauthorized access")
		return
	}

	id := c.Param("customer_id")
	if !serverutils.IsValidUUID(id) {
		serverutils.WriteError(c, 400, "Invalid customer ID", "Invalid UUID format")
		return
	}

	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	// Delete the customer from the database
	if err := bmsDB.DB.Delete(&models.Customer{}, "id = ?", id).Error; err != nil {
		serverutils.WriteError(c, 500, "Failed to delete customer", err.Error())
		return
	}

	serverutils.WriteJSON(c, 200, "Customer deleted", nil)
}

// =====================================================================================================================

// Fetch a customer by ID
func FetchCustomerByID(bmsDB *devicesdb.BMS_DB, id string) (*models.Customer, error) {
	var customer models.Customer
	result := bmsDB.DB.First(&customer, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &customer, nil
}

// Fetch a customer by Name (including soft-deleted records)
func FetchCustomerByName(bmsDB *devicesdb.BMS_DB, name string) (*models.Customer, error) {
	var customer models.Customer
	result := bmsDB.DB.Unscoped().Where("name = ?", name).First(&customer)
	if result.Error != nil {
		return nil, result.Error
	}
	return &customer, nil
}
