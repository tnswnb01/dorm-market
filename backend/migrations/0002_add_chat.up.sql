-- Phase 2: In-app real-time chat
-- 1 conversation ต่อ 1 คู่ (listing, buyer) — seller มาจาก listing.seller_id เสมอ
-- กันไม่ให้มีคนคุยกันซ้ำหลาย thread สำหรับสินค้าชิ้นเดียวกัน

CREATE TABLE conversations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id  UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    buyer_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    seller_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- อัปเดตทุกครั้งที่มีข้อความใหม่ ใช้ sort รายการแชทให้ล่าสุดขึ้นก่อน
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (listing_id, buyer_id),
    CHECK (buyer_id != seller_id) -- ป้องกันเจ้าของประกาศทักแชทหาตัวเอง
);

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_conversations_buyer ON conversations(buyer_id);
CREATE INDEX idx_conversations_seller ON conversations(seller_id);
CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at);
