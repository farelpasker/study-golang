// Package controllers menangani proses bisnis HTTP termasuk proses Checkout dan Order
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
	"time"

	"gorm.io/gorm"
)

// Checkout menangani proses pengubahan keranjang belanja menjadi pesanan (Order)
// Endpoint: POST /api/checkout
// Body (JSON): {"payment_method": "bank_transfer"}
// Akses: User login (AuthMiddleware)
func Checkout(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil user ID dari JWT context
	userID := r.Context().Value(middlewares.UserIDKey).(string)

	// 2. Decode payload JSON
	var req requests.CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	// 3. Validasi payload (metode pembayaran)
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// 4. Mulai transaksi database (Database Transaction)
	// Kita wajib menggunakan transaksi agar jika salah satu proses gagal
	// (misalnya stok habis atau gagal simpan item), seluruh perubahan akan di-ROLLBACK (dibatalkan)
	tx := config.DB.Begin()
	defer func() {
		if rec := recover(); rec != nil {
			tx.Rollback()
		}
	}()

	// 5. Ambil data keranjang belanja milik user beserta data item & produknya
	var cart models.Cart
	err := tx.Where("user_id = ?", userID).
		Preload("CartItems.Product").
		First(&cart).Error

	if err != nil || len(cart.CartItems) == 0 {
		tx.Rollback()
		utils.Error(w, http.StatusBadRequest, "Keranjang belanja Anda masih kosong, silakan tambahkan produk terlebih dahulu")
		return
	}

	// 6. Validasi stok setiap produk dan hitung total harga
	var totalPrice float64 = 0
	for _, item := range cart.CartItems {
		// Pastikan data produk ada
		if item.Product == nil {
			tx.Rollback()
			utils.Error(w, http.StatusBadRequest, "Salah satu produk dalam keranjang tidak ditemukan atau sudah tidak tersedia")
			return
		}

		// Cek apakah stok mencukupi
		if item.Product.Stock < item.Quantity {
			tx.Rollback()
			utils.Error(w, http.StatusBadRequest, fmt.Sprintf("Stok untuk produk '%s' tidak mencukupi (tersedia: %d, diminta: %d)", item.Product.Name, item.Product.Stock, item.Quantity))
			return
		}

		// Tambahkan subtotal ke total harga keseluruhan
		subtotal := float64(item.Quantity) * item.Product.Price
		totalPrice += subtotal
	}

	// 7. Buat nomor pesanan unik (Order Number)
	// Format: ORD-YYYYMMDDHHMMSS-RANDOM (Contoh: ORD-20260915170530-8472)
	orderNumber := fmt.Sprintf("ORD-%s-%04d", time.Now().Format("20060102150405"), time.Now().Nanosecond()%10000)

	// 8. Buat entitas Order baru
	order := models.Order{
		UserId:        userID,
		OrderNumber:   orderNumber,
		TotalPrice:    totalPrice,
		Status:        "pending",
		PaymentMethod: req.PaymentMethod,
	}

	// Simpan header pesanan ke database (Omit relasi agar GORM tidak bingung)
	if err := tx.Omit("OrderItems", "User").Create(&order).Error; err != nil {
		tx.Rollback()
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat pesanan: "+err.Error())
		return
	}

	// 9. Simpan setiap item ke tabel order_items dan kurangi stok produk
	for _, item := range cart.CartItems {
		itemSubtotal := float64(item.Quantity) * item.Product.Price

		orderItem := models.OrderItem{
			OrderId:   order.Id.String(),
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
			Price:     item.Product.Price,
			Subtotal:  itemSubtotal,
		}

		// Simpan order item
		if err := tx.Omit("Order", "Product").Create(&orderItem).Error; err != nil {
			tx.Rollback()
			utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan detail pesanan")
			return
		}

		// Kurangi stok produk secara langsung di database
		newStock := item.Product.Stock - item.Quantity
		if err := tx.Model(&models.Product{}).Where("id = ?", item.ProductId).Update("stock", newStock).Error; err != nil {
			tx.Rollback()
			utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui sisa stok produk")
			return
		}
	}

	// 10. Kosongkan keranjang belanja user setelah checkout berhasil
	if err := tx.Where("cart_id = ?", cart.Id.String()).Delete(&models.CartItem{}).Error; err != nil {
		tx.Rollback()
		utils.Error(w, http.StatusInternalServerError, "Gagal mengosongkan keranjang belanja")
		return
	}

	// 11. Selesaikan dan simpan permanen seluruh perubahan transaksi
	if err := tx.Commit().Error; err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyelesaikan transaksi checkout")
		return
	}

	// 12. Muat relasi item dan produk untuk dikirimkan sebagai respon ke client
	config.DB.Preload("OrderItems.Product").First(&order, "id = ?", order.Id)

	utils.Success(w, http.StatusCreated, map[string]interface{}{
		"message": "Checkout berhasil, pesanan telah dibuat",
		"order":   order,
	})
}

// GetMyOrders menampilkan daftar riwayat pesanan milik user yang sedang login
// Endpoint: GET /api/orders
// Akses: User login (AuthMiddleware)
func GetMyOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middlewares.UserIDKey).(string)

	var orders []models.Order
	// Ambil daftar order user, diurutkan dari yang terbaru
	if err := config.DB.Where("user_id = ?", userID).
		Preload("OrderItems.Product").
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar pesanan")
		return
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"total":  len(orders),
		"orders": orders,
	})
}

// GetOrderByID menampilkan detail satu pesanan berdasarkan ID
// Endpoint: GET /api/orders/{id}
// Akses: User login (AuthMiddleware)
func GetOrderByID(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middlewares.UserIDKey).(string)
	userRole, _ := r.Context().Value(middlewares.UserRoleKey).(string)
	orderID := r.PathValue("id")

	if orderID == "" {
		utils.Error(w, http.StatusBadRequest, "ID pesanan diperlukan")
		return
	}

	var order models.Order
	query := config.DB.Preload("OrderItems.Product").Preload("User")

	// Jika bukan admin, pastikan user hanya bisa melihat pesanannya sendiri
	if userRole != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&order, "id = ?", orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(w, http.StatusNotFound, "Pesanan tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil detail pesanan")
		return
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"order": order,
	})
}