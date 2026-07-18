-- ระบบ support ticket แยกต่างหากจากแชทซื้อขายเดิม (conversations/messages) — เพราะเป็นคนละ
-- domain กัน: อันนี้คือผู้ใช้คุยกับแอดมิน ไม่ใช่ผู้ซื้อคุยกับผู้ขาย
-- สถานะ: open (ใหม่/รอตอบ) -> pending (แอดมินตอบแล้ว รอผู้ใช้) -> closed (จบเรื่อง)
-- ผู้ใช้ทักข้อความเพิ่มตอน closed จะ reopen เป็น open ให้อัตโนมัติ (ดู logic ใน service)

CREATE TABLE support_tickets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject     TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'pending', 'closed')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ticket_messages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id   UUID NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    sender_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_support_tickets_status ON support_tickets(status);
CREATE INDEX idx_support_tickets_user ON support_tickets(user_id);
CREATE INDEX idx_ticket_messages_ticket ON ticket_messages(ticket_id, created_at);
