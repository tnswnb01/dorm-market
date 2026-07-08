package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"dormmarket/internal/auth"
	"dormmarket/internal/repository"
	"dormmarket/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: authService}
}

type registerRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	Name         string `json:"name"`
	DormBuilding string `json:"dormBuilding"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	token, user, err := h.auth.Register(r.Context(), service.RegisterInput{
		Email:        req.Email,
		Password:     req.Password,
		Name:         req.Name,
		DormBuilding: req.DormBuilding,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	token, user, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("ต้องเข้าสู่ระบบก่อน"))
		return
	}

	user, err := h.auth.Me(r.Context(), userID)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, errors.New("ไม่พบผู้ใช้"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("เกิดข้อผิดพลาด"))
		return
	}

	writeJSON(w, http.StatusOK, user)
}

type googleLoginRequest struct {
	IDToken string `json:"idToken"`
}

// GoogleLogin — POST /api/auth/google รับ ID token จาก Google Identity Services
// (ฝั่ง frontend ใช้ Google Sign-In button ได้ token นี้มาโดยตรง ไม่ต้อง redirect ไปมา)
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req googleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("รูปแบบข้อมูลไม่ถูกต้อง"))
		return
	}

	token, user, err := h.auth.LoginWithGoogle(r.Context(), req.IDToken)
	switch {
	case errors.Is(err, service.ErrGoogleAuthDisabled):
		writeError(w, http.StatusNotImplemented, err)
	case errors.Is(err, auth.ErrInvalidGoogleToken):
		writeError(w, http.StatusUnauthorized, err)
	case err != nil:
		log.Printf("google login ล้มเหลว: %v", err) // print รายละเอียดจริงไว้ที่ terminal ฝั่ง server เพื่อ debug
		writeError(w, http.StatusInternalServerError, errors.New("เข้าสู่ระบบด้วย Google ไม่สำเร็จ"))
	default:
		writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
	}
}
