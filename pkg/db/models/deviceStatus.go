package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeviceStatus struct {
	gorm.Model
	ID                 uuid.UUID `gorm:"type:char(36);primaryKey"`
	DeviceSerialNumber string    `gorm:"type:char(36);not null;unique"`
	LastSeen           time.Time `gorm:"type:datetime;not null"`
	Device             Device    `gorm:"foreignKey:DeviceSerialNumber"`
}

// Hook to generate UUID before creating a record
func (ds *DeviceStatus) BeforeCreate(tx *gorm.DB) (err error) {
	ds.ID = uuid.New() // Generate new UUID
	return
}
