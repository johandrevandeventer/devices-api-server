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

type DeviceRequest struct {
	Gateway                string `json:"gateway"`
	Controller             string `json:"controller"`
	ControllerSerialNumber string `json:"controller_serial_number"`
	DeviceType             string `json:"device_type"`
	DeviceName             string `json:"device_name"`
	DeviceSerialNumber     string `json:"device_serial_number"`
	BuildingURL            string `json:"building_url"`
	AuthToken              string `json:"auth_token"`
}

type DeviceResponse struct {
	ID                     uuid.UUID `json:"id"`
	CustomerID             uuid.UUID `json:"customer_id"`
	CustomerName           string    `json:"customer_name"`
	SiteID                 uuid.UUID `json:"site_id"`
	SiteName               string    `json:"site_name"`
	Gateway                string    `json:"gateway"`
	Controller             string    `json:"controller"`
	ControllerSerialNumber string    `json:"controller_serial_number"`
	DeviceType             string    `json:"device_type"`
	DeviceName             string    `json:"device_name"`
	DeviceSerialNumber     string    `json:"device_serial_number"`
	BuildingURL            string    `json:"building_url"`
	AuthToken              string    `json:"auth_token"`
}

// Route: POST /customers/:customer_id/sites/:site_id/devices
// Create a new device
func DeviceCreate(c *gin.Context) {
	var body DeviceRequest

	if err := c.BindJSON(&body); err != nil {
		serverutils.WriteError(c, 400, "Invalid request body", "Invalid JSON format")
		return
	}

	customerID := c.Param("customer_id")
	siteID := c.Param("site_id")

	// Validate the customer ID
	if !serverutils.IsValidUUID(customerID) {
		serverutils.WriteError(c, 400, "Invalid customer ID", "Invalid UUID format")
		return
	}

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

	// Fetch and validate customer
	customer, err := FetchCustomerByID(bmsDB, customerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Customer not found", "No customer found with the given ID")
		return
	} else if err != nil {
		serverutils.WriteError(c, 500, "Database error", err.Error())
		return
	}

	// Fetch and validate site
	site, err := FetchSiteByID(bmsDB, siteID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Site not found", "No site found with the given ID")
		return
	} else if err != nil {
		serverutils.WriteError(c, 500, "Database error", err.Error())
		return
	}

	// Check if the customer owns the site
	if site.CustomerID != customer.ID {
		serverutils.WriteError(c, 403, "Forbidden", "There is no site with the given ID for the given customer")
		return
	}

	// Check if device already exists
	device, err := FetchDeviceBySerialNumber(bmsDB, body.DeviceSerialNumber)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 500, "Database error", err.Error())
		return
	}

	if device == nil {
		// Create new device
		newDevice := models.Device{
			SiteID:                 site.ID,
			Gateway:                body.Gateway,
			Controller:             body.Controller,
			ControllerSerialNumber: body.ControllerSerialNumber,
			DeviceType:             body.DeviceType,
			DeviceName:             body.DeviceName,
			DeviceSerialNumber:     body.DeviceSerialNumber,
			BuildingURL:            body.BuildingURL,
			AuthToken:              body.AuthToken,
		}
		if err := bmsDB.DB.Create(&newDevice).Error; err != nil {
			serverutils.WriteError(c, 500, "Failed to create device", err.Error())
			return
		}
		serverutils.WriteJSON(c, 200, "Device created", DeviceResponse{
			ID:                     newDevice.ID,
			CustomerID:             customer.ID,
			CustomerName:           customer.Name,
			SiteID:                 site.ID,
			SiteName:               site.Name,
			Gateway:                newDevice.Gateway,
			Controller:             newDevice.Controller,
			ControllerSerialNumber: newDevice.ControllerSerialNumber,
			DeviceType:             newDevice.DeviceType,
			DeviceName:             newDevice.DeviceName,
			DeviceSerialNumber:     newDevice.DeviceSerialNumber,
			BuildingURL:            newDevice.BuildingURL,
			AuthToken:              newDevice.AuthToken,
		})
		return
	}

	// Restore soft-deleted device
	if device.DeletedAt.Valid {
		now := time.Now()
		device.DeletedAt = gorm.DeletedAt{}
		device.CreatedAt, device.UpdatedAt = now, now

		fmt.Println(device.Site)

		if err := bmsDB.DB.Unscoped().
			Model(&device).
			Select("deleted_at", "created_at", "updated_at").
			Updates(device).Error; err != nil {
			serverutils.WriteError(c, 500, "Failed to restore device", err.Error())
			return
		}
		serverutils.WriteJSON(c, 200, "Device restored", DeviceResponse{
			ID:                     device.ID,
			CustomerID:             customer.ID,
			CustomerName:           customer.Name,
			SiteID:                 site.ID,
			SiteName:               site.Name,
			Gateway:                device.Gateway,
			Controller:             device.Controller,
			ControllerSerialNumber: device.ControllerSerialNumber,
			DeviceType:             device.DeviceType,
			DeviceName:             device.DeviceName,
			DeviceSerialNumber:     device.DeviceSerialNumber,
			BuildingURL:            device.BuildingURL,
			AuthToken:              device.AuthToken,
		})
		return
	}

	serverutils.WriteError(c, 400, "Device already exists", "A device with this serial number already exists")
}

