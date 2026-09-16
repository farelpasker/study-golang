// Package controllers berisi semua handler HTTP termasuk fitur keranjang belanja (Cart)
package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"project-empat-golang/config"
	"project-empat-golang/middlewares"
	"project-empat-golang/models"
	"project-empat-golang/requests"
	"project-empat-golang/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getOrCreateUserCart mencari cart milik user, atau membuat cart baru jika belum ada
func getOrCreateUserCart(userID string) (*models.Cart, error) {
	var cart models.Cart
	err := config.DB.Where("user_id = ?", userID).First(&cart).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Buat keranjang baru jika user belum memiliki keranjang
			cart = models.Cart{
				UserId: userID,
			}
			if err := config.DB.Create(&cart).Error; err != nil {
				return nil, err
			}
			return &cart, nil
		}
		return nil, err
	}
	return &cart, nil
}

// GetCart mengambil isi keranjang belanja user yang sedang login
// Endpoint: GET /api/cart
// Akses: User login (AuthMiddleware)
func GetCart(w http.ResponseWriter, r *http.Request) {
	// Ambil ID user dari JWT token context
	userID := r.Context().Value(middlewares.UserIDKey).(string)

	var cart models.Cart
	// Cari cart milik user beserta relasi items dan produk di dalamnya
	err := config.DB.Where("user_id = ?", userID).
		Preload("CartItems.Product.Category").
		First(&cart).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Jika belum memiliki cart sama sekali, kembalikan cart kosong
			utils.Success(w, http.StatusOK, map[string]interface{}{
				"cart_id":     nil,
				"items":       []interface{}{},
				"total_items": 0,
				"total_price": 0,
			})
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data keranjang")
		return
	}

	// Struct custom response untuk menampilkan item dengan subtotal per item
	type CartItemDetail struct {
		Id        uuid.UUID       `json:"id"`
		ProductId string          `json:"product_id"`
		Product   *models.Product `json:"product"`
		Quantity  int             `json:"quantity"`
		Subtotal  float64         `json:"subtotal"`
	}

	var itemsResponse []CartItemDetail
	var totalItems int = 0 //0,4,
	var totalPrice float64 = 0 //20.000

	// Hitung subtotal tiap item dan total keseluruhan
	//sama dengan foreach, iterasi setiap item di cart.CartItems
	for _, item := range cart.CartItems {
		var subtotal float64 = 0
		if item.Product != nil {
			subtotal = float64(item.Quantity) * item.Product.Price
		}
		totalItems += item.Quantity
		totalPrice += subtotal

		itemsResponse = append(itemsResponse, CartItemDetail{
			Id:        item.Id,
			ProductId: item.ProductId,
			Product:   item.Product,
			Quantity:  item.Quantity,
			Subtotal:  subtotal,
		})
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"cart_id":     cart.Id,
		"items":       itemsResponse,
		"total_items": totalItems,
		"total_price": totalPrice,
	})
}

// AddToCart menambahkan produk ke dalam keranjang
// Endpoint: POST /api/cart/items
// Body (JSON): {"product_id": "uuid", "quantity": 1}
// Akses: User login (AuthMiddleware)
func AddToCart(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middlewares.UserIDKey).(string)

	// Decode request body JSON
	var req requests.AddToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	// Validasi input
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// 1. Cek apakah produk benar-benar ada di database
	var product models.Product
	if err := config.DB.First(&product, "id = ?", req.ProductId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(w, http.StatusBadRequest, "Produk tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa produk")
		return
	}

	// 2. Cek apakah stok produk mencukupi
	if product.Stock < req.Quantity {
		utils.Error(w, http.StatusBadRequest, fmt.Sprintf("Stok produk tidak mencukupi, sisa stok: %d", product.Stock))
		return
	}

	// 3. Ambil atau buat cart untuk user ini
	cart, err := getOrCreateUserCart(userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengakses keranjang")
		return
	}

	// 4. Cek apakah produk ini sudah pernah dimasukkan ke dalam keranjang sebelumnya
	var existingItem models.CartItem
	err = config.DB.Where("cart_id = ? AND product_id = ?", cart.Id.String(), req.ProductId).First(&existingItem).Error

	if err == nil {
		// Jika produk sudah ada di keranjang, tambahkan jumlahnya
		newQuantity := existingItem.Quantity + req.Quantity

		// Pastikan total kuantitas baru tidak melebihi stok yang tersedia
		if product.Stock < newQuantity {
			utils.Error(w, http.StatusBadRequest, fmt.Sprintf("Total kuantitas di keranjang (%d) melebihi sisa stok (%d)", newQuantity, product.Stock))
			return
		}

		existingItem.Quantity = newQuantity
		if err := config.DB.Save(&existingItem).Error; err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui item di keranjang")
			return
		}

		utils.Success(w, http.StatusOK, map[string]interface{}{
			"message": "Kuantitas produk di keranjang berhasil ditambahkan",
			"item":    existingItem,
		})
		return
	} else if err == gorm.ErrRecordNotFound {
		// Jika produk belum ada di keranjang, buat item baru
		newItem := models.CartItem{
			CartId:    cart.Id.String(),
			ProductId: req.ProductId,
			Quantity:  req.Quantity,
		}

		if err := config.DB.Create(&newItem).Error; err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal menambahkan produk ke keranjang")
			return
		}

		// Preload data produk untuk respon
		config.DB.Preload("Product").First(&newItem, "id = ?", newItem.Id)

		utils.Success(w, http.StatusCreated, map[string]interface{}{
			"message": "Produk berhasil ditambahkan ke keranjang",
			"item":    newItem,
		})
		return
	} else {
		utils.Error(w, http.StatusInternalServerError, "Gagal memproses keranjang")
		return
	}
}

