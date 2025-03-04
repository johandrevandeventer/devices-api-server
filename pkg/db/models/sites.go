package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Site struct {
	gorm.Model
	ID         uuid.UUID `gorm:"type:char(36);primaryKey"`
	Name       string    `gorm:"type:char(36);uniqueIndex;not null"`
	CustomerID uuid.UUID `gorm:"type:char(36);not null"`
	Customer   Customer  `gorm:"foreignKey:CustomerID"`
}

// Hook to generate UUID before creating a record
func (s *Site) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New() // Generate new UUID
	return
}
