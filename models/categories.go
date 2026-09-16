package models

type Category struct {
	Base
	Name        string `gorm:"type:varchar(100);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Image	   string `gorm:"type:varchar(255)" json:"image"`
	Products    []Product `gorm:"foreignKey:CategoryId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"products,omitempty"`
}