package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"dormmarket/internal/auth"
	"dormmarket/internal/models"
	"dormmarket/internal/repository"
	"dormmarket/internal/service"
)

type TicketHandler struct {
	tickets *service.TicketService
}

func NewTicketHandler(ticketService *service.TicketService) *TicketHandler {
	return &TicketHandler{tickets: ticketService}
}

type createTicketRequest struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// Create godoc
// @Summary		เปิด support ticket ใหม่
// @Tags			tickets
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			request	body		createTicketRequest	true	"หัวข้อ + ข้อความแรก"
// @Success		201		{object}	models.SupportTicket
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Router			/api/tickets [post]
func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	var req createTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	ticket, err := h.tickets.CreateTicket(r.Context(), service.CreateTicketInput{
		UserID: userID, Subject: req.Subject, Message: req.Message,
	})
	switch {
	case errors.Is(err, service.ErrTicketSubjectRequired), errors.Is(err, service.ErrTicketMessageRequired):
		writeError(w, http.StatusBadRequest, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, errors.New("เปิด ticket ไม่สำเร็จ"))
	default:
		writeJSON(w, http.StatusCreated, ticket)
	}
}

// ListMine godoc
// @Summary		Ticket ทั้งหมดของฉัน
// @Tags			tickets
// @Produce		json
// @Security		BearerAuth
// @Success		200	{array}		models.SupportTicket
// @Failure		401	{object}	ErrorResponse
// @Router			/api/tickets [get]
func (h *TicketHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	tickets, err := h.tickets.ListMine(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if tickets == nil {
		tickets = []models.SupportTicket{}
	}
	writeJSON(w, http.StatusOK, tickets)
}

// ListAll godoc
// @Summary		Ticket ทั้งหมด (แอดมินเท่านั้น)
// @Tags			tickets
// @Produce		json
// @Security		BearerAuth
// @Param			status	query		string	false	"open, pending หรือ closed"
// @Success		200		{array}		models.SupportTicket
// @Failure		500		{object}	ErrorResponse
// @Router			/api/admin/tickets [get]
func (h *TicketHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	var status *models.TicketStatus
	if v := r.URL.Query().Get("status"); v != "" {
		s := models.TicketStatus(v)
		status = &s
	}

	tickets, err := h.tickets.ListAll(r.Context(), status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if tickets == nil {
		tickets = []models.SupportTicket{}
	}
	writeJSON(w, http.StatusOK, tickets)
}

type ticketThreadResponse struct {
	Ticket   *models.SupportTicket  `json:"ticket"`
	Messages []models.TicketMessage `json:"messages"`
}

// GetThread godoc
// @Summary		รายละเอียด ticket + ข้อความทั้งหมด
// @Description	เข้าถึงได้เฉพาะเจ้าของ ticket หรือแอดมิน
// @Tags			tickets
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Ticket ID"
// @Success		200	{object}	ticketThreadResponse
// @Failure		403	{object}	ErrorResponse
// @Failure		404	{object}	ErrorResponse
// @Router			/api/tickets/{id} [get]
func (h *TicketHandler) GetThread(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	ticket, messages, err := h.tickets.GetThread(r.Context(), r.PathValue("id"), userID, auth.IsAdminFromContext(r.Context()))
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("ไม่พบ ticket นี้"))
	case errors.Is(err, service.ErrTicketAccessDenied):
		writeError(w, http.StatusForbidden, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
	default:
		if messages == nil {
			messages = []models.TicketMessage{}
		}
		writeJSON(w, http.StatusOK, ticketThreadResponse{Ticket: ticket, Messages: messages})
	}
}

type addTicketMessageRequest struct {
	Body string `json:"body"`
}

// AddMessage godoc
// @Summary		ตอบกลับใน ticket
// @Description	เข้าถึงได้เฉพาะเจ้าของ ticket หรือแอดมิน — ถ้าแอดมินตอบ ticket จะเปลี่ยนเป็น pending ถ้าเจ้าของ ticket ทัก จะ reopen เป็น open
// @Tags			tickets
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string					true	"Ticket ID"
// @Param			request	body		addTicketMessageRequest	true	"ข้อความ"
// @Success		201		{object}	models.TicketMessage
// @Failure		400		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Router			/api/tickets/{id}/messages [post]
func (h *TicketHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	var req addTicketMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	msg, err := h.tickets.AddMessage(r.Context(), r.PathValue("id"), userID, auth.IsAdminFromContext(r.Context()), req.Body)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("ไม่พบ ticket นี้"))
	case errors.Is(err, service.ErrTicketAccessDenied):
		writeError(w, http.StatusForbidden, err)
	case errors.Is(err, service.ErrTicketMessageRequired):
		writeError(w, http.StatusBadRequest, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, errors.New("ส่งข้อความไม่สำเร็จ"))
	default:
		writeJSON(w, http.StatusCreated, msg)
	}
}

type updateTicketStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus godoc
// @Summary		เปลี่ยนสถานะ ticket
// @Description	แอดมินตั้งสถานะอะไรก็ได้ เจ้าของ ticket ปิดเรื่องเองได้อย่างเดียว
// @Tags			tickets
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string						true	"Ticket ID"
// @Param			request	body		updateTicketStatusRequest	true	"สถานะใหม่"
// @Success		204
// @Failure		400	{object}	ErrorResponse
// @Failure		403	{object}	ErrorResponse
// @Failure		404	{object}	ErrorResponse
// @Router			/api/tickets/{id}/status [patch]
func (h *TicketHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	var req updateTicketStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	err := h.tickets.UpdateStatus(r.Context(), r.PathValue("id"), userID, auth.IsAdminFromContext(r.Context()), models.TicketStatus(req.Status))
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("ไม่พบ ticket นี้"))
	case errors.Is(err, service.ErrTicketAccessDenied):
		writeError(w, http.StatusForbidden, err)
	case errors.Is(err, service.ErrInvalidTicketStatus):
		writeError(w, http.StatusBadRequest, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, errors.New("เปลี่ยนสถานะไม่สำเร็จ"))
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
