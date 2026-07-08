package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

// สร้าง Google ID token ปลอมที่เซ็นด้วย private key ของเราเอง (จำลองสิ่งที่ Google ทำจริง)
// แล้วยัด public key เข้า verifier ตรงๆ (ข้ามการดึงจาก googleapis.com ที่แซนด์บ็อกซ์นี้เข้าไม่ได้)
// วิธีนี้ทดสอบตรรกะ "verify signature + validate claims" ได้จริง ไม่ใช่แค่ mock ผ่านๆ
func makeSignedToken(t *testing.T, priv *rsa.PrivateKey, claims GoogleClaims) string {
	t.Helper()

	header := map[string]string{"alg": "RS256", "typ": "JWT", "kid": "test-key-1"}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsPart := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerPart + "." + claimsPart

	hashed := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed[:])
	if err != nil {
		t.Fatalf("เซ็น token ทดสอบไม่สำเร็จ: %v", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func newTestVerifier(t *testing.T, priv *rsa.PrivateKey, clientID string) *RealGoogleVerifier {
	t.Helper()
	v := NewGoogleVerifier(clientID)
	// ยัด public key เข้าไปตรงๆ แทนการดึงจาก network จริง (unexported field ใน package เดียวกัน)
	v.keys["test-key-1"] = &priv.PublicKey
	v.fetchedAt = time.Now()
	return v
}

func validClaims(clientID string) GoogleClaims {
	return GoogleClaims{
		Sub:           "google-user-123",
		Email:         "student@ku.ac.th",
		EmailVerified: true,
		Name:          "นักศึกษา ทดสอบ",
		Aud:           clientID,
		Iss:           "https://accounts.google.com",
		Exp:           time.Now().Add(time.Hour).Unix(),
	}
}

func TestGoogleVerifier_Verify_Success(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	verifier := newTestVerifier(t, priv, "my-client-id.apps.googleusercontent.com")

	token := makeSignedToken(t, priv, validClaims("my-client-id.apps.googleusercontent.com"))

	claims, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("token ที่ถูกต้องควร verify ผ่าน แต่ error: %v", err)
	}
	if claims.Email != "student@ku.ac.th" {
		t.Errorf("ได้ email ผิด: %s", claims.Email)
	}
	if claims.Sub != "google-user-123" {
		t.Errorf("ได้ sub ผิด: %s", claims.Sub)
	}
}

func TestGoogleVerifier_Verify_WrongSignature(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	otherPriv, _ := rsa.GenerateKey(rand.Reader, 2048) // key คนละอันจากที่ verifier รู้จัก

	verifier := newTestVerifier(t, priv, "my-client-id")
	// เซ็นด้วย private key อีกอัน (จำลองคนร้ายพยายามปลอม token)
	token := makeSignedToken(t, otherPriv, validClaims("my-client-id"))

	_, err := verifier.Verify(context.Background(), token)
	if err == nil {
		t.Fatal("token ที่เซ็นด้วย key ผิดต้อง verify ไม่ผ่าน")
	}
}

func TestGoogleVerifier_Verify_WrongAudience(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	verifier := newTestVerifier(t, priv, "my-real-client-id")

	// token ออกให้แอปอื่น (aud ไม่ตรงกับ client id ของเรา) — ป้องกัน token confusion attack
	token := makeSignedToken(t, priv, validClaims("some-other-app-client-id"))

	_, err := verifier.Verify(context.Background(), token)
	if err == nil {
		t.Fatal("aud ไม่ตรงกันต้อง verify ไม่ผ่าน")
	}
}

func TestGoogleVerifier_Verify_Expired(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	verifier := newTestVerifier(t, priv, "my-client-id")

	claims := validClaims("my-client-id")
	claims.Exp = time.Now().Add(-time.Hour).Unix() // หมดอายุไปแล้ว 1 ชม.
	token := makeSignedToken(t, priv, claims)

	_, err := verifier.Verify(context.Background(), token)
	if err == nil {
		t.Fatal("token หมดอายุต้อง verify ไม่ผ่าน")
	}
}

func TestGoogleVerifier_Verify_WrongIssuer(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	verifier := newTestVerifier(t, priv, "my-client-id")

	claims := validClaims("my-client-id")
	claims.Iss = "https://evil.example.com"
	token := makeSignedToken(t, priv, claims)

	_, err := verifier.Verify(context.Background(), token)
	if err == nil {
		t.Fatal("issuer ที่ไม่ใช่ Google ต้อง verify ไม่ผ่าน")
	}
}

func TestGoogleVerifier_Verify_UnverifiedEmail(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	verifier := newTestVerifier(t, priv, "my-client-id")

	claims := validClaims("my-client-id")
	claims.EmailVerified = false
	token := makeSignedToken(t, priv, claims)

	_, err := verifier.Verify(context.Background(), token)
	if err == nil {
		t.Fatal("อีเมลที่ยังไม่ verify ต้องไม่ผ่าน (กัน account takeover ผ่านอีเมลปลอม)")
	}
}

func TestGoogleVerifier_Verify_MalformedToken(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	verifier := newTestVerifier(t, priv, "my-client-id")

	tests := []string{"", "not-a-jwt", "only.two-parts", "a.b.c.d"}
	for _, tok := range tests {
		t.Run(tok, func(t *testing.T) {
			_, err := verifier.Verify(context.Background(), tok)
			if err == nil {
				t.Errorf("token รูปแบบผิดต้อง verify ไม่ผ่าน")
			}
		})
	}
}
