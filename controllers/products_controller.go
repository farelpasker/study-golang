package controllers

import (
	"log"
	"net/http"
	"project-empat-golang/config"
	"project-empat-golang/models"
	"project-empat-golang/requests"
	"project-empat-golang/utils"
	"strconv"

	"gorm.io/gorm"
)

func GetAllProducts(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	search := r.URL.Query().Get("search")

	var products []models.Product
	var total int64

	query := config.DB.Model(&models.Product{}).Preload("Category")

	if search != "" {
		search = "%" + search + "%"
		query = query.Where("name LIKE ", search)
	}

	query.Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&products).Error; err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data produk")
		return
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"products":    products,
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	})
}

func GetProductByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.Error(w, http.StatusBadRequest, "ID produk diperlukan")
		return
	}

	var product models.Product
	if err := config.DB.Preload("Category").First(&product, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(w, http.StatusNotFound, "Produk tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data produk")
		return
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message": "Berhasil mengambil data detail produk",
		"product": product,
	})
}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "Format request tidak valid, gunakan multipart/form-data")
		return
	}

	//Data form hanya mau string
	req := requests.ProductRequest{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Price:       r.FormValue("price"),
		Stock:       r.FormValue("stock"),
		CategoryId:  r.FormValue("category_id"),
	}

	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Cek apakah category_id valid dan benar-benar ada di database (belum di-soft delete)
	var category models.Category
	if err := config.DB.First(&category, "id = ?", req.CategoryId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(w, http.StatusBadRequest, "Kategori tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal memverifikasi kategori")
		return
	}

	var imageProductPath string
	file, handler, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		path, err := utils.SaveUploadedFile(file, handler, "products")
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan file gambar")
			return
		}
		imageProductPath = path
	}

	// Konversi Price dan Stock dari string ke tipe data yang sesuai
	price, _ := strconv.ParseFloat(req.Price, 64)
	stock, _ := strconv.Atoi(req.Stock)

	product := models.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       price,
		Stock:       stock,
		CategoryId:  req.CategoryId,
		Image:       imageProductPath,
	}

	if err := config.DB.Create(&product).Error; err != nil {
		utils.DeleteFile(imageProductPath)
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat produk")
		return
	}

	// Preload Category agar response langsung menyertakan data kategori
	config.DB.Preload("Category").First(&product, "id = ?", product.Id)

	utils.Success(w, http.StatusCreated, map[string]interface{}{
		"message": "Produk berhasil dibuat",
		"product": product,
	})
}

func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.Error(w, http.StatusBadRequest, "ID produk diperlukan")
		return
	}

	var product models.Product
	if err := config.DB.First(&product, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(w, http.StatusNotFound, "Produk tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data produk")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "Format request tidak valid, gunakan multipart/form-data")
		return
	}

	req := requests.ProductRequest{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Price:       r.FormValue("price"),
		Stock:       r.FormValue("stock"),
		CategoryId:  r.FormValue("category_id"),
	}

	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	var category models.Category
	if err := config.DB.First(&category, "id = ?", req.CategoryId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(w, http.StatusBadRequest, "Kategori tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal memverifikasi kategori")
		return
	}

	oldImage := product.Image
	file, handler, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		path, err := utils.SaveUploadedFile(file, handler, "products")
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan file gambar")
			return
		}
		product.Image = path
	}

	price, _ := strconv.ParseFloat(req.Price, 64)
	stock, _ := strconv.Atoi(req.Stock)

	product.Name = req.Name
	product.Description = req.Description
	product.Price = price
	product.Stock = stock
	product.CategoryId = req.CategoryId

	if err := config.DB.Omit("Category").Save(&product).Error; err != nil {
		log.Println("Error saat memperbarui produk:", err)
		if product.Image != oldImage {
			utils.DeleteFile(product.Image)
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui produk: "+err.Error())
		return
	}

	if product.Image != oldImage {
		utils.DeleteFile(oldImage)
	}

	// Preload Category agar response menyertakan detail kategori terbaru
	config.DB.Preload("Category").First(&product, "id = ?", product.Id)

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message": "Produk berhasil diperbarui",
		"product": product,
	})
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.Error(w, http.StatusBadRequest, "ID produk diperlukan")
		return
	}

	var product models.Product
	if err := config.DB.First(&product, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(w, http.StatusNotFound, "Produk tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data produk")
		return
	}

	if err := config.DB.Delete(&product).Error; err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus produk")
		return
	}
	
	utils.DeleteFile(product.Image)

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message": "Produk berhasil dihapus",
	})
}