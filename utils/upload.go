// Package utils berisi helper untuk upload file
package utils

import (
	"fmt"            // Untuk membuat pesan error
	"io"             // Untuk io.Copy (menyalin isi file)
	"mime/multipart" // Tipe data untuk file upload (multipart.File, multipart.FileHeader)
	"os"             // Untuk operasi file system (buat folder, buat file, hapus file)
	"path/filepath"  // Untuk manipulasi path file (ambil extension, gabung path)
	"strings"        // Untuk manipulasi string (ToLower)
	"time"           // Untuk timestamp di nama file

	"github.com/google/uuid" // Untuk generate nama file unik (UUID)
)

// AllowedImageTypes adalah daftar tipe file gambar yang diizinkan
// Key: extension file (huruf kecil), Value: true
// Dipakai untuk memvalidasi apakah file yang diupload adalah gambar
var AllowedImageTypes = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

// MaxFileSize adalah batas maksimal ukuran file (2MB)
// 2 * 1024 * 1024 = 2,097,152 bytes = 2MB
const MaxFileSize = 2 * 1024 * 1024

// SaveUploadedFile menyimpan file yang diupload ke folder uploads/{subFolder}
// Parameter:
//   - file: isi file yang bisa dibaca (dari r.FormFile)
//   - header: info file (nama, ukuran, tipe MIME)
//   - subFolder: nama subfolder di dalam uploads/ (contoh: "profiles", "categories")
//
// Return:
//   - string: path relatif file yang disimpan (untuk disimpan ke database)
//   - error: pesan error kalau gagal
//
// Contoh hasil: "uploads/profiles/550e8400-e29b-41d4-a716-446655440000_1726358400.jpg"
func SaveUploadedFile(file multipart.File, header *multipart.FileHeader, subFolder string) (string, error) {
	// === Validasi ukuran file ===
	// header.Size berisi ukuran file dalam bytes
	if header.Size > MaxFileSize {
		return "", fmt.Errorf("ukuran file maksimal 2MB")
	}

	// === Validasi tipe file ===
	// filepath.Ext() mengambil extension dari nama file
	// Contoh: "foto.JPG" → ".JPG" → strings.ToLower → ".jpg"
	ext := strings.ToLower(filepath.Ext(header.Filename))
	// Cek apakah extension ada di daftar yang diizinkan
	if !AllowedImageTypes[ext] {
		return "", fmt.Errorf("tipe file tidak diizinkan, gunakan: jpg, jpeg, png, gif, webp")
	}

	// === Buat folder upload kalau belum ada ===
	// filepath.Join menggabungkan path: "uploads" + "profiles" → "uploads/profiles"
	// os.MkdirAll membuat folder beserta parent-nya (seperti mkdir -p di Linux)
	// os.ModePerm = 0777 (permission read/write/execute untuk semua user)
	uploadDir := filepath.Join("uploads", subFolder)
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("gagal membuat folder upload")
	}

	// === Generate nama file unik ===
	// Format: {UUID}_{timestamp}{extension}
	// Contoh: "550e8400-e29b-41d4-a716-446655440000_1726358400.jpg"
	// Pakai UUID + timestamp supaya tidak ada nama file yang sama (collision)
	filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, filename) // Gabung: "uploads/profiles/namafile.jpg"

	// === Buat file baru di disk ===
	// os.Create membuat file kosong di path yang ditentukan
	// dst (destination) adalah file tujuan yang akan kita isi
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan file")
	}
	defer dst.Close() // Pastikan file ditutup setelah selesai

	// === Copy isi file upload ke file tujuan ===
	// io.Copy membaca dari file (source/upload) dan menulis ke dst (destination/disk)
	// Ini yang sebenarnya "menyimpan" file ke disk
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("gagal menyimpan file")
	}

	// Return path file yang disimpan (untuk disimpan ke database)
	return filePath, nil
}

// DeleteFile menghapus file dari filesystem/disk
// Dipakai untuk cleanup:
//   - Hapus foto lama saat user upload foto baru
//   - Hapus foto yang sudah diupload kalau proses selanjutnya gagal
//
// Kalau filePath kosong (""), tidak melakukan apa-apa
func DeleteFile(filePath string) {
	if filePath != "" {
		os.Remove(filePath) // Hapus file, error diabaikan (kalau file sudah tidak ada)
	}
}
