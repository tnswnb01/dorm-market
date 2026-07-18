package router

import (
	"log"
	"net/http"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"

	"dormmarket/internal/auth"
	"dormmarket/internal/handler"
	"dormmarket/internal/repository"
)

type Handlers struct {
	Auth        *handler.AuthHandler
	Listing     *handler.ListingHandler
	Category    *handler.CategoryHandler
	Chat        *handler.ChatHandler
	Review      *handler.ReviewHandler
	Shipment    *handler.ShipmentHandler
	Report      *handler.ReportHandler
	Ticket      *handler.TicketHandler
	Users       repository.UserRepository
	JWTSecret   string
	UploadsDir  string
	AllowOrigin string
}

func New(h Handlers) http.Handler {
	mux := http.NewServeMux()

	// --- Public routes ---
	mux.HandleFunc("POST /api/auth/register", h.Auth.Register)
	mux.HandleFunc("POST /api/auth/login", h.Auth.Login)
	mux.HandleFunc("POST /api/auth/google", h.Auth.GoogleLogin)
	mux.HandleFunc("GET /api/categories", h.Category.List)
	mux.HandleFunc("GET /api/listings", h.Listing.List)
	mux.HandleFunc("GET /api/listings/suggest-price", h.Listing.SuggestPrice)
	mux.HandleFunc("POST /api/listings/search-by-image", h.Listing.SearchByImage)
	mux.HandleFunc("GET /api/listings/{id}", h.Listing.Get)
	mux.HandleFunc("GET /api/users/{id}/reviews", h.Review.ListForUser)

	// --- Protected routes (ต้องแนบ Bearer token + บัญชีต้องไม่ถูกแบน) ---
	authMW := auth.Middleware(h.JWTSecret)
	activeMW := auth.RequireActiveUser(h.Users)
	protect := func(handlerFunc http.HandlerFunc) http.Handler {
		return authMW(activeMW(handlerFunc))
	}

	mux.Handle("GET /api/auth/me", protect(h.Auth.Me))
	mux.Handle("POST /api/listings", protect(h.Listing.Create))
	mux.Handle("PUT /api/listings/{id}", protect(h.Listing.Update))
	mux.Handle("DELETE /api/listings/{id}", protect(h.Listing.Delete))
	mux.Handle("POST /api/listings/{id}/images", protect(h.Listing.UploadImage))
	mux.Handle("PATCH /api/listings/{id}/status", protect(h.Listing.UpdateStatus))
	mux.Handle("POST /api/conversations", protect(h.Chat.StartConversation))
	mux.Handle("GET /api/conversations", protect(h.Chat.ListConversations))
	mux.Handle("GET /api/conversations/{id}", protect(h.Chat.GetDetails))
	mux.Handle("GET /api/conversations/{id}/messages", protect(h.Chat.ListMessages))

	// WebSocket auth ผ่าน query param token แทน middleware ปกติ (ดูเหตุผลใน chat_handler.go)
	mux.HandleFunc("GET /ws/conversations/{id}", h.Chat.ServeWebSocket)
	mux.Handle("POST /api/reviews", protect(h.Review.Create))
	mux.Handle("GET /api/listings/{id}/can-review", protect(h.Review.CanReview))
	mux.Handle("POST /api/conversations/{id}/shipment", protect(h.Shipment.Create))
	mux.Handle("GET /api/conversations/{id}/shipment", protect(h.Shipment.Get))
	mux.Handle("PATCH /api/conversations/{id}/shipment/status", protect(h.Shipment.UpdateStatus))

	// --- รายงาน + support ticket (ต้อง login + ไม่ถูกแบน) ---
	mux.Handle("POST /api/reports", protect(h.Report.Create))
	mux.Handle("POST /api/tickets", protect(h.Ticket.Create))
	mux.Handle("GET /api/tickets", protect(h.Ticket.ListMine))
	mux.Handle("GET /api/tickets/{id}", protect(h.Ticket.GetThread))
	mux.Handle("POST /api/tickets/{id}/messages", protect(h.Ticket.AddMessage))
	mux.Handle("PATCH /api/tickets/{id}/status", protect(h.Ticket.UpdateStatus))

	// --- Admin routes (ต้องเป็น admin เท่านั้น) ---
	adminProtect := func(handlerFunc http.HandlerFunc) http.Handler {
		return authMW(activeMW(auth.RequireAdmin(handlerFunc)))
	}
	mux.Handle("GET /api/admin/reports", adminProtect(h.Report.List))
	mux.Handle("PATCH /api/admin/reports/{id}/resolve", adminProtect(h.Report.Resolve))
	mux.Handle("GET /api/admin/tickets", adminProtect(h.Ticket.ListAll))

	// --- Static file (รูปที่อัปโหลด) ---
	fs := http.FileServer(http.Dir(h.UploadsDir))
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", fs))

	// --- API docs (Swagger UI) — สร้าง/อัปเดต spec ด้วย `swag init` ที่ backend/ ---
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	return loggingMiddleware(corsMiddleware(h.AllowOrigin, mux))
}

func corsMiddleware(allowOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}
