-- เพิ่มการรองรับ Google Login (OAuth) แบบ hybrid — คงระบบอีเมล/รหัสผ่านเดิมไว้ด้วย
-- google_id เป็น NULL ได้สำหรับ user ที่สมัครด้วยอีเมล/รหัสผ่านแบบเดิม
-- 1 บัญชี Google ผูกได้กับ 1 user เท่านั้น (unique constraint)

ALTER TABLE users ADD COLUMN google_id TEXT UNIQUE;
