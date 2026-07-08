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

type ShipmentHandler struct {
	shipments *service.ShipmentService
}

func NewShipmentHandler(shipmentService *service.ShipmentService) *ShipmentHandler {
	return &ShipmentHandler{shipments: shipmentService}
}

type createShipmentRequest struct {
	Method         string `json:"method"`
	CourierName    string `json:"courierName"`
	TrackingNumber string `json:"trackingNumber"`
}

func shipmentErrorStatus(err error) int {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrNotConversationParty), errors.Is(err, service.ErrOnlySellerCanManage):
		return http.StatusForbidden
	case errors.Is(err, service.ErrInvalidShipmentMethod),
		errors.Is(err, service.ErrCourierRequired),
		errors.Is(err, service.ErrShipmentExists),
		errors.Is(err, service.ErrInvalidShipmentStatus):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// Create godoc
// @Summary	สร้างข้อมูลการจัดส่ง (ผู้ขายเท่านั้น)
// @Tags		shipments
// @Accept		json
// @Produce	json
// @Security	BearerAuth
// @Param		id		path		string					true	"Conversation ID"
// @Param		request	body		createShipmentRequest	true	"ข้อมูลการจัดส่ง"
// @Success	201		{object}	models.Shipment
// @Failure	400		{object}	ErrorResponse
// @Failure	403		{object}	ErrorResponse
// @Failure	404		{object}	ErrorResponse
// @Router		/api/conversations/{id}/shipment [post]
func (h *ShipmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}
	conversationID := r.PathValue("id")

	var req createShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	shipment, err := h.shipments.Create(r.Context(), service.CreateShipmentInput{
		ConversationID: conversationID,
		SellerID:       userID,
		Method:         models.ShipmentMethod(req.Method),
		CourierName:    req.CourierName,
		TrackingNumber: req.TrackingNumber,
	})
	if err != nil {
		writeError(w, shipmentErrorStatus(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, shipment)
}

// Get godoc
// @Summary	ดูข้อมูลการจัดส่ง
// @Tags		shipments
// @Produce	json
// @Security	BearerAuth
// @Param		id	path		string	true	"Conversation ID"
// @Success	200	{object}	models.Shipment
// @Failure	403	{object}	ErrorResponse
// @Failure	404	{object}	ErrorResponse
// @Router		/api/conversations/{id}/shipment [get]
func (h *ShipmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}
	conversationID := r.PathValue("id")

	shipment, err := h.shipments.Get(r.Context(), conversationID, userID)
	if err != nil {
		writeError(w, shipmentErrorStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}

type updateShipmentStatusRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

// UpdateStatus godoc
// @Summary		อัปเดตสถานะการจัดส่ง (ผู้ขายเท่านั้น)
// @Description	สถานะที่รองรับ: pending, shipped, completed, cancelled
// @Tags			shipments
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string						true	"Conversation ID"
// @Param			request	body		updateShipmentStatusRequest	true	"สถานะใหม่ + หมายเหตุ"
// @Success		200		{object}	models.Shipment
// @Failure		400		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Router			/api/conversations/{id}/shipment/status [patch]
func (h *ShipmentHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}
	conversationID := r.PathValue("id")

	var req updateShipmentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	shipment, err := h.shipments.UpdateStatus(
		r.Context(), conversationID, userID, models.ShipmentStatus(req.Status), req.Note,
	)
	if err != nil {
		writeError(w, shipmentErrorStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}
