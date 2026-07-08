package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"dormmarket/internal/auth"
	"dormmarket/internal/models"
	"dormmarket/internal/repository"
	"dormmarket/internal/service"
	"dormmarket/internal/ws"
)

type ChatHandler struct {
	chat      *service.ChatService
	hub       *ws.Hub
	jwtSecret string
}

func NewChatHandler(chatService *service.ChatService, hub *ws.Hub, jwtSecret string) *ChatHandler {
	return &ChatHandler{chat: chatService, hub: hub, jwtSecret: jwtSecret}
}

type startConversationRequest struct {
	ListingID string `json:"listingId"`
}

// StartConversation godoc
// @Summary		เริ่มบทสนทนากับผู้ขาย
// @Description	เรียกตอนกดปุ่ม "ติดต่อผู้ขาย" — idempotent เรียกซ้ำกี่ครั้งก็ได้ห้องเดิม
// @Tags			chat
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			request	body		startConversationRequest	true	"รหัสประกาศที่จะเริ่มคุย"
// @Success		200		{object}	models.Conversation
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Router			/api/conversations [post]
func (h *ChatHandler) StartConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	var req startConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	conv, err := h.chat.StartConversation(r.Context(), req.ListingID, userID)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("ไม่พบประกาศนี้"))
	case errors.Is(err, service.ErrCannotMessageSelf):
		writeError(w, http.StatusBadRequest, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
	default:
		writeJSON(w, http.StatusOK, conv)
	}
}

// ListConversations godoc
// @Summary	รายการบทสนทนาของฉัน (inbox)
// @Tags		chat
// @Produce	json
// @Security	BearerAuth
// @Success	200	{array}		models.Conversation
// @Failure	401	{object}	ErrorResponse
// @Router		/api/conversations [get]
func (h *ChatHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	conversations, err := h.chat.ListConversations(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if conversations == nil {
		conversations = []models.Conversation{}
	}
	writeJSON(w, http.StatusOK, conversations)
}

// GetDetails godoc
// @Summary		รายละเอียดบทสนทนา
// @Description	คืนข้อมูล conversation พร้อม listing แนบมาด้วย ใช้ตอนเปิดหน้าแชท
// @Tags			chat
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Conversation ID"
// @Success		200	{object}	models.Conversation
// @Failure		401	{object}	ErrorResponse
// @Failure		403	{object}	ErrorResponse
// @Failure		404	{object}	ErrorResponse
// @Router			/api/conversations/{id} [get]
func (h *ChatHandler) GetDetails(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}
	conversationID := r.PathValue("id")

	conv, err := h.chat.GetConversationDetails(r.Context(), conversationID, userID)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("ไม่พบการสนทนานี้"))
	case errors.Is(err, service.ErrNotConversationParty):
		writeError(w, http.StatusForbidden, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
	default:
		writeJSON(w, http.StatusOK, conv)
	}
}

// ListMessages godoc
// @Summary	รายการข้อความในบทสนทนา
// @Tags		chat
// @Produce	json
// @Security	BearerAuth
// @Param		id	path		string	true	"Conversation ID"
// @Success	200	{array}		models.Message
// @Failure	401	{object}	ErrorResponse
// @Failure	403	{object}	ErrorResponse
// @Failure	404	{object}	ErrorResponse
// @Router		/api/conversations/{id}/messages [get]
func (h *ChatHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}
	conversationID := r.PathValue("id")

	messages, err := h.chat.ListMessages(r.Context(), conversationID, userID)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("ไม่พบการสนทนานี้"))
	case errors.Is(err, service.ErrNotConversationParty):
		writeError(w, http.StatusForbidden, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
	default:
		if messages == nil {
			messages = []models.Message{}
		}
		writeJSON(w, http.StatusOK, messages)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// อนุญาตทุก origin เพื่อความง่ายตอน dev — ก่อน deploy จริงควรจำกัดเฉพาะ origin ของ frontend
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWebSocket — GET /ws/conversations/{id}?token=... (ต้อง login + เป็นคู่สนทนา)
//
// ใช้ query param แทน Authorization header เพราะ browser WebSocket API มาตรฐาน
// ตั้งค่า custom header ตอน handshake ไม่ได้ — เป็นข้อจำกัดของ WebSocket spec เอง ไม่ใช่ของเรา
func (h *ChatHandler) ServeWebSocket(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("id")
	token := r.URL.Query().Get("token")

	claims, err := auth.ParseToken(token, h.jwtSecret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	if !h.chat.CanAccessConversation(r.Context(), conversationID, userID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade ล้มเหลว: %v", err)
		return
	}

	h.hub.ServeClient(conn, conversationID, userID, func(content string) {
		msg, err := h.chat.SendMessage(r.Context(), conversationID, userID, content)
		if err != nil {
			// ข้อความไม่ผ่าน validation (ว่างเปล่า/ยาวเกิน) — แค่ไม่ broadcast ไม่ต้อง kick client ออก
			log.Printf("ws: ส่งข้อความไม่สำเร็จ (user %s): %v", userID, err)
			return
		}
		h.hub.Broadcast(conversationID, msg)
	})
}
