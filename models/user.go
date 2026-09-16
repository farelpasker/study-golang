package models

type User struct {
	Base
	Name     string `gorm:"type:varchar(100);not null" json:"name"`
	Email    string `gorm:"type:varchar(100);unique;not null" json:"email"`
	Password string `gorm:"type:varchar(255);not null" json:"-"`
	Profile  string `gorm:"type:varchar(255);not null" json:"profile"`
	Phone    string `gorm:"type:varchar(20);not null" json:"phone"`
	Role     string `gorm:"type:varchar(20);not null;default:'customer'" json:"role"`
}
