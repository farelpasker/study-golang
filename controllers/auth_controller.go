// Package controllers berisi semua handler/fungsi yang menangani HTTP request
package controllers

import (
	"net/http"                         // Package HTTP bawaan Go untuk handler web
	"project-empat-golang/config"      // Akses koneksi database (config.DB)
	"project-empat-golang/middlewares" // Akses context key (UserIDKey) untuk ambil user ID dari token
	"project-empat-golang/models"      // Struct model User untuk query database
	"project-empat-golang/requests"    // Struct validasi input dari client
	"project-empat-golang/utils"       // Helper response JSON & upload file

	"encoding/json" // Untuk decode JSON dari request body (dipakai di Login)
	"time"          // Untuk set waktu expired token JWT

	"github.com/golang-jwt/jwt/v5" // Library untuk membuat & memvalidasi JWT token
	"golang.org/x/crypto/bcrypt"   // Library untuk hash & verifikasi password (bcrypt)
)

// Register menangani pendaftaran user baru
// Endpoint: POST /register
// Content-Type: multipart/form-data (karena ada upload file)
// Field: name, email, password, phone, profile (file, opsional)
func Register(w http.ResponseWriter, r *http.Request) {
	// ParseMultipartForm membaca request body sebagai multipart/form-data
	// 10 << 20 = 10 * 1MB = 10MB (batas maksimal ukuran request)
	// Ini diperlukan karena kita menerima file upload (profile image)
	var req requests.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Jalankan validasi input (cek kosong, format email, panjang password, dll)
	// Kalau ada yang tidak valid, kirim error 400 dan berhenti
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// === Hash password ===
	// bcrypt.GenerateFromPassword mengubah password teks biasa menjadi hash yang aman
	// bcrypt.DefaultCost = 10 (jumlah iterasi hashing, semakin tinggi semakin aman tapi lambat)
	// Contoh: "secret123" → "$2a$10$N9qo8uLOickgx2ZMRZoMy..."
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		// Kalau gagal hash, hapus file profile yang sudah diupload (cleanup)
		utils.Error(w, http.StatusInternalServerError, "Gagal enkripsi password")
		return
	}

	// Buat struct User baru dengan data dari request
	// ID, CreatedAt, UpdatedAt otomatis diisi oleh GORM dan hook BeforeCreate
	user := models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword), // Simpan password yang sudah di-hash (bukan plain text)
		Phone:    req.Phone,
		Role:     "customer", // Role di-hardcode "customer" — user tidak bisa daftar sebagai admin
	}

	// Simpan user baru ke database
	// config.DB.Create() menjalankan: INSERT INTO users (...) VALUES (...)
	// Kalau email sudah ada (karena unique constraint), akan error
	if err := config.DB.Create(&user).Error; err != nil {
		utils.Error(w, http.StatusBadRequest, "Email sudah digunakan atau terjadi kesalahan saat membuat pengguna")
		return
	}

	// Kirim response 201 Created dengan data user yang baru dibuat
	// Password tidak dikembalikan karena di model ada tag json:"-"
	utils.Success(w, http.StatusCreated, map[string]interface{}{
		"message": "Pengguna berhasil dibuat",
		"user": map[string]interface{}{
			"id":      user.Id,
			"name":    user.Name,
			"email":   user.Email,
			"phone":   user.Phone,
			"role":    user.Role,
		},
	})
}

// Login menangani proses login dan mengembalikan JWT token
// Endpoint: POST /login
// Content-Type: application/json
// Body: {"email": "...", "password": "..."}
func Login(w http.ResponseWriter, r *http.Request) {
	// Decode JSON body ke struct LoginRequest
	// json.NewDecoder membaca dari r.Body (isi request)
	// .Decode(&req) mengisi struct req dengan data JSON
	var req requests.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validasi input (email & password tidak boleh kosong)
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Cari user berdasarkan email di database
	// .Where("email = ?", req.Email) → filter berdasarkan email
	// .First(&user) → ambil 1 data pertama yang cocok
	// Tanda ? adalah placeholder untuk mencegah SQL injection
	var user models.User
	if err := config.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Pesan error sengaja sama ("Email atau password salah")
		// supaya hacker tidak bisa tahu apakah email terdaftar atau tidak
		utils.Error(w, http.StatusUnauthorized, "Email atau password salah")
		return
	}

	// Bandingkan password yang diinput dengan hash password di database
	// bcrypt.CompareHashAndPassword akan:
	// 1. Ambil hash dari database: "$2a$10$N9qo8uLOickgx2ZMRZoMy..."
	// 2. Hash password input dengan salt yang sama
	// 3. Bandingkan hasilnya — kalau sama berarti password benar
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		utils.Error(w, http.StatusUnauthorized, "Email atau password salah")
		return
	}

	// === Buat JWT Token ===
	// Claims adalah data yang disimpan di dalam token
	// Token ini nanti dikirim ke client dan dipakai untuk autentikasi request selanjutnya
	claims := jwt.MapClaims{
		"user_id": user.Id.String(), // ID user (diubah ke string karena UUID)
		"role":    user.Role,        // Role user (admin/customer) — dipakai di RoleMiddleware
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // Token expired dalam 72 jam (3 hari)
	}

	// Buat token baru dengan method signing HMAC SHA256
	// jwt.NewWithClaims membuat objek token, belum ditandatangani
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Tandatangani token dengan secret key dari config (.env)
	// Hasilnya adalah string token seperti: "eyJhbGciOiJIUzI1NiIs..."
	// Token ini yang dikirim ke client untuk dipakai di header Authorization
	tokenString, err := token.SignedString([]byte(config.AppConfig.JWTSecret))
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat token")
		return
	}

	// Kirim response dengan token
	// Client harus menyimpan token ini dan mengirimnya di header:
	// Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message": "Login berhasil",
		"token":   tokenString,
	})
}

