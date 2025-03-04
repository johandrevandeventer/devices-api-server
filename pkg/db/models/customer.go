package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Customer struct {
	gorm.Model
	ID   uuid.UUID `gorm:"type:char(36);primaryKey"`
	Name string    `gorm:"type:char(36);uniqueIndex;not null"`
}

// Hook to generate UUID before creating a record
func (c *Customer) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID = uuid.New() // Generate new UUID
	return
}
