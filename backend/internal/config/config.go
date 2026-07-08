package config

import (
	"os"
)

// Config รวมค่าตั้งค่าทั้งหมดที่แอปต้องใช้ อ่านจาก environment variable
// ให้ default ที่ใช้งานได้ทันทีตอน dev โดยไม่ต้องตั้งอะไรเพิ่ม
type Config struct {
	Port          string
	DatabaseURL   string
	JWTSecret     string
	UploadsDir    string
	AllowedOrigin string
	GoogleClientID string // ว่างได้ถ้าไม่เปิดใช้ Google Login — ดู README เรื่องวิธีตั้งค่า
	MLServiceURL   string // ว่างได้ถ้าไม่เปิดใช้ image similarity search — เช่น http://localhost:8001
}

func Load() Config {
	return Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://dormmarket:dormmarket@localhost:5432/dormmarket?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-only-secret-change-in-production"),
		UploadsDir:     getEnv("UPLOADS_DIR", "data/uploads"),
		AllowedOrigin:  getEnv("ALLOWED_ORIGIN", "*"),
		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),
		MLServiceURL:   getEnv("ML_SERVICE_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
