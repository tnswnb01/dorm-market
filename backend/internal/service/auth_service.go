package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"dormmarket/internal/auth"
	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

var (
	ErrEmailTaken         = errors.New("อีเมลนี้ถูกใช้งานแล้ว")
	ErrInvalidCreds       = errors.New("อีเมลหรือรหัสผ่านไม่ถูกต้อง")
	ErrWeakPassword       = errors.New("รหัสผ่านต้องมีอย่างน้อย 8 ตัวอักษร")
	ErrInvalidEmail       = errors.New("รูปแบบอีเมลไม่ถูกต้อง")
	ErrNameRequired       = errors.New("กรุณาระบุชื่อ")
	ErrGoogleAuthDisabled = errors.New("ระบบยังไม่ได้เปิดใช้งาน Google Login")
)

const tokenTTL = 7 * 24 * time.Hour

type AuthService struct {
	users          repository.UserRepository
	jwtSecret      string
	googleVerifier auth.GoogleVerifier // nil ได้ถ้าไม่ได้ตั้งค่า Google OAuth ไว้ (ฟีเจอร์นี้จะปิดไปเฉยๆ)
}

func NewAuthService(users repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
}

// WithGoogleVerifier ต่อ Google OAuth เข้า service — เรียกแบบ optional หลังสร้าง
// AuthService แล้ว (ไม่บังคับใส่ตอน construct เพราะ deployment ที่ไม่ได้ตั้งค่า
// Google Client ID ไว้ก็ควรใช้งานอีเมล/รหัสผ่านต่อได้ตามปกติ)
func (s *AuthService) WithGoogleVerifier(v auth.GoogleVerifier) *AuthService {
	s.googleVerifier = v
	return s
}

type RegisterInput struct {
	Email        string
	Password     string
	Name         string
	DormBuilding string
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (string, *models.User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if !strings.Contains(email, "@") {
		return "", nil, ErrInvalidEmail
	}
	if len(in.Password) < 8 {
		return "", nil, ErrWeakPassword
	}
	if strings.TrimSpace(in.Name) == "" {
		return "", nil, ErrNameRequired
	}

	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return "", nil, ErrEmailTaken
	} else if !errors.Is(err, repository.ErrNotFound) {
		return "", nil, err
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return "", nil, err
	}

	user := &models.User{
		Email:        email,
		PasswordHash: hash,
		Name:         strings.TrimSpace(in.Name),
		DormBuilding: strings.TrimSpace(in.DormBuilding),
	}
	if err := s.users.Create(ctx, user); err != nil {
		return "", nil, err
	}

	token, err := auth.GenerateToken(user.ID, s.jwtSecret, tokenTTL)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		return "", nil, ErrInvalidCreds
	}
	if err != nil {
		return "", nil, err
	}

	if !auth.CheckPassword(password, user.PasswordHash) {
		return "", nil, ErrInvalidCreds
	}

	token, err := auth.GenerateToken(user.ID, s.jwtSecret, tokenTTL)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *AuthService) Me(ctx context.Context, userID string) (*models.User, error) {
	return s.users.GetByID(ctx, userID)
}

// LoginWithGoogle รับ Google ID token (จาก Google Identity Services ฝั่ง frontend)
// verify แล้ว หา-หรือสร้าง user ให้ตรงกับบัญชี Google นั้น ตามลำดับนี้:
//  1. เคยผูก google_id นี้ไว้แล้ว -> login เข้าบัญชีเดิม
//  2. ยังไม่เคยผูก แต่มี email ตรงกับบัญชีที่สมัครด้วยอีเมล/รหัสผ่านไว้ก่อน -> ผูกให้อัตโนมัติ
//  3. ไม่เคยมีมาก่อนเลย -> สร้างบัญชีใหม่ (ตั้ง password เป็นค่าสุ่มที่ไม่มีใครรู้ เพราะ
//     บัญชีนี้ตั้งใจให้ login ผ่าน Google เท่านั้น จนกว่าจะมีฟีเจอร์ "ตั้งรหัสผ่าน" เพิ่มทีหลัง)
func (s *AuthService) LoginWithGoogle(ctx context.Context, idToken string) (string, *models.User, error) {
	if s.googleVerifier == nil {
		return "", nil, ErrGoogleAuthDisabled
	}

	claims, err := s.googleVerifier.Verify(ctx, idToken)
	if err != nil {
		return "", nil, err
	}

	user, err := s.users.GetByGoogleID(ctx, claims.Sub)
	switch {
	case err == nil:
		// เคยผูกไว้แล้ว ใช้บัญชีเดิม
	case errors.Is(err, repository.ErrNotFound):
		user, err = s.findOrCreateByEmail(ctx, claims)
		if err != nil {
			return "", nil, err
		}
	default:
		return "", nil, err
	}

	token, err := auth.GenerateToken(user.ID, s.jwtSecret, tokenTTL)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *AuthService) findOrCreateByEmail(ctx context.Context, claims *auth.GoogleClaims) (*models.User, error) {
	email := strings.ToLower(strings.TrimSpace(claims.Email))

	existing, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		// มีบัญชีอีเมล/รหัสผ่านอยู่แล้ว ผูก Google เข้ากับบัญชีเดิมให้อัตโนมัติ
		// (ปลอดภัยเพราะเราเช็ค email_verified=true จาก Google มาแล้วใน verifier)
		if linkErr := s.users.LinkGoogleID(ctx, existing.ID, claims.Sub); linkErr != nil {
			return nil, linkErr
		}
		googleID := claims.Sub
		existing.GoogleID = &googleID
		return existing, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// ไม่เคยมีบัญชีมาก่อนเลย สร้างใหม่ ตั้งรหัสผ่านเป็นค่าสุ่มที่เดาไม่ได้
	randomPassword, err := generateRandomPassword()
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(randomPassword)
	if err != nil {
		return nil, err
	}

	googleID := claims.Sub
	user := &models.User{
		Email:        email,
		PasswordHash: hash,
		Name:         claims.Name,
		GoogleID:     &googleID,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// generateRandomPassword สร้างรหัสผ่านสุ่มที่ไม่มีใครเดาได้ ใช้สำหรับบัญชีที่สมัคร
// ผ่าน Google เท่านั้น (ป้องกันไม่ให้ column password_hash เป็นค่าว่าง/เดาง่าย)
func generateRandomPassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
