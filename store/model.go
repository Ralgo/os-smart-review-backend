package store

import (
	"gorm.io/gorm"
)

type Product struct {
	gorm.Model
	ExternalID string `gorm:"unique, uniqueindex"`
	ShopID     string
	Reviews    []Review
	Keywords   string
}

type Review struct {
	gorm.Model
	ProductID   uint `gorm:"index"`
	Author      string
	Title       string
	Content     string
	Rating      int
	IAGenerated bool
}
