// Package controllers berisi handler untuk menangani HTTP request terkait kategori
package controllers

import (
	"net/http" // Package HTTP bawaan Go untuk handler web
	"strconv"  // Untuk konversi string ke integer (page, limit)

	"project-empat-golang/config"   // Akses koneksi database (config.DB)
	"project-empat-golang/models"   // Struct model Category untuk query database
	"project-empat-golang/requests" // Struct validasi input dari client
	"project-empat-golang/utils"    // Helper response JSON & upload file

	"gorm.io/gorm" // GORM ORM — untuk cek error gorm.ErrRecordNotFound
)

// GetAllCategories mengambil semua kategori dengan pagination dan search
// Endpoint: GET /categories?page=1&limit=10&search=keyword
// Akses: Public (tidak perlu login)
func GetAllCategories(w http.ResponseWriter, r *http.Request) {
	// === Ambil parameter pagination dari URL ===
	// r.URL.Query().Get("page") mengambil query parameter "page" dari URL
	// Contoh: /categories?page=2 → hasilnya string "2"
	// strconv.Atoi mengubah string "2" jadi integer 2
	// Tanda _ berarti kita abaikan error-nya (kalau gagal konversi, hasilnya 0)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1 // Default halaman 1 kalau tidak diisi atau kurang dari 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10 // Default 10 data per halaman, maksimal 100
	}

	// Hitung offset (berapa data yang dilewati)
	// Page 1, limit 10 → offset 0 (mulai dari data ke-1)
	// Page 2, limit 10 → offset 10 (lewati 10 data, mulai dari data ke-11)
	// Page 3, limit 10 → offset 20 (lewati 20 data, mulai dari data ke-21)
	offset := (page - 1) * limit

	// Ambil keyword pencarian dari URL
	// Contoh: /categories?search=elektronik → search = "elektronik"
	search := r.URL.Query().Get("search")

	// Siapkan variabel untuk menampung hasil query
	var categories []models.Category // Slice (array) untuk menampung data kategori
	var total int64                  // Untuk menampung total data yang cocok

	// Buat base query — memberitahu GORM bahwa kita mau query tabel categories
	// SQL: SELECT * FROM categories WHERE deleted_at IS NULL
	query := config.DB.Model(&models.Category{})

	// === Tambahkan filter search kalau ada keyword ===
	if search != "" {
		// Tambah % di depan dan belakang keyword untuk wildcard search
		// "elektronik" → "%elektronik%"
		// LIKE '%elektronik%' akan cocokkan: "Alat Elektronik", "elektronik murah", dll
		search = "%" + search + "%"

		// Tambahkan kondisi WHERE ke query
		// Tanda ? adalah placeholder — GORM akan menggantinya dengan value search secara aman
		// Ini mencegah SQL injection
		// SQL: WHERE name LIKE '%elektronik%' OR description LIKE '%elektronik%'
		query = query.Where("name LIKE ? OR description LIKE ?", search, search)
	}

	// Hitung total data yang cocok (sebelum pagination)
	// SQL: SELECT COUNT(*) FROM categories WHERE ... (kondisi search kalau ada)
	query.Count(&total)

	// Eksekusi query dengan pagination
	// .Limit(10) → SQL: LIMIT 10 (batasi jumlah data)
	// .Offset(20) → SQL: OFFSET 20 (lewati 20 data pertama)
	// .Find(&categories) → jalankan query dan simpan hasilnya ke slice categories
	// .Error → ambil error-nya (nil kalau sukses)
	if err := query.Limit(limit).Offset(offset).Find(&categories).Error; err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data kategori")
		return
	}

	// Hitung total halaman
	// Contoh: total=25, limit=10 → 25/10 = 2, sisa 25%10 = 5 → totalPages = 3
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++ // Tambah 1 halaman kalau ada sisa data
	}

	// Kirim response JSON 200 OK dengan data kategori + info pagination
	utils.Success(w, http.StatusOK, map[string]interface{}{
		"categories":  categories, // Data kategori
		"page":        page,       // Halaman saat ini
		"limit":       limit,      // Jumlah data per halaman
		"total":       total,      // Total seluruh data yang cocok
		"total_pages": totalPages, // Total halaman
	})
}

// GetCategoryByID mengambil 1 kategori berdasarkan ID, termasuk produk-produknya
// Endpoint: GET /categories/{id}
// Akses: Public
func GetCategoryByID(w http.ResponseWriter, r *http.Request) {
	// r.PathValue("id") mengambil parameter {id} dari URL path
	// Contoh: route /categories/{id}, URL /categories/abc-123 → id = "abc-123"
	// Fitur ini tersedia di Go 1.22+
	id := r.PathValue("id")
	if id == "" {
		utils.Error(w, http.StatusBadRequest, "ID kategori diperlukan")
		return
	}

	// Cari kategori di database berdasarkan ID
	// .Preload("Products") → eager loading: otomatis ambil semua produk yang punya CategoryId ini
	//   GORM akan menjalankan query terpisah: SELECT * FROM products WHERE category_id = 'abc-123'
	//   Hasilnya diisi ke field Products di struct Category
	// .First(&category, "id = ?", id) → ambil 1 data pertama dimana id cocok
	//   SQL: SELECT * FROM categories WHERE id = 'abc-123' AND deleted_at IS NULL LIMIT 1
	var category models.Category
	if err := config.DB.Preload("Products").First(&category, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Data tidak ada di database
			utils.Error(w, http.StatusNotFound, "Kategori tidak ditemukan")
			return
		}
		// Error lain (misal koneksi database mati)
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data kategori")
		return
	}

	// Kirim response dengan data kategori + produk-produknya
	utils.Success(w, http.StatusOK, map[string]interface{}{
		"category": category,
	})
}

