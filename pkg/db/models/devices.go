package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Device struct {
	gorm.Model
	ID                     uuid.UUID `gorm:"type:char(255);primaryKey"`
	Gateway                string    `gorm:"type:char(255);not null"`
	Controller             string    `gorm:"type:char(255);not null"`
	ControllerSerialNumber string    `gorm:"type:char(255);not null"`
	DeviceType             string    `gorm:"type:char(255);not null"`
	DeviceSerialNumber     string    `gorm:"type:char(255);not null;unique"`
	DeviceName             string    `gorm:"type:char(255);not null"`
	BuildingURL            string    `gorm:"type:char(255);not null"`
	AuthToken              string    `gorm:"type:text;not null"`
	SiteID                 uuid.UUID `gorm:"type:char(255);not null"`
	Site                   Site      `gorm:"foreignKey:SiteID"`
}

// Hook to generate UUID before creating a record
func (d *Device) BeforeCreate(tx *gorm.DB) (err error) {
	d.ID = uuid.New() // Generate new UUID
	return
}
