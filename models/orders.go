package models

type Order struct {
	Base
	UserId        string      `gorm:"type:char(36);not null" json:"user_id"`
	User          *User       `gorm:"foreignKey:UserId" json:"user,omitempty"`
	OrderNumber   string      `gorm:"type:varchar(100);not null;unique" json:"order_number"`
	TotalPrice    float64     `gorm:"type:decimal(10,2);not null;default:0" json:"total_price"`
	Status        string      `gorm:"type:varchar(50);not null;default:'pending'" json:"status"`
	PaymentMethod string      `gorm:"type:varchar(50);not null;default:'credit_card'" json:"payment_method"`
	OrderItems    []OrderItem `gorm:"foreignKey:OrderId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"order_items,omitempty"`
}
