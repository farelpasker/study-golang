package models

type OrderItem struct {
	Base
	OrderId   string   `gorm:"type:char(36);not null" json:"order_id"`
	Order     *Order   `gorm:"foreignKey:OrderId" json:"order,omitempty"`
	ProductId string   `gorm:"type:char(36);not null" json:"product_id"`
	Product   *Product `gorm:"foreignKey:ProductId" json:"product,omitempty"`
	Quantity  int      `gorm:"type:int;not null" json:"quantity"`
	Price     float64  `gorm:"type:decimal(10,2);not null" json:"price"`
	Subtotal  float64  `gorm:"type:decimal(10,2);not null" json:"subtotal"`
}
