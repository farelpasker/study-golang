package requests

import (
	"fmt"
	"strconv"
)

type ProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       string  `json:"price"`
	Stock       string  `json:"stock"`
	CategoryId  string  `json:"category_id"`
	Image       string  `json:"image"`
}

func (r *ProductRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("nama produk wajib diisi")
	}
	if r.Description == "" {
		return fmt.Errorf("deskripsi produk wajib diisi")
	}

	price, err := strconv.ParseFloat(r.Price, 64)
	if err != nil {
		return fmt.Errorf("harga produk tidak valid")
	}
	if price <= 0 {
		return fmt.Errorf("harga produk harus lebih besar dari 0")
	}

	stock, err := strconv.Atoi(r.Stock)
	if err != nil {
		return fmt.Errorf("stok produk tidak valid")
	}
	if stock < 0 {
		return fmt.Errorf("stok produk tidak boleh negatif")
	}

	if r.CategoryId == "" {
		return fmt.Errorf("ID kategori produk wajib diisi")
	}

	return nil
}
