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

type ReviewHandler struct {
	reviews *service.ReviewService
}

func NewReviewHandler(reviewService *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{reviews: reviewService}
}

type createReviewRequest struct {
	ListingID string `json:"listingId"`
	Rating    int    `json:"rating"`
	Comment   string `json:"comment"`
}

// Create godoc
// @Summary		เขียนรีวิวผู้ขาย
// @Description	รีวิวได้เฉพาะประกาศที่สถานะ sold แล้วและเป็นผู้ซื้อในสนทนานั้นจริง
// @Tags			reviews
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			request	body		createReviewRequest	true	"ข้อมูลรีวิว"
// @Success		201		{object}	models.Review
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Router			/api/reviews [post]
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	var req createReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	review, err := h.reviews.Create(r.Context(), service.CreateReviewInput{
		ListingID:  req.ListingID,
		ReviewerID: userID,
		Rating:     req.Rating,
		Comment:    req.Comment,
	})
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("ไม่พบประกาศนี้"))
	case errors.Is(err, service.ErrInvalidRating),
		errors.Is(err, service.ErrListingNotSold),
		errors.Is(err, service.ErrCannotReviewSelf),
		errors.Is(err, service.ErrNotEligibleReview),
		errors.Is(err, service.ErrAlreadyReviewed):
		writeError(w, http.StatusBadRequest, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, errors.New("บันทึกรีวิวไม่สำเร็จ"))
	default:
		writeJSON(w, http.StatusCreated, review)
	}
}

// ListForUser godoc
// @Summary	รีวิวทั้งหมดของผู้ใช้คนหนึ่ง (public)
// @Tags		reviews
// @Produce	json
// @Param		id	path		string	true	"User ID"
// @Success	200	{array}		models.Review
// @Failure	500	{object}	ErrorResponse
// @Router		/api/users/{id}/reviews [get]
func (h *ReviewHandler) ListForUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	reviews, err := h.reviews.ListForUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if reviews == nil {
		reviews = []models.Review{}
	}
	writeJSON(w, http.StatusOK, reviews)
}

// CanReview godoc
// @Summary		เช็คว่ารีวิวประกาศนี้ได้หรือยัง
// @Description	ใช้โชว์/ซ่อนปุ่ม "เขียนรีวิว" ฝั่ง frontend
// @Tags			reviews
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Listing ID"
// @Success		200	{object}	map[string]bool
// @Failure		401	{object}	ErrorResponse
// @Router			/api/listings/{id}/can-review [get]
func (h *ReviewHandler) CanReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}
	listingID := r.PathValue("id")
	writeJSON(w, http.StatusOK, map[string]bool{
		"canReview": h.reviews.CanReview(r.Context(), listingID, userID),
	})
}
