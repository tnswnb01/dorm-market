-- ระบบแอดมิน + แบนผู้ใช้ — role กำหนดเองผ่าน DB/migration เท่านั้น ไม่มี UI สมัครเป็นแอดมิน
-- is_banned เป็น flag แยกจาก role เพราะ admin เองก็ถูกแบนได้ในทางทฤษฎี (defense in depth
-- ใน RequireAdmin middleware เช็คทั้งสองเงื่อนไข)

ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin'));
ALTER TABLE users ADD COLUMN is_banned BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN ban_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN banned_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN banned_by UUID REFERENCES users(id);

CREATE INDEX idx_users_is_banned ON users(is_banned) WHERE is_banned = true;