// GetMe mengambil data profil user yang sedang login
// Endpoint: GET /me
// Header: Authorization: Bearer <token>
func GetMe(w http.ResponseWriter, r *http.Request) {
	// Ambil userID dari context
	// Context ini sudah diisi oleh AuthMiddleware saat memvalidasi token
	// middlewares.UserIDKey adalah key untuk mengambil user_id dari context
	// .(string) adalah type assertion — mengkonversi interface{} ke string
	userID := r.Context().Value(middlewares.UserIDKey).(string)

	// Cari user di database berdasarkan ID
	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		utils.Error(w, http.StatusNotFound, "User tidak ditemukan")
		return
	}

	// Kirim response dengan data user lengkap (tanpa password)
	utils.Success(w, http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{
			"id":         user.Id,
			"name":       user.Name,
			"email":      user.Email,
			"profile":    user.Profile,
			"phone":      user.Phone,
			"role":       user.Role,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
	})
}

// UpdateProfile mengupdate data user yang sedang login
// Endpoint: PUT /me
// Content-Type: multipart/form-data (karena ada upload file)
// Field: name, email, phone, password (opsional), profile (file, opsional)
func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Ambil userID dari context (diisi oleh AuthMiddleware dari token JWT)
	userID := r.Context().Value(middlewares.UserIDKey).(string)

	// Cari data user saat ini di database
	// Ini penting supaya kita punya data lama untuk dibandingkan
	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		utils.Error(w, http.StatusNotFound, "User tidak ditemukan")
		return
	}

	// Parse multipart form untuk membaca field dan file
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "Format request tidak valid, gunakan multipart/form-data")
		return
	}

	// Ambil data dari form
	req := requests.UpdateProfileRequest{
		Name:     r.FormValue("name"),
		Email:    r.FormValue("email"),
		Phone:    r.FormValue("phone"),
		Password: r.FormValue("password"), // Opsional — kosong berarti tidak ganti password
	}

	// Validasi input
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// === Cek duplikasi email ===
	// Kalau user ganti email, cek apakah email baru sudah dipakai user lain
	// "email = ? AND id != ?" → cari email yang sama TAPI bukan milik user ini
	if req.Email != user.Email {
		var existing models.User
		if err := config.DB.Where("email = ? AND id != ?", req.Email, userID).First(&existing).Error; err == nil {
			// err == nil artinya query berhasil menemukan user lain dengan email yang sama
			utils.Error(w, http.StatusBadRequest, "Email sudah digunakan oleh pengguna lain")
			return
		}
	}

	// === Handle upload foto profil baru (opsional) ===
	// Simpan path foto lama untuk dihapus nanti kalau ada foto baru
	oldProfile := user.Profile
	file, header, err := r.FormFile("profile")
	if err == nil {
		// User mengupload foto profil baru
		defer file.Close()
		path, err := utils.SaveUploadedFile(file, header, "profiles")
		if err != nil {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		user.Profile = path // Set path foto baru ke user
	}
	// Kalau user tidak upload foto baru, user.Profile tetap yang lama (tidak berubah)

	// Update field-field user dengan data baru dari request
	user.Name = req.Name
	user.Email = req.Email
	user.Phone = req.Phone

	// === Update password (opsional) ===
	// Hanya update password kalau user mengisi field password
	// Kalau kosong, password lama tetap dipertahankan
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			// Kalau gagal hash dan ada foto baru, hapus foto baru (cleanup)
			if user.Profile != oldProfile {
				utils.DeleteFile(user.Profile)
			}
			utils.Error(w, http.StatusInternalServerError, "Gagal enkripsi password")
			return
		}
		user.Password = string(hashedPassword) // Set password baru yang sudah di-hash
	}

	// Simpan perubahan ke database
	// config.DB.Save() menjalankan: UPDATE users SET name=..., email=..., ... WHERE id=...
	if err := config.DB.Save(&user).Error; err != nil {
		// Kalau gagal simpan dan ada foto baru, hapus foto baru (cleanup)
		if user.Profile != oldProfile {
			utils.DeleteFile(user.Profile)
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengupdate profil")
		return
	}

	// Kalau update berhasil dan ada foto baru, hapus foto profil lama dari disk
	if user.Profile != oldProfile {
		utils.DeleteFile(oldProfile)
	}

	// Kirim response dengan data user yang sudah diupdate
	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message": "Profil berhasil diupdate",
		"user": map[string]interface{}{
			"id":      user.Id,
			"name":    user.Name,
			"email":   user.Email,
			"profile": user.Profile,
			"phone":   user.Phone,
			"role":    user.Role,
		},
	})
}

// Logout menangani proses logout user
// Endpoint: POST /logout
// Header: Authorization: Bearer <token>
//
// Karena JWT bersifat stateless (token tidak disimpan di server),
// logout dilakukan di sisi client dengan menghapus token yang tersimpan.
// Endpoint ini hanya memberikan konfirmasi ke client bahwa "logout berhasil".
func Logout(w http.ResponseWriter, r *http.Request) {
	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message": "Logout berhasil, silakan hapus token di sisi client",
	})
}

