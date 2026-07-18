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

type ReportHandler struct {
	reports *service.ReportService
}

func NewReportHandler(reportService *service.ReportService) *ReportHandler {
	return &ReportHandler{reports: reportService}
}

type createReportRequest struct {
	TargetType      string `json:"targetType"`
	TargetListingID string `json:"targetListingId"`
	TargetUserID    string `json:"targetUserId"`
	Reason          string `json:"reason"`
	Description     string `json:"description"`
}

// Create godoc
// @Summary		รายงานประกาศหรือผู้ใช้
// @Description	targetType เป็น "listing" หรือ "user" — ใส่ targetListingId หรือ targetUserId ให้ตรงกัน
// @Tags			reports
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			request	body		createReportRequest	true	"ข้อมูลรายงาน"
// @Success		201		{object}	models.Report
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Router			/api/reports [post]
func (h *ReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	var req createReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	report, err := h.reports.Create(r.Context(), service.CreateReportInput{
		ReporterID:      userID,
		TargetType:      models.ReportTargetType(req.TargetType),
		TargetListingID: req.TargetListingID,
		TargetUserID:    req.TargetUserID,
		Reason:          models.ReportReason(req.Reason),
		Description:     req.Description,
	})
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("ไม่พบเป้าหมายที่จะรายงาน"))
	case errors.Is(err, service.ErrInvalidReportReason),
		errors.Is(err, service.ErrInvalidTargetType),
		errors.Is(err, service.ErrCannotReportSelf):
		writeError(w, http.StatusBadRequest, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, errors.New("ส่งรายงานไม่สำเร็จ"))
	default:
		writeJSON(w, http.StatusCreated, report)
	}
}

// List godoc
// @Summary		รายการรายงานทั้งหมด (แอดมินเท่านั้น)
// @Tags			reports
// @Produce		json
// @Security		BearerAuth
// @Param			status	query		string	false	"pending, resolved หรือ dismissed"
// @Success		200		{array}		models.Report
// @Failure		500		{object}	ErrorResponse
// @Router			/api/admin/reports [get]
func (h *ReportHandler) List(w http.ResponseWriter, r *http.Request) {
	var status *models.ReportStatus
	if v := r.URL.Query().Get("status"); v != "" {
		s := models.ReportStatus(v)
		status = &s
	}

	reports, err := h.reports.List(r.Context(), status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if reports == nil {
		reports = []models.Report{}
	}
	writeJSON(w, http.StatusOK, reports)
}

type resolveReportRequest struct {
	Action string `json:"action"`
	Note   string `json:"note"`
}

// Resolve godoc
// @Summary		ดำเนินการกับรายงาน (แอดมินเท่านั้น)
// @Description	action เป็น "none" (ปิดโดยไม่ทำอะไร), "ban_user" (แบนผู้ใช้ที่ถูกรายงาน) หรือ "remove_listing" (ลบประกาศที่ถูกรายงาน)
// @Tags			reports
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string					true	"Report ID"
// @Param			request	body		resolveReportRequest	true	"การดำเนินการ"
// @Success		200		{object}	models.Report
// @Failure		400		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Router			/api/admin/reports/{id}/resolve [patch]
func (h *ReportHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	adminID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	var req resolveReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	reportID := r.PathValue("id")
	report, err := h.reports.Resolve(r.Context(), reportID, adminID, models.ReportResolutionAction(req.Action), req.Note)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("ไม่พบรายงานนี้"))
	case errors.Is(err, service.ErrReportAlreadyResolved),
		errors.Is(err, service.ErrInvalidResolutionAction),
		errors.Is(err, service.ErrResolutionActionMismatch):
		writeError(w, http.StatusBadRequest, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, errors.New("ดำเนินการไม่สำเร็จ"))
	default:
		writeJSON(w, http.StatusOK, report)
	}
}
