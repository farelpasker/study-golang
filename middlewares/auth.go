// Package middlewares berisi middleware untuk autentikasi dan otorisasi
// Middleware adalah fungsi yang dijalankan SEBELUM handler utama
// Dipakai untuk: validasi token JWT, cek role user, dll
package middlewares

import (
	"context"                     // Untuk menyimpan data (user_id, role) ke context request
	"fmt"                         // Untuk membuat pesan error
	"net/http"                    // Package HTTP untuk handler
	"project-empat-golang/config" // Akses konfigurasi (JWT secret)
	"project-empat-golang/utils"  // Helper response JSON
	"strings"                     // Untuk manipulasi string (cek prefix "Bearer ", trim prefix)

	"github.com/golang-jwt/jwt/v5" // Library JWT untuk parse & validasi token
)

// contextKey adalah tipe custom untuk key di context
// Pakai tipe custom supaya tidak bentrok dengan key dari package lain
type contextKey string

// UserIDKey adalah key untuk menyimpan/mengambil user_id dari context
// Dipakai di controller: r.Context().Value(middlewares.UserIDKey)
const UserIDKey contextKey = "userID"

// UserRoleKey adalah key untuk menyimpan/mengambil role dari context
// Dipakai di RoleMiddleware untuk cek izin akses
const UserRoleKey contextKey = "userRole"

// AuthMiddleware memvalidasi JWT token dari header Authorization
// Alur kerja:
//  1. Ambil header "Authorization" dari request
//  2. Cek apakah format-nya "Bearer <token>"
//  3. Parse & validasi token menggunakan JWT secret
//  4. Ambil user_id dan role dari claims token
//  5. Simpan ke context request, lanjutkan ke handler berikutnya
//
// Penggunaan di routes:
//
//	mux.HandleFunc("GET /me", middlewares.AuthMiddleware(controllers.GetMe))
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	// Return fungsi baru yang membungkus handler asli (next)
	// Fungsi ini yang akan dijalankan saat ada request masuk
	return func(w http.ResponseWriter, r *http.Request) {
		// === Step 1: Ambil header Authorization ===
		// Header harus format: "Authorization: Bearer eyJhbGciOiJ..."
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// Tidak ada header atau format salah
			utils.Error(w, http.StatusUnauthorized, "Header Authorization Bearer diperlukan")
			return // Berhenti di sini, handler berikutnya TIDAK dijalankan
		}

		// === Step 2: Ambil token string ===
		// Hapus prefix "Bearer " dari header, sisanya adalah token
		// "Bearer eyJhbGciOiJ..." → "eyJhbGciOiJ..."
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// === Step 3: Parse & validasi token ===
		// jwt.MapClaims adalah map untuk menampung data dari token
		claims := jwt.MapClaims{}

		// jwt.ParseWithClaims akan:
		// 1. Decode token (base64)
		// 2. Verifikasi signature menggunakan secret key
		// 3. Cek apakah token belum expired
		// 4. Isi claims dengan data dari token
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			// Callback ini dipanggil untuk memberikan secret key
			// Sebelumnya, cek apakah method signing-nya HMAC (HS256)
			// Ini mencegah serangan "algorithm switching"
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Method signing tidak valid")
			}
			// Return secret key yang sama dengan yang dipakai saat membuat token
			return []byte(config.AppConfig.JWTSecret), nil
		})

		// Cek apakah parsing berhasil dan token valid
		if err != nil || !token.Valid {
			utils.Error(w, http.StatusUnauthorized, "Token tidak valid")
			return
		}

		// === Step 4: Ambil data dari claims ===
		// claims["user_id"] mengambil user_id yang disimpan saat login
		// .(string) adalah type assertion — konversi interface{} ke string
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			utils.Error(w, http.StatusUnauthorized, "Token tidak valid")
			return
		}

		// Ambil role dari claims (bisa kosong kalau tidak ada)
		role, _ := claims["role"].(string)

		// === Step 5: Simpan ke context dan lanjutkan ===
		// context.WithValue menyimpan data ke context request
		// Data ini bisa diambil di handler berikutnya dengan:
		//   r.Context().Value(middlewares.UserIDKey).(string)
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, UserRoleKey, role)

		// Lanjutkan ke handler berikutnya (next) dengan context yang sudah diisi
		// r.WithContext(ctx) membuat request baru dengan context yang diupdate
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RoleMiddleware membatasi akses berdasarkan role user
// Ini adalah kombinasi AuthMiddleware + pengecekan role
// Hanya user dengan role yang diizinkan yang bisa akses
//
// Parameter:
//   - next: handler yang akan dijalankan kalau role cocok
//   - allowedRoles: daftar role yang diizinkan (bisa lebih dari 1)
//
// Penggunaan di routes:
//
//	mux.HandleFunc("POST /categories", middlewares.RoleMiddleware(handler, "admin"))
//	mux.HandleFunc("GET /orders", middlewares.RoleMiddleware(handler, "admin", "customer"))
func RoleMiddleware(next http.HandlerFunc, allowedRoles ...string) http.HandlerFunc {
	// Bungkus dengan AuthMiddleware dulu (validasi token)
	// Setelah token valid, baru cek role
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Ambil role dari context (sudah diisi oleh AuthMiddleware)
		role, ok := r.Context().Value(UserRoleKey).(string)
		if !ok || role == "" {
			utils.Error(w, http.StatusForbidden, "Akses ditolak: role tidak ditemukan")
			return
		}

		// Cek apakah role user ada di daftar yang diizinkan
		// Loop semua allowedRoles, kalau ada yang cocok → lanjutkan
		for _, allowed := range allowedRoles {
			if role == allowed {
				next.ServeHTTP(w, r) // Role cocok, jalankan handler
				return
			}
		}

		// Kalau sampai sini, berarti role tidak ada di daftar yang diizinkan
		utils.Error(w, http.StatusForbidden, "Akses ditolak: Anda tidak memiliki izin untuk mengakses resource ini")
	})
}
