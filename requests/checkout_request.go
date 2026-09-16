// Package requests berisi struct validasi untuk request fitur checkout dan pesanan
package requests

import "fmt"

// CheckoutRequest menampung payload saat user melakukan checkout
// Dikirim sebagai JSON: {"payment_method": "bank_transfer"}
type CheckoutRequest struct {
	CartId string `json:"cart_id"`
	PaymentMethod string `json:"payment_method"`
}

// Validate memvalidasi metode pembayaran yang dipilih
func (r *CheckoutRequest) Validate() error {
	if r.PaymentMethod == "" {
		return fmt.Errorf("metode pembayaran (payment_method) wajib diisi")
	}

	// Daftar metode pembayaran yang diizinkan
	validMethods := map[string]bool{
		"credit_card":   true,
		"bank_transfer": true,
		"paypal":        true,
		"cod":           true,
	}

	if !validMethods[r.PaymentMethod] {
		return fmt.Errorf("metode pembayaran tidak valid. Pilihan yang tersedia: credit_card, bank_transfer, paypal, cod")
	}

	return nil
}