// UpdateCartItem mengubah jumlah/kuantitas item yang ada di keranjang
// Endpoint: PUT /api/cart/items/{id}
// Body (JSON): {"quantity": 2}
// Akses: User login (AuthMiddleware)
func UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middlewares.UserIDKey).(string)
	itemID := r.PathValue("id")
	if itemID == "" {
		utils.Error(w, http.StatusBadRequest, "ID item keranjang diperlukan")
		return
	}

	var req requests.UpdateCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// 1. Cari cart milik user yang sedang login
	var cart models.Cart
	if err := config.DB.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		utils.Error(w, http.StatusNotFound, "Keranjang tidak ditemukan")
		return
	}

	// 2. Cari item berdasarkan ID dan pastikan item tersebut memang milik cart user ini (aspek keamanan)
	var cartItem models.CartItem
	if err := config.DB.Where("id = ? AND cart_id = ?", itemID, cart.Id.String()).Preload("Product").First(&cartItem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(w, http.StatusNotFound, "Item keranjang tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil item keranjang")
		return
	}

	// 3. Cek ketersediaan stok produk
	if cartItem.Product != nil && cartItem.Product.Stock < req.Quantity {
		utils.Error(w, http.StatusBadRequest, fmt.Sprintf("Stok tidak mencukupi, sisa stok: %d", cartItem.Product.Stock))
		return
	}

	// 4. Update kuantitas
	cartItem.Quantity = req.Quantity
	if err := config.DB.Save(&cartItem).Error; err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui jumlah item")
		return
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message": "Jumlah item keranjang berhasil diperbarui",
		"item":    cartItem,
	})
}

// DeleteCartItem menghapus 1 item tertentu dari keranjang belanja
// Endpoint: DELETE /api/cart/items/{id}
// Akses: User login (AuthMiddleware)
func DeleteCartItem(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middlewares.UserIDKey).(string)
	itemID := r.PathValue("id")
	if itemID == "" {
		utils.Error(w, http.StatusBadRequest, "ID item keranjang diperlukan")
		return
	}

	// 1. Cari cart milik user
	var cart models.Cart
	if err := config.DB.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		utils.Error(w, http.StatusNotFound, "Keranjang tidak ditemukan")
		return
	}

	// 2. Cari item di keranjang user
	var cartItem models.CartItem
	if err := config.DB.Where("id = ? AND cart_id = ?", itemID, cart.Id.String()).First(&cartItem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(w, http.StatusNotFound, "Item keranjang tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil item keranjang")
		return
	}

	// 3. Hapus item
	if err := config.DB.Delete(&cartItem).Error; err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus item dari keranjang")
		return
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message": "Item berhasil dihapus dari keranjang",
	})
}

// ClearCart mengosongkan seluruh item yang ada di keranjang belanja
// Endpoint: DELETE /api/cart
// Akses: User login (AuthMiddleware)
func ClearCart(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middlewares.UserIDKey).(string)

	var cart models.Cart
	if err := config.DB.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Success(w, http.StatusOK, map[string]interface{}{
				"message": "Keranjang sudah kosong",
			})
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengakses keranjang")
		return
	}

	// Hapus semua item yang memiliki cart_id ini
	if err := config.DB.Where("cart_id = ?", cart.Id.String()).Delete(&models.CartItem{}).Error; err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengosongkan keranjang")
		return
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message": "Keranjang belanja berhasil dikosongkan",
	})
}