package models

type Product struct {
	Base
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Price       float64   `gorm:"type:decimal(10,2);not null" json:"price"`
	Stock       int       `gorm:"type:int;not null;default:0" json:"stock"`
	Image       string    `gorm:"type:varchar(255)" json:"image"`
	CategoryId  string    `gorm:"type:char(36);not null" json:"category_id"`
	Category    *Category `gorm:"foreignKey:CategoryId" json:"category,omitempty"`
}
