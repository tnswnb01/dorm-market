package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// เป็น JWT implementation แบบง่าย (HS256 เท่านั้น) เขียนเองด้วย stdlib ล้วน
// เพียงพอสำหรับ auth ภายในระบบของเรา ไม่ต้องพึ่ง library ภายนอกเพิ่ม

var ErrInvalidToken = errors.New("invalid or expired token")

type Claims struct {
	UserID    string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

func GenerateToken(userID string, secret string, ttl time.Duration) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	claims := Claims{
		UserID:    userID,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(ttl).Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsPart := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerPart + "." + claimsPart

	signature := sign(signingInput, secret)
	return signingInput + "." + signature, nil
}

func ParseToken(token string, secret string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSig := sign(signingInput, secret)
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return nil, ErrInvalidToken
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrInvalidToken
	}

	return &claims, nil
}

func sign(input string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
