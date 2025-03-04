package handlers

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	serverutils "github.com/johandrevandeventer/devices-api-server/internal/server/utils"
	devicesdb "github.com/johandrevandeventer/devices-api-server/pkg/db"
	"github.com/johandrevandeventer/devices-api-server/pkg/db/models"
	"gorm.io/gorm"
)

type SiteResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	CustomerID   uuid.UUID `json:"customer_id"`
	CustomerName string    `json:"customer_name"`
}

type SiteRequest struct {
	Name string `json:"name"`
}

// Route: POST /sites
// Create a new site
func SiteCreate(c *gin.Context) {
	customerID := c.Param("customer_id")

	// Validate the customer ID
	if !serverutils.IsValidUUID(customerID) {
		serverutils.WriteError(c, 400, "Invalid customer ID", "Invalid UUID format")
		return
	}

	// Parse the request body
	var body SiteRequest
	if err := c.BindJSON(&body); err != nil || body.Name == "" {
		serverutils.WriteError(c, 400, "Invalid request body", "Name field is required")
		return
	}

	// Get the database instance
	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	// Check if the customer exists
	customer, err := FetchCustomerByID(bmsDB, customerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Customer not found", "No customer found with the given ID")
		return
	} else if err != nil {
		serverutils.WriteError(c, 500, "Failed to fetch customer", err.Error())
		return
	}

	site, err := FetchSiteByName(bmsDB, body.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 500, "Failed to fetch site", err.Error())
		return
	}

	if site == nil {
		// Create new site
		newSite := models.Site{Name: body.Name, CustomerID: customer.ID}
		if err := bmsDB.DB.Create(&newSite).Error; err != nil {
			serverutils.WriteError(c, 500, "Failed to create site", err.Error())
			return
		}
		serverutils.WriteJSON(c, 200, "Site created", SiteResponse{ID: newSite.ID, Name: newSite.Name, CustomerID: customer.ID, CustomerName: customer.Name})
		return
	}

	if site.DeletedAt.Valid {
		// Restore the site
		now := time.Now()
		site.DeletedAt = gorm.DeletedAt{}
		site.CreatedAt, site.UpdatedAt = now, now

		if err := bmsDB.DB.Unscoped().Save(&site).Error; err != nil {
			serverutils.WriteError(c, 500, "Failed to restore site", err.Error())
			return
		}
		serverutils.WriteJSON(c, 200, "Site restored", SiteResponse{ID: site.ID, Name: site.Name, CustomerID: customer.ID, CustomerName: customer.Name})
		return
	}

	serverutils.WriteError(c, 400, "Site already exists", "A site with this name already exists")
}

// Route: GET /sites
// Fetch all sites
func SiteFetchAll(c *gin.Context) {
	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	var sites []models.Site
	if err := bmsDB.DB.Find(&sites).Error; err != nil {
		serverutils.WriteError(c, 500, "Failed to fetch sites", err.Error())
		return
	}

	var response []SiteResponse
	for _, site := range sites {
		customer, err := FetchCustomerByID(bmsDB, site.CustomerID.String())
		if err != nil {
			serverutils.WriteError(c, 500, "Failed to fetch customer", err.Error())
			return
		}

		response = append(response, SiteResponse{ID: site.ID, Name: site.Name, CustomerID: customer.ID, CustomerName: customer.Name})
	}

	serverutils.WriteJSON(c, 200, "Sites fetched", response)
}

// Route: GET /sites/:site_id
// Fetch a site by ID
func SiteFetchByID(c *gin.Context) {
	role := c.GetString("role")
	requesterID := c.GetString("user_id")
	siteID := c.Param("site_id")

	// Validate the site ID
	if !serverutils.IsValidUUID(siteID) {
		serverutils.WriteError(c, 400, "Invalid site ID", "Invalid UUID format")
		return
	}

	// Get the database instance
	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	// Fetch the site
	site, err := FetchSiteByID(bmsDB, siteID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Site not found", "No site found with the given ID")
		return
	} else if err != nil {
		serverutils.WriteError(c, 500, "Failed to fetch site", err.Error())
		return
	}

	// Fetch the customer
	customer, err := FetchCustomerByID(bmsDB, site.CustomerID.String())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Customer not found", "No customer found with the given ID")
		return
	} else if err != nil {
		serverutils.WriteError(c, 500, "Failed to fetch customer", err.Error())
		return
	}

	// Check if the requester is an admin or the site owner
	if role != "admin" && requesterID != customer.ID.String() {
		serverutils.WriteError(c, 403, "Forbidden", "You are not authorized to access this site")
		return
	}

	serverutils.WriteJSON(c, 200, "Site fetched", SiteResponse{ID: site.ID, Name: site.Name, CustomerID: customer.ID, CustomerName: customer.Name})
}