// CreateCategory membuat kategori baru
// Endpoint: POST /categories
// Content-Type: multipart/form-data
// Field: name (wajib), description, image (file, opsional)
// Akses: Admin only (dilindungi RoleMiddleware)
func CreateCategory(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "Format request tidak valid, gunakan multipart/form-data")
		return
	}

	// Ambil field dari form data
	req := requests.CategoryRequest{
		Name:        r.FormValue("name"),        // Nama kategori (wajib)
		Description: r.FormValue("description"), // Deskripsi kategori (opsional)
	}

	// Validasi input — nama tidak boleh kosong dan maksimal 100 karakter
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// === Handle upload gambar kategori (opsional) ===
	var imageCategoryPath string
	file, header, err := r.FormFile("image") // Ambil file dari field "image"
	if err == nil {
		// File berhasil diambil (user mengupload gambar)
		defer file.Close() // Pastikan file ditutup setelah selesai

		// Simpan file ke folder uploads/categories/ dengan nama unik
		path, err := utils.SaveUploadedFile(file, header, "categories")
		if err != nil {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}

		imageCategoryPath = path // Simpan path untuk disimpan ke database
	}

	// Buat struct Category baru
	// ID dan timestamps otomatis diisi oleh GORM
	category := models.Category{
		Name:        req.Name,
		Description: req.Description,
		Image:       imageCategoryPath, // Path gambar (kosong kalau tidak upload)
	}

	// Simpan ke database
	// SQL: INSERT INTO categories (id, name, description, image, ...) VALUES (...)
	if err := config.DB.Create(&category).Error; err != nil {
		utils.DeleteFile(imageCategoryPath) // Hapus file gambar kalau gagal simpan (cleanup)
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat kategori")
		return
	}

	// Kirim response 201 Created
	utils.Success(w, http.StatusCreated, map[string]interface{}{
		"message":  "Kategori berhasil dibuat",
		"category": category,
	})
}

// UpdateCategory mengupdate kategori berdasarkan ID
// Endpoint: PUT /categories/{id}
// Content-Type: multipart/form-data
// Field: name (wajib), description, image (file, opsional)
// Akses: Admin only
func UpdateCategory(w http.ResponseWriter, r *http.Request) {
	// Ambil ID dari URL path
	id := r.PathValue("id")
	if id == "" {
		utils.Error(w, http.StatusBadRequest, "ID kategori diperlukan")
		return
	}

	// Cari data kategori yang lama di database
	// Ini penting supaya: (1) bisa cek apakah data ada, (2) punya data lama untuk dibandingkan
	var category models.Category
	if err := config.DB.First(&category, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(w, http.StatusNotFound, "Kategori tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data kategori")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "Format request tidak valid, gunakan multipart/form-data")
		return
	}

	// Ambil data baru dari form
	req := requests.CategoryRequest{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}

	// Validasi input
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// === Handle upload gambar baru (opsional) ===
	// Simpan path gambar lama untuk dihapus nanti kalau ada gambar baru
	oldImage := category.Image
	file, header, err := r.FormFile("image")
	if err == nil {
		// User mengupload gambar baru
		defer file.Close()
		path, err := utils.SaveUploadedFile(file, header, "categories")
		if err != nil {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		category.Image = path // Set gambar baru
	}
	// Kalau user tidak upload gambar baru, category.Image tetap yang lama

	// Update field dengan data baru
	category.Name = req.Name
	category.Description = req.Description

	// Simpan perubahan ke database
	// SQL: UPDATE categories SET name=..., description=..., image=..., updated_at=NOW() WHERE id=...
	if err := config.DB.Save(&category).Error; err != nil {
		// Kalau gagal simpan dan ada gambar baru, hapus gambar baru (cleanup)
		if category.Image != oldImage {
			utils.DeleteFile(category.Image)
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengupdate kategori")
		return
	}

	// Kalau sukses dan ada gambar baru, hapus gambar lama dari disk
	if category.Image != oldImage {
		utils.DeleteFile(oldImage)
	}

	// Kirim response 200 OK dengan data terbaru
	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message":  "Kategori berhasil diupdate",
		"category": category,
	})
}

// DeleteCategory menghapus kategori berdasarkan ID (soft delete)
// Endpoint: DELETE /categories/{id}
// Akses: Admin only
// Soft delete: data tidak benar-benar dihapus, hanya set deleted_at = NOW()
func DeleteCategory(w http.ResponseWriter, r *http.Request) {
	// Ambil ID dari URL path
	id := r.PathValue("id")
	if id == "" {
		utils.Error(w, http.StatusBadRequest, "ID kategori diperlukan")
		return
	}

	// Cari kategori di database
	var category models.Category
	if err := config.DB.First(&category, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(w, http.StatusNotFound, "Kategori tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data kategori")
		return
	}

	// Soft delete — GORM otomatis set deleted_at = NOW() karena model pakai gorm.DeletedAt
	// SQL: UPDATE categories SET deleted_at = '2026-09-15 ...' WHERE id = '...'
	// Data tetap ada di database tapi tidak muncul di query biasa
	if err := config.DB.Delete(&category).Error; err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus kategori")
		return
	}

	// Hapus file gambar kategori dari disk
	utils.DeleteFile(category.Image)

	// Kirim response 200 OK
	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message": "Kategori berhasil dihapus",
	})
}

