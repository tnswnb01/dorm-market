package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"dormmarket/internal/auth"
	"dormmarket/internal/models"
	"dormmarket/internal/repository"
	"dormmarket/internal/service"
)

type ListingHandler struct {
	listings   *service.ListingService
	uploadsDir string
}

func NewListingHandler(listingService *service.ListingService, uploadsDir string) *ListingHandler {
	return &ListingHandler{listings: listingService, uploadsDir: uploadsDir}
}

type createListingRequest struct {
	CategoryID  string `json:"categoryId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Condition   string `json:"condition"`
	Price       int    `json:"price"`
}

func (h *ListingHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	var req createListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	listing, err := h.listings.Create(r.Context(), service.CreateListingInput{
		SellerID:    userID,
		CategoryID:  req.CategoryID,
		Title:       req.Title,
		Description: req.Description,
		Condition:   models.ListingCondition(req.Condition),
		Price:       req.Price,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusCreated, listing)
}

type updateListingRequest struct {
	CategoryID  string `json:"categoryId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Condition   string `json:"condition"`
	Price       int    `json:"price"`
}

// Update — PUT /api/listings/{id} (เจ้าของเท่านั้น)
func (h *ListingHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}
	id := r.PathValue("id")

	var req updateListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	listing, err := h.listings.Update(r.Context(), service.UpdateListingInput{
		ID:          id,
		SellerID:    userID,
		CategoryID:  req.CategoryID,
		Title:       req.Title,
		Description: req.Description,
		Condition:   models.ListingCondition(req.Condition),
		Price:       req.Price,
	})
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("ไม่พบประกาศ หรือคุณไม่ใช่เจ้าของ"))
	case errors.Is(err, service.ErrTitleRequired),
		errors.Is(err, service.ErrInvalidPrice),
		errors.Is(err, service.ErrInvalidCategory):
		writeError(w, http.StatusBadRequest, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
	default:
		writeJSON(w, http.StatusOK, listing)
	}
}

// Delete — DELETE /api/listings/{id} (เจ้าของเท่านั้น) เป็น soft delete
func (h *ListingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}
	id := r.PathValue("id")

	err := h.listings.Delete(r.Context(), id, userID)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, errors.New("ไม่พบประกาศ หรือคุณไม่ใช่เจ้าของ"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ListingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	listing, err := h.listings.Get(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, errors.New("ไม่พบประกาศนี้"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

func (h *ListingHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := models.ListingFilter{
		CategoryID: q.Get("categoryId"),
		SellerID:   q.Get("sellerId"),
		Search:     q.Get("search"),
	}
	if v := q.Get("minPrice"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.MinPrice = &n
		}
	}
	if v := q.Get("maxPrice"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.MaxPrice = &n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Offset = n
		}
	}

	listings, err := h.listings.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if listings == nil {
		listings = []models.Listing{}
	}
	writeJSON(w, http.StatusOK, listings)
}

func (h *ListingHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}
	listingID := r.PathValue("id")

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("อัปโหลดไฟล์ไม่สำเร็จ"))
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("กรุณาแนบรูปอย่างน้อย 1 รูป"))
		return
	}

	for i, fh := range files {
		src, err := fh.Open()
		if err != nil {
			writeError(w, http.StatusInternalServerError, errors.New("อ่านไฟล์ไม่สำเร็จ"))
			return
		}

		imageData, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			writeError(w, http.StatusInternalServerError, errors.New("อ่านไฟล์ไม่สำเร็จ"))
			return
		}

		ext := filepath.Ext(fh.Filename)
		if ext == "" {
			ext = ".jpg"
		}
		filename := fmt.Sprintf("%s-%s%s", listingID, randomID(), ext)
		dstPath := filepath.Join(h.uploadsDir, filename)

		if err := os.WriteFile(dstPath, imageData, 0644); err != nil {
			writeError(w, http.StatusInternalServerError, errors.New("บันทึกไฟล์ไม่สำเร็จ"))
			return
		}

		if err := h.listings.AddImage(r.Context(), listingID, userID, imageData, fh.Filename, "/uploads/"+filename, i); err != nil {
			if errors.Is(err, service.ErrNotOwner) {
				writeError(w, http.StatusForbidden, err)
				return
			}
			writeError(w, http.StatusInternalServerError, errors.New("บันทึกข้อมูลรูปไม่สำเร็จ"))
			return
		}
	}

	listing, err := h.listings.Get(r.Context(), listingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

func (h *ListingHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}
	id := r.PathValue("id")

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	err := h.listings.UpdateStatus(r.Context(), id, userID, models.ListingStatus(req.Status))
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, errors.New("ไม่พบประกาศ หรือคุณไม่ใช่เจ้าของ"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (h *ListingHandler) SuggestPrice(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	suggestion, err := h.listings.SuggestPrice(
		r.Context(), q.Get("categoryId"), models.ListingCondition(q.Get("condition")),
	)
	if errors.Is(err, service.ErrInvalidCategory) {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if suggestion == nil {
		// ข้อมูลยังน้อยเกินไปที่จะแนะนำราคา — ไม่ใช่ error แค่ไม่มีคำแนะนำให้
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"available": true, "suggestion": suggestion})
}

func randomID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// SearchByImage — POST /api/listings/search-by-image (public, ไม่ต้อง login)
// รับรูป 1 รูป คืนรายการประกาศที่มีรูปคล้ายกันที่สุด
func (h *ListingHandler) SearchByImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("อัปโหลดไฟล์ไม่สำเร็จ"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("กรุณาแนบรูปที่จะค้นหา"))
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("อ่านไฟล์ไม่สำเร็จ"))
		return
	}

	listings, err := h.listings.SearchByImage(r.Context(), imageData, header.Filename)
	switch {
	case errors.Is(err, service.ErrImageSearchDisabled):
		writeError(w, http.StatusNotImplemented, err)
	case err != nil:
		writeError(w, http.StatusInternalServerError, errors.New("ค้นหาด้วยรูปไม่สำเร็จ"))
	default:
		if listings == nil {
			listings = []models.Listing{}
		}
		writeJSON(w, http.StatusOK, listings)
	}
}
