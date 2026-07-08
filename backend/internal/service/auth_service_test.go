package service

import (
	"context"
	"errors"
	"testing"

	"dormmarket/internal/auth"
)

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name    string
		input   RegisterInput
		seed    *RegisterInput // สมัครไปแล้วก่อนหน้า (ไว้เทสเคสอีเมลซ้ำ)
		wantErr error
	}{
		{
			name: "สมัครสำเร็จ",
			input: RegisterInput{
				Email: "somchai@ku.ac.th", Password: "password123",
				Name: "สมชาย", DormBuilding: "หอ A",
			},
			wantErr: nil,
		},
		{
			name:    "อีเมลรูปแบบผิด",
			input:   RegisterInput{Email: "not-an-email", Password: "password123", Name: "สมชาย"},
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "รหัสผ่านสั้นเกินไป",
			input:   RegisterInput{Email: "a@ku.ac.th", Password: "123", Name: "สมชาย"},
			wantErr: ErrWeakPassword,
		},
		{
			name:    "ไม่ระบุชื่อ",
			input:   RegisterInput{Email: "a@ku.ac.th", Password: "password123", Name: "  "},
			wantErr: ErrNameRequired,
		},
		{
			name: "อีเมลซ้ำ",
			seed: &RegisterInput{
				Email: "dup@ku.ac.th", Password: "password123", Name: "คนแรก",
			},
			input: RegisterInput{
				Email: "dup@ku.ac.th", Password: "password123", Name: "คนที่สอง",
			},
			wantErr: ErrEmailTaken,
		},
		{
			name: "อีเมลตัวพิมพ์ต่างกันถือว่าซ้ำ (normalize เป็นตัวเล็ก)",
			seed: &RegisterInput{
				Email: "case@ku.ac.th", Password: "password123", Name: "คนแรก",
			},
			input: RegisterInput{
				Email: "CASE@KU.AC.TH", Password: "password123", Name: "คนที่สอง",
			},
			wantErr: ErrEmailTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUserRepository()
			svc := NewAuthService(users, "test-secret")
			ctx := context.Background()

			if tt.seed != nil {
				if _, _, err := svc.Register(ctx, *tt.seed); err != nil {
					t.Fatalf("seed register ไม่ควร error: %v", err)
				}
			}

			token, user, err := svc.Register(ctx, tt.input)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ต้องการ error %v แต่ได้ %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("ไม่ควร error แต่ได้: %v", err)
			}
			if token == "" {
				t.Error("ต้องได้ token กลับมาด้วย")
			}
			if user.ID == "" {
				t.Error("user ต้องมี ID หลังสมัครสำเร็จ")
			}
			if user.PasswordHash == tt.input.Password {
				t.Error("password ต้องถูก hash ไม่ใช่เก็บ plain text")
			}
			if user.TrustScore != 100 {
				t.Errorf("trust score เริ่มต้นต้องเป็น 100 ได้ %d", user.TrustScore)
			}
		})
	}
}

func TestAuthService_Register_PropagatesUnexpectedDBError(t *testing.T) {
	users := newFakeUserRepository()
	users.forceGetByEmailErr = errSimulatedDBFailure
	svc := NewAuthService(users, "test-secret")

	_, _, err := svc.Register(context.Background(), RegisterInput{
		Email: "a@ku.ac.th", Password: "password123", Name: "สมชาย",
	})

	if !errors.Is(err, errSimulatedDBFailure) {
		t.Fatalf("ควร propagate DB error ที่ไม่ใช่ ErrNotFound ออกมา แต่ได้ %v", err)
	}
}

func TestAuthService_Login(t *testing.T) {
	users := newFakeUserRepository()
	svc := NewAuthService(users, "test-secret")
	ctx := context.Background()

	_, _, err := svc.Register(ctx, RegisterInput{
		Email: "somchai@ku.ac.th", Password: "correct-password", Name: "สมชาย",
	})
	if err != nil {
		t.Fatalf("seed register ไม่ควร error: %v", err)
	}

	t.Run("login สำเร็จ", func(t *testing.T) {
		token, user, err := svc.Login(ctx, "somchai@ku.ac.th", "correct-password")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if token == "" {
			t.Error("ต้องได้ token")
		}
		if user.Email != "somchai@ku.ac.th" {
			t.Errorf("ได้ user ผิดคน: %s", user.Email)
		}
	})

	t.Run("รหัสผ่านผิด", func(t *testing.T) {
		_, _, err := svc.Login(ctx, "somchai@ku.ac.th", "wrong-password")
		if !errors.Is(err, ErrInvalidCreds) {
			t.Fatalf("ต้องการ ErrInvalidCreds แต่ได้ %v", err)
		}
	})

	t.Run("ไม่มีอีเมลนี้ในระบบ", func(t *testing.T) {
		_, _, err := svc.Login(ctx, "notfound@ku.ac.th", "anything")
		if !errors.Is(err, ErrInvalidCreds) {
			t.Fatalf("ต้องการ ErrInvalidCreds (ไม่บอกว่าอีเมลไม่มีอยู่ตรงๆ เพื่อความปลอดภัย) แต่ได้ %v", err)
		}
	})

	t.Run("อีเมลตัวพิมพ์ใหญ่/เล็กต่างกันก็ login ได้", func(t *testing.T) {
		_, _, err := svc.Login(ctx, "SOMCHAI@KU.AC.TH", "correct-password")
		if err != nil {
			t.Fatalf("ควร login ผ่านแม้พิมพ์ตัวใหญ่: %v", err)
		}
	})
}

