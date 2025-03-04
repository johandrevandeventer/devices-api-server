package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthToken struct {
	gorm.Model
	ID         uuid.UUID `gorm:"type:char(36);primaryKey"`
	CustomerID uuid.UUID `gorm:"type:char(36);not null"`
	Customer   Customer  `gorm:"foreignKey:CustomerID"`
	Action     string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_customer_action"`
	Token      string    `gorm:"type:text;not null"`
}

// Hook to generate UUID before creating a record
func (a *AuthToken) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = uuid.New() // Generate new UUID
	return
}
