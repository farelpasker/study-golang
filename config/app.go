package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort    string
	JWTSecret  string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

var AppConfig *Config

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("Peringatan: File .env tidak ditemukan, menggunakan variabel sistem")
	}

	AppConfig = &Config{
		AppPort:    getEnv("APP_PORT", "8080"),
		JWTSecret:  requireEnv("JWT_SECRET"),
		DBHost:     requireEnv("DB_HOST"),
		DBPort:     requireEnv("DB_PORT"),
		DBUser:     requireEnv("DB_USERNAME"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     requireEnv("DB_NAME"),
	}
}

func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func requireEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		log.Fatalf("FATAL: Environment variable %s wajib diisi! Periksa file .env", key)
	}
	return value
}