// Route: GET /devices
// Fetch all devices
func DeviceFetchAll(c *gin.Context) {
	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	var devices []models.Device
	if err := bmsDB.DB.Preload("Site.Customer").Find(&devices).Error; err != nil {
		serverutils.WriteError(c, 500, "Failed to fetch devices", err.Error())
		return
	}

	var response []DeviceResponse
	for _, device := range devices {
		customer, err := FetchCustomerByID(bmsDB, device.Site.CustomerID.String())
		if err != nil {
			serverutils.WriteError(c, 500, "Failed to fetch customer", err.Error())
			return
		}

		response = append(response, DeviceResponse{
			ID:                     device.ID,
			CustomerID:             customer.ID,
			CustomerName:           customer.Name,
			SiteID:                 device.Site.ID,
			SiteName:               device.Site.Name,
			Gateway:                device.Gateway,
			Controller:             device.Controller,
			ControllerSerialNumber: device.ControllerSerialNumber,
			DeviceType:             device.DeviceType,
			DeviceName:             device.DeviceName,
			DeviceSerialNumber:     device.DeviceSerialNumber,
			BuildingURL:            device.BuildingURL,
			AuthToken:              device.AuthToken,
		})
	}

	serverutils.WriteJSON(c, 200, "Devices fetched", response)
}

// Route: GET /customers/:customer_id/devices
// Fetch all devices for a customer
func DeviceFetchByCustomerID(c *gin.Context) {
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

	// Fetch and validate customer
	customer, err := FetchCustomerByID(bmsDB, customerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Customer not found", "No customer found with the given ID")
		return
	}

	if role != "admin" && customer.ID.String() != requesterID {
		serverutils.WriteError(c, 403, "Forbidden", "You are not authorized to access this customer's devices")
		return
	}

	var devices []models.Device
	if err := bmsDB.DB.Preload("Site.Customer").Where("site_id IN (SELECT id FROM sites WHERE customer_id = ?)", customer.ID).Find(&devices).Error; err != nil {
		serverutils.WriteError(c, 500, "Failed to fetch devices", err.Error())
		return
	}

	var response []DeviceResponse
	for _, device := range devices {
		response = append(response, DeviceResponse{
			ID:                     device.ID,
			CustomerID:             customer.ID,
			CustomerName:           customer.Name,
			SiteID:                 device.Site.ID,
			SiteName:               device.Site.Name,
			Gateway:                device.Gateway,
			Controller:             device.Controller,
			ControllerSerialNumber: device.ControllerSerialNumber,
			DeviceType:             device.DeviceType,
			DeviceName:             device.DeviceName,
			DeviceSerialNumber:     device.DeviceSerialNumber,
			BuildingURL:            device.BuildingURL,
			AuthToken:              device.AuthToken,
		})
	}

	serverutils.WriteJSON(c, 200, "Devices fetched", response)
}

// Route: GET /sites/:site_id/devices
// Fetch all devices for a site
func DeviceFetchBySiteID(c *gin.Context) {
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

	// Fetch and validate site
	site, err := FetchSiteByID(bmsDB, siteID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Site not found", "No site found with the given ID")
		return
	}

	var devices []models.Device
	if err := bmsDB.DB.Preload("Site.Customer").Where("site_id = ?", site.ID).Find(&devices).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			serverutils.WriteError(c, 404, "No devices found", "No devices found for the given site")
			return
		} else {
			serverutils.WriteError(c, 500, "Failed to fetch devices", err.Error())
			return
		}
	}

	var response []DeviceResponse
	for _, device := range devices {
		// customer, err := FetchCustomerByID(bmsDB, site.CustomerID.String())
		// if err != nil {
		// 	serverutils.WriteError(c, 500, "Failed to fetch customer", err.Error())
		// 	return
		// }

		response = append(response, DeviceResponse{
			ID:                     device.ID,
			CustomerID:             device.Site.Customer.ID,
			CustomerName:           device.Site.Customer.Name,
			SiteID:                 site.ID,
			SiteName:               site.Name,
			Gateway:                device.Gateway,
			Controller:             device.Controller,
			ControllerSerialNumber: device.ControllerSerialNumber,
			DeviceType:             device.DeviceType,
			DeviceName:             device.DeviceName,
			DeviceSerialNumber:     device.DeviceSerialNumber,
			BuildingURL:            device.BuildingURL,
			AuthToken:              device.AuthToken,
		})
	}

	serverutils.WriteJSON(c, 200, "Devices fetched", response)
}

