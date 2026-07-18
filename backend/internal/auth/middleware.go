package auth

import (
	"context"
	"net/http"
	"strings"

	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

type contextKey string

const (
	userIDContextKey   contextKey = "userID"
	userRoleContextKey contextKey = "userRole"
)

// Middleware ตรวจ Authorization: Bearer <token> แล้วฝัง userID ไว้ใน context
// ถ้า token ไม่ถูกต้องหรือไม่มี จะตอบ 401 ทันที ไม่ปล่อยผ่านไป handler ถัดไป
func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":"ต้องเข้าสู่ระบบก่อน"}`, http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")

			claims, err := ParseToken(token, secret)
			if err != nil {
				http.Error(w, `{"error":"เซสชันหมดอายุ กรุณาเข้าสู่ระบบใหม่"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalMiddleware เหมือน Middleware แต่ไม่ block request ถ้าไม่มี token
// ใช้กับ endpoint ที่ทำงานได้ทั้งแบบ login และไม่ login (เช่น ดูรายการสินค้า)
func OptionalMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				token := strings.TrimPrefix(header, "Bearer ")
				if claims, err := ParseToken(token, secret); err == nil {
					ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFromContext ดึง userID ที่ middleware ฝังไว้ ใช้ใน handler
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDContextKey).(string)
	return id, ok
}

// IsAdminFromContext บอกว่า user ปัจจุบันเป็นแอดมินไหม — ต้องผ่าน RequireActiveUser มาก่อน
// เท่านั้นถึงจะมีค่านี้ใน context (route ที่ผ่านแค่ Middleware เฉยๆ จะได้ false เสมอ)
func IsAdminFromContext(ctx context.Context) bool {
	role, _ := ctx.Value(userRoleContextKey).(models.UserRole)
	return role == models.RoleAdmin
}

// RequireActiveUser เชื่อมต่อจาก Middleware อีกที (ต้องรันหลัง Middleware เสมอ เพราะต้องมี
// userID ใน context แล้ว) เช็คว่าบัญชียังไม่ถูกแบน — ทำให้การแบนมีผลทันทีแม้ token เดิมจะยัง
// ไม่หมดอายุก็ตาม (ต่างจากเช็คแค่ตอน login ซึ่งจะเปิดช่องให้ใช้งานต่อได้จนกว่า token หมดอายุ)
// เก็บ role ไว้ใน context ด้วยเลยในตัว เพื่อให้ RequireAdmin และ handler อื่นๆ (เช่น ticket
// handler ที่ต้องรู้ว่าเป็นแอดมินไหม) ไม่ต้อง query ผู้ใช้ซ้ำอีกรอบ
func RequireActiveUser(users repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"ต้องเข้าสู่ระบบก่อน"}`, http.StatusUnauthorized)
				return
			}
			user, err := users.GetByID(r.Context(), userID)
			if err != nil {
				http.Error(w, `{"error":"ไม่พบผู้ใช้"}`, http.StatusUnauthorized)
				return
			}
			if user.IsBanned {
				http.Error(w, `{"error":"บัญชีนี้ถูกระงับการใช้งาน"}`, http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), userRoleContextKey, user.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin ต้องรันหลัง RequireActiveUser เสมอ (ใช้ role ที่ฝังไว้ใน context จากตรงนั้น
// ไม่ query DB ซ้ำ) — ใช้กับ route เฉพาะแอดมินเท่านั้น เช่นหน้าจัดการ report/ticket
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdminFromContext(r.Context()) {
			http.Error(w, `{"error":"ต้องเป็นแอดมินเท่านั้น"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
