// Package requests berisi struct validasi input untuk fitur keranjang belanja
package requests

import "fmt"

// AddToCartRequest menampung payload untuk menambahkan produk ke dalam keranjang
// Dikirim sebagai JSON: {"product_id": "...", "quantity": 1}
type AddToCartRequest struct {
	ProductId string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// Validate memvalidasi data saat menambahkan produk ke keranjang
func (r *AddToCartRequest) Validate() error {
	if r.ProductId == "" {
		return fmt.Errorf("ID produk (product_id) wajib diisi")
	}
	if r.Quantity <= 0 {
		return fmt.Errorf("jumlah produk (quantity) minimal 1")
	}
	return nil
}

// UpdateCartItemRequest menampung payload untuk memperbarui jumlah item di keranjang
// Dikirim sebagai JSON: {"quantity": 2}
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity"`
}

// Validate memvalidasi data saat memperbarui jumlah item di keranjang
func (r *UpdateCartItemRequest) Validate() error {
	if r.Quantity <= 0 {
		return fmt.Errorf("jumlah produk (quantity) minimal 1")
	}
	return nil
}