// Route: GET /customers/:customer_id/sites
// Fetch all sites for a customer
func SiteFetchByCustomerID(c *gin.Context) {
	role := c.GetString("role")
	requesterID := c.GetString("user_id")
	customerID := c.Param("customer_id")

	// Validate the customer ID
	if !serverutils.IsValidUUID(customerID) {
		serverutils.WriteError(c, 400, "Invalid customer ID", "Invalid UUID format")
		return
	}

	// Get the database instance
	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	// Fetch the customer
	customer, err := FetchCustomerByID(bmsDB, customerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Customer not found", "No customer found with the given ID")
		return
	} else if err != nil {
		serverutils.WriteError(c, 500, "Failed to fetch customer", err.Error())
		return
	}

	// Check if the requester is an admin or the customer owner
	if role != "admin" && requesterID != customer.ID.String() {
		serverutils.WriteError(c, 403, "Forbidden", "You are not authorized to access this customer's sites")
		return
	}

	// Fetch the sites
	var sites []models.Site
	if err := bmsDB.DB.Where("customer_id = ?", customer.ID).Find(&sites).Error; err != nil {
		serverutils.WriteError(c, 500, "Failed to fetch sites", err.Error())
		return
	}

	var response []SiteResponse
	for _, site := range sites {
		response = append(response, SiteResponse{ID: site.ID, Name: site.Name, CustomerID: customer.ID, CustomerName: customer.Name})
	}

	serverutils.WriteJSON(c, 200, "Sites fetched", response)
}

// Route: PUT /sites/:site_id
// Update a site by ID
func SiteUpdate(c *gin.Context) {
	siteID := c.Param("site_id")

	// Validate the site ID
	if !serverutils.IsValidUUID(siteID) {
		serverutils.WriteError(c, 400, "Invalid site ID", "Invalid UUID format")
		return
	}

	// Parse the request body
	var body SiteRequest
	if err := c.BindJSON(&body); err != nil || body.Name == "" {
		serverutils.WriteError(c, 400, "Invalid request body", "Name field is required")
		return
	}

	// Get the database instance
	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	// Fetch the site
	site, err := FetchSiteByID(bmsDB, siteID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Site not found", "No site found with the given ID")
		return
	}

	if result := bmsDB.DB.Model(&site).Select("Name").Updates(models.Site{Name: body.Name}); result.Error != nil {
		serverutils.WriteError(c, 500, "Failed to update site", result.Error.Error())
		return
	}

	serverutils.WriteJSON(c, 200, "Site updated", SiteResponse{ID: site.ID, Name: site.Name, CustomerID: site.Customer.ID, CustomerName: site.Customer.Name})
}

// Route: DELETE /sites/:site_id
// Delete a site by ID
func SiteDelete(c *gin.Context) {
	siteID := c.Param("site_id")

	// Validate the site ID
	if !serverutils.IsValidUUID(siteID) {
		serverutils.WriteError(c, 400, "Invalid site ID", "Invalid UUID format")
		return
	}

	// Get the database instance
	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	// Fetch the site
	site, err := FetchSiteByID(bmsDB, siteID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Site not found", "No site found with the given ID")
		return
	}

	if err := bmsDB.DB.Delete(&site).Error; err != nil {
		serverutils.WriteError(c, 500, "Failed to delete site", err.Error())
		return
	}

	serverutils.WriteJSON(c, 200, "Site deleted", nil)
}

// =====================================================================================================================

// Fetch a site by ID and preload the associated Customer
func FetchSiteByID(bmsDB *devicesdb.BMS_DB, id string) (*models.Site, error) {
	var site models.Site
	result := bmsDB.DB.Preload("Customer").First(&site, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &site, nil
}

// Fetch a site by Name (including soft-deleted records)
func FetchSiteByName(bmsDB *devicesdb.BMS_DB, name string) (*models.Site, error) {
	var site models.Site
	result := bmsDB.DB.Unscoped().Where("name = ?", name).First(&site)
	if result.Error != nil {
		return nil, result.Error
	}
	return &site, nil
}
