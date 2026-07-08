package auth

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	token, err := GenerateToken("user-123", "my-secret", time.Hour)
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if token == "" {
		t.Fatal("ต้องได้ token กลับมา")
	}

	claims, err := ParseToken(token, "my-secret")
	if err != nil {
		t.Fatalf("parse token ที่เพิ่ง generate ควรผ่าน: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("UserID ไม่ตรง ได้ %s", claims.UserID)
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, _ := GenerateToken("user-123", "correct-secret", time.Hour)

	_, err := ParseToken(token, "wrong-secret")
	if err != ErrInvalidToken {
		t.Fatalf("secret ผิดต้องได้ ErrInvalidToken แต่ได้ %v", err)
	}
}

func TestParseToken_Expired(t *testing.T) {
	// สร้าง token ที่หมดอายุไปแล้ว (ttl ติดลบ)
	token, _ := GenerateToken("user-123", "secret", -time.Hour)

	_, err := ParseToken(token, "secret")
	if err != ErrInvalidToken {
		t.Fatalf("token หมดอายุต้องได้ ErrInvalidToken แต่ได้ %v", err)
	}
}

func TestParseToken_Malformed(t *testing.T) {
	tests := []string{
		"",
		"not-a-jwt",
		"only.two-parts",
		"one.two.three.four",
	}

	for _, tok := range tests {
		t.Run(tok, func(t *testing.T) {
			_, err := ParseToken(tok, "secret")
			if err != ErrInvalidToken {
				t.Errorf("รูปแบบ token ผิด ต้องได้ ErrInvalidToken แต่ได้ %v", err)
			}
		})
	}
}

func TestParseToken_TamperedPayload(t *testing.T) {
	token, _ := GenerateToken("user-123", "secret", time.Hour)

	// แก้ตัวอักษรตัวสุดท้ายก่อน signature (จำลองการปลอมแปลง payload)
	tampered := token[:len(token)-5] + "AAAAA"

	_, err := ParseToken(tampered, "secret")
	if err != ErrInvalidToken {
		t.Fatalf("token ที่ถูกแก้ไขต้องได้ ErrInvalidToken แต่ได้ %v", err)
	}
}
