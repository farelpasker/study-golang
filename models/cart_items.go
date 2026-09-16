package models

type CartItem struct {
	Base
	CartId    string   `gorm:"type:char(36);not null" json:"cart_id"`
	Cart      *Cart    `gorm:"foreignKey:CartId" json:"cart,omitempty"`
	ProductId string   `gorm:"type:char(36);not null" json:"product_id"`
	Product   *Product `gorm:"foreignKey:ProductId" json:"product,omitempty"`
	Quantity  int      `gorm:"type:int;not null;default:1" json:"quantity"`
}