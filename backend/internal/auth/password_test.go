package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("my-password")
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if hash == "my-password" {
		t.Fatal("ต้องถูก hash ไม่ใช่เก็บ plain text")
	}

	if !CheckPassword("my-password", hash) {
		t.Error("รหัสผ่านที่ถูกต้องควรผ่านการตรวจสอบ")
	}
	if CheckPassword("wrong-password", hash) {
		t.Error("รหัสผ่านที่ผิดไม่ควรผ่านการตรวจสอบ")
	}
}

func TestHashPassword_DifferentEachTime(t *testing.T) {
	// bcrypt ใส่ salt แบบสุ่ม hash ผลลัพธ์ของรหัสผ่านเดียวกันจึงต้องไม่เหมือนกันทุกครั้ง
	hash1, _ := HashPassword("same-password")
	hash2, _ := HashPassword("same-password")

	if hash1 == hash2 {
		t.Error("hash ของรหัสผ่านเดียวกัน 2 ครั้งไม่ควรเหมือนกัน (ต้องมี salt)")
	}

	if !CheckPassword("same-password", hash1) || !CheckPassword("same-password", hash2) {
		t.Error("ทั้งสอง hash ต้องตรวจสอบผ่านด้วยรหัสผ่านต้นฉบับเดียวกัน")
	}
}
