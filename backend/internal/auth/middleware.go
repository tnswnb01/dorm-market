package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userIDContextKey contextKey = "userID"

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