func TestAuthService_Me(t *testing.T) {
	users := newFakeUserRepository()
	svc := NewAuthService(users, "test-secret")
	ctx := context.Background()

	_, user, _ := svc.Register(ctx, RegisterInput{
		Email: "a@ku.ac.th", Password: "password123", Name: "สมชาย",
	})

	t.Run("หา user เจอ", func(t *testing.T) {
		got, err := svc.Me(ctx, user.ID)
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if got.ID != user.ID {
			t.Errorf("ได้ user ผิดคน")
		}
	})

	t.Run("ไม่พบ user", func(t *testing.T) {
		_, err := svc.Me(ctx, "ไม่มีจริง")
		if err == nil {
			t.Fatal("ควร error เมื่อหา user ไม่เจอ")
		}
	})
}

func TestAuthService_LoginWithGoogle(t *testing.T) {
	t.Run("ปิดใช้งานถ้าไม่ได้ตั้งค่า Google verifier", func(t *testing.T) {
		users := newFakeUserRepository()
		svc := NewAuthService(users, "test-secret") // ไม่เรียก WithGoogleVerifier

		_, _, err := svc.LoginWithGoogle(context.Background(), "some-token")
		if !errors.Is(err, ErrGoogleAuthDisabled) {
			t.Fatalf("ต้องการ ErrGoogleAuthDisabled แต่ได้ %v", err)
		}
	})

	t.Run("token ไม่ถูกต้อง ส่ง error ของ verifier กลับตรงๆ", func(t *testing.T) {
		users := newFakeUserRepository()
		verifyErr := errors.New("token ปลอม")
		svc := NewAuthService(users, "test-secret").WithGoogleVerifier(&fakeGoogleVerifier{err: verifyErr})

		_, _, err := svc.LoginWithGoogle(context.Background(), "bad-token")
		if !errors.Is(err, verifyErr) {
			t.Fatalf("ต้องการ error จาก verifier แต่ได้ %v", err)
		}
	})

	t.Run("บัญชีใหม่ทั้งหมด สร้าง user ใหม่อัตโนมัติ", func(t *testing.T) {
		users := newFakeUserRepository()
		claims := &auth.GoogleClaims{
			Sub: "google-111", Email: "newperson@ku.ac.th", Name: "คนใหม่", EmailVerified: true,
		}
		svc := NewAuthService(users, "test-secret").WithGoogleVerifier(&fakeGoogleVerifier{claims: claims})

		token, user, err := svc.LoginWithGoogle(context.Background(), "valid-token")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if token == "" {
			t.Error("ต้องได้ app token กลับมา")
		}
		if user.Email != "newperson@ku.ac.th" || user.Name != "คนใหม่" {
			t.Errorf("ข้อมูล user ที่สร้างไม่ตรงกับ Google claims: %+v", user)
		}
		if user.GoogleID == nil || *user.GoogleID != "google-111" {
			t.Errorf("ต้องผูก google_id ไว้ด้วย")
		}
	})

	t.Run("login ซ้ำด้วย google_id เดิม ได้ user คนเดิม ไม่สร้างซ้ำ", func(t *testing.T) {
		users := newFakeUserRepository()
		claims := &auth.GoogleClaims{
			Sub: "google-222", Email: "person@ku.ac.th", Name: "คนเดิม", EmailVerified: true,
		}
		svc := NewAuthService(users, "test-secret").WithGoogleVerifier(&fakeGoogleVerifier{claims: claims})
		ctx := context.Background()

		_, user1, _ := svc.LoginWithGoogle(ctx, "token-1")
		_, user2, _ := svc.LoginWithGoogle(ctx, "token-2")

		if user1.ID != user2.ID {
			t.Errorf("login ซ้ำด้วย google_id เดิมต้องได้ user คนเดิม ได้ %s กับ %s", user1.ID, user2.ID)
		}
		if len(users.byID) != 1 {
			t.Errorf("ไม่ควรสร้าง user ซ้ำ ตอนนี้มี %d คน", len(users.byID))
		}
	})

	t.Run("อีเมลตรงกับบัญชีสมัครด้วยรหัสผ่านเดิม ผูก Google ให้อัตโนมัติ", func(t *testing.T) {
		users := newFakeUserRepository()
		svc := NewAuthService(users, "test-secret")
		ctx := context.Background()

		// สมัครด้วยอีเมล/รหัสผ่านไว้ก่อน
		_, existingUser, err := svc.Register(ctx, RegisterInput{
			Email: "hybrid@ku.ac.th", Password: "password123", Name: "คนเดิม",
		})
		if err != nil {
			t.Fatalf("register ไม่ควร error: %v", err)
		}

		// ทีหลังมา login ด้วย Google ด้วยอีเมลเดียวกัน
		claims := &auth.GoogleClaims{
			Sub: "google-333", Email: "HYBRID@KU.AC.TH", Name: "คนเดิม (จาก Google)", EmailVerified: true,
		}
		svc.WithGoogleVerifier(&fakeGoogleVerifier{claims: claims})

		_, googleUser, err := svc.LoginWithGoogle(ctx, "token")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if googleUser.ID != existingUser.ID {
			t.Errorf("ควรผูกเข้ากับบัญชีเดิม (ID เดียวกัน) ได้คนละคน: %s vs %s", googleUser.ID, existingUser.ID)
		}
		if len(users.byID) != 1 {
			t.Errorf("ไม่ควรสร้างบัญชีซ้ำ ตอนนี้มี %d คน", len(users.byID))
		}

		// login ด้วยรหัสผ่านเดิมก็ยังต้องใช้ได้อยู่ (ไม่ได้ตัดสิทธิ์ทางเดิมทิ้ง)
		_, _, err = svc.Login(ctx, "hybrid@ku.ac.th", "password123")
		if err != nil {
			t.Errorf("หลังผูก Google แล้ว login ด้วยรหัสผ่านเดิมควรยังใช้ได้: %v", err)
		}
	})
}
