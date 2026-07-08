package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// เขียน Google ID token (RS256 JWT) verifier เองด้วย stdlib ล้วน แทนที่จะพึ่ง
// google.golang.org/api/idtoken (dependency หนักและดึง package อื่นตามมาเยอะ)
// แนวคิดเดียวกับที่เราเขียน JWT (HS256) เองใน jwt.go แค่ต่างตรงนี้ verify ด้วย
// RSA public key ของ Google แทน HMAC secret ของเราเอง

var (
	ErrInvalidGoogleToken = errors.New("google id token ไม่ถูกต้องหรือหมดอายุ")
	googleCertsURL        = "https://www.googleapis.com/oauth2/v3/certs"
)

type GoogleClaims struct {
	Sub           string `json:"sub"` // Google user ID ตัวจริง (unique ถาวร) ใช้อันนี้ผูกบัญชี ไม่ใช้ email
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Aud           string `json:"aud"` // ต้องตรงกับ Google Client ID ของเรา
	Iss           string `json:"iss"`
	Exp           int64  `json:"exp"`
}

// GoogleVerifier คือ interface กลาง — production ใช้ตัวจริงที่เช็คกับ Google
// ส่วนตอนเทสสลับไปใช้ fake ที่คืนค่า claims ตายตัวได้ (ดู fakes_test.go ฝั่ง service)
type GoogleVerifier interface {
	Verify(ctx context.Context, idToken string) (*GoogleClaims, error)
}

type googleJWK struct {
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type googleJWKSet struct {
	Keys []googleJWK `json:"keys"`
}

// RealGoogleVerifier ดึง public key (JWKS) จาก Google มาแคชไว้ แล้ว verify signature จริง
//
// หมายเหตุ: ต้องมี internet เข้าถึง googleapis.com ตอนรัน — ถ้าใช้งานในสภาพแวดล้อมที่
// ถูกจำกัด network (เช่นแซนด์บ็อกซ์ที่ใช้พัฒนาโค้ดนี้) จะ verify ไม่ได้จริง ต้องรันที่เครื่อง
// ที่มีเน็ตปกติ
type RealGoogleVerifier struct {
	clientID string

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func NewGoogleVerifier(clientID string) *RealGoogleVerifier {
	return &RealGoogleVerifier{clientID: clientID, keys: make(map[string]*rsa.PublicKey)}
}

func (v *RealGoogleVerifier) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	fresh := time.Since(v.fetchedAt) < time.Hour
	v.mu.RUnlock()
	if ok && fresh {
		return key, nil
	}

	if err := v.refreshKeys(ctx); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok = v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("ไม่พบ public key ของ Google ที่ kid=%s (certs อาจหมุนไปแล้ว)", kid)
	}
	return key, nil
}

func (v *RealGoogleVerifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleCertsURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("ดึง Google public keys ไม่สำเร็จ (เช็ค internet connection): %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var jwks googleJWKSet
	if err := json.Unmarshal(body, &jwks); err != nil {
		return err
	}

	newKeys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		pubKey, err := jwkToRSAPublicKey(k)
		if err != nil {
			continue // ข้าม key ที่ parse ไม่ได้ ไม่ควร fail ทั้งชุดเพราะ key เดียว
		}
		newKeys[k.Kid] = pubKey
	}

	v.mu.Lock()
	v.keys = newKeys
	v.fetchedAt = time.Now()
	v.mu.Unlock()
	return nil
}

func jwkToRSAPublicKey(k googleJWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

func (v *RealGoogleVerifier) Verify(ctx context.Context, idToken string) (*GoogleClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidGoogleToken
	}

	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidGoogleToken
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, ErrInvalidGoogleToken
	}
	if header.Alg != "RS256" {
		return nil, fmt.Errorf("%w: alg ต้องเป็น RS256 ได้ %s", ErrInvalidGoogleToken, header.Alg)
	}

	pubKey, err := v.getKey(ctx, header.Kid)
	if err != nil {
		return nil, err
	}

	signingInput := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidGoogleToken
	}

	hashed := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed[:], signature); err != nil {
		return nil, fmt.Errorf("%w: signature ไม่ถูกต้อง", ErrInvalidGoogleToken)
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidGoogleToken
	}
	var claims GoogleClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidGoogleToken
	}

	if claims.Aud != v.clientID {
		return nil, fmt.Errorf("%w: aud ไม่ตรงกับ Google Client ID ของเรา", ErrInvalidGoogleToken)
	}
	if claims.Iss != "https://accounts.google.com" && claims.Iss != "accounts.google.com" {
		return nil, fmt.Errorf("%w: issuer ไม่ใช่ Google", ErrInvalidGoogleToken)
	}
	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("%w: token หมดอายุแล้ว", ErrInvalidGoogleToken)
	}
	if !claims.EmailVerified {
		return nil, fmt.Errorf("%w: อีเมล Google ยังไม่ได้ verify", ErrInvalidGoogleToken)
	}

	return &claims, nil
}