// Route: GET /devices/:device_serial_number
// Fetch a device by serial number
func DeviceFetchBySerialNumber(c *gin.Context) {
	role := c.GetString("role")
	requesterID := c.GetString("user_id")
	serialNumber := c.Param("device_serial_number")

	// Get the database instance
	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	// Fetch and validate device
	device, err := FetchDeviceBySerialNumber(bmsDB, serialNumber)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Device not found", "No device found with the given serial number")
		return
	}

	// customer, err := FetchCustomerByID(bmsDB, device.Site.CustomerID.String())
	// if err != nil {
	// 	serverutils.WriteError(c, 500, "Failed to fetch customer", err.Error())
	// 	return
	// }

	if role != "admin" && device.Site.Customer.ID.String() != requesterID {
		serverutils.WriteError(c, 403, "Forbidden", "You are not authorized to access this customer's devices")
		return
	}

	serverutils.WriteJSON(c, 200, "Device fetched", DeviceResponse{
		ID:                     device.ID,
		CustomerID:             device.Site.Customer.ID,
		CustomerName:           device.Site.Customer.Name,
		SiteID:                 device.Site.ID,
		SiteName:               device.Site.Name,
		Gateway:                device.Gateway,
		Controller:             device.Controller,
		ControllerSerialNumber: device.ControllerSerialNumber,
		DeviceType:             device.DeviceType,
		DeviceName:             device.DeviceName,
		DeviceSerialNumber:     device.DeviceSerialNumber,
		BuildingURL:            device.BuildingURL,
		AuthToken:              device.AuthToken,
	})
}

// Route: PUT /devices/:device_serial_number
// Update a device
func DeviceUpdate(c *gin.Context) {
	var body DeviceRequest

	if err := c.BindJSON(&body); err != nil {
		serverutils.WriteError(c, 400, "Invalid request body", "Invalid JSON format")
		return
	}

	serialNumber := c.Param("device_serial_number")

	// Get the database instance
	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	// Fetch and validate device
	device, err := FetchDeviceBySerialNumber(bmsDB, serialNumber)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Device not found", "No device found with the given serial number")
		return
	}

	// Update the device
	device.Gateway = body.Gateway
	device.Controller = body.Controller
	device.ControllerSerialNumber = body.ControllerSerialNumber
	device.DeviceType = body.DeviceType
	device.DeviceName = body.DeviceName
	device.DeviceSerialNumber = body.DeviceSerialNumber
	device.BuildingURL = body.BuildingURL
	device.AuthToken = body.AuthToken

	if err := bmsDB.DB.Save(&device).Error; err != nil {
		serverutils.WriteError(c, 500, "Failed to update device", err.Error())
		return
	}

	serverutils.WriteJSON(c, 200, "Device updated", DeviceResponse{
		ID:                     device.ID,
		CustomerID:             device.Site.Customer.ID,
		CustomerName:           device.Site.Customer.Name,
		SiteID:                 device.Site.ID,
		SiteName:               device.Site.Name,
		Gateway:                device.Gateway,
		Controller:             device.Controller,
		ControllerSerialNumber: device.ControllerSerialNumber,
		DeviceType:             device.DeviceType,
		DeviceName:             device.DeviceName,
		DeviceSerialNumber:     device.DeviceSerialNumber,
		BuildingURL:            device.BuildingURL,
		AuthToken:              device.AuthToken,
	})
}

// Route: DELETE /devices/:device_serial_number
// Delete a device
func DeviceDelete(c *gin.Context) {
	serialNumber := c.Param("device_serial_number")

	// Get the database instance
	bmsDB, ok := serverutils.GetDBInstance(c)
	if !ok {
		return
	}

	// Fetch and validate device
	device, err := FetchDeviceBySerialNumber(bmsDB, serialNumber)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		serverutils.WriteError(c, 404, "Device not found", "No device found with the given serial number")
		return
	}

	// Soft-delete the device
	if err := bmsDB.DB.Delete(&device).Error; err != nil {
		serverutils.WriteError(c, 500, "Failed to delete device", err.Error())
		return
	}

	serverutils.WriteJSON(c, 200, "Device deleted", nil)
}

// =====================================================================================================================

// Fetch a device by serial number
func FetchDeviceBySerialNumber(bmsDB *devicesdb.BMS_DB, serialNumber string) (*models.Device, error) {
	var device models.Device
	result := bmsDB.DB.Debug().Unscoped().Preload("Site.Customer").Where("device_serial_number = ?", serialNumber).First(&device)
	if result.Error != nil {
		return nil, result.Error
	}
	return &device, nil
}
