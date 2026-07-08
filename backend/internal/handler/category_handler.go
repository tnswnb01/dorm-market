package handler

import (
	"net/http"

	"dormmarket/internal/service"
)

type CategoryHandler struct {
	listings *service.ListingService
}

func NewCategoryHandler(listingService *service.ListingService) *CategoryHandler {
	return &CategoryHandler{listings: listingService}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.listings.ListCategories(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, categories)
}
