package models

type Cart struct {
	Base
	UserId    string `gorm:"type:char(36);not null" json:"user_id"`
	User      *User  `gorm:"foreignKey:UserId" json:"user,omitempty"`
	CartItems []CartItem `gorm:"foreignKey:CartId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"cart_items,omitempty"`
}