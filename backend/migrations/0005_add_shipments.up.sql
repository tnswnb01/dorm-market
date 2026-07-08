-- ระบบติดตามสถานะสินค้าแบบ manual — ผู้ขายกรอก/อัปเดตเอง (ไม่ได้เชื่อม API ขนส่งจริง)
-- ผูกกับ conversation แทน listing ตรงๆ เพราะ 1 ประกาศอาจมีคนสนใจคุยด้วยหลายคน
-- ต้องรู้ชัดว่า shipment นี้เป็นของคู่ (ผู้ซื้อ, ผู้ขาย) คู่ไหนกันแน่

CREATE TABLE shipments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL UNIQUE REFERENCES conversations(id) ON DELETE CASCADE,
    method          TEXT NOT NULL CHECK (method IN ('pickup', 'delivery')),
    courier_name    TEXT NOT NULL DEFAULT '',   -- ใช้เฉพาะ method = delivery
    tracking_number TEXT NOT NULL DEFAULT '',   -- ใช้เฉพาะ method = delivery
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'shipped', 'completed', 'cancelled')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- เก็บ timeline การเปลี่ยนสถานะทุกครั้ง ให้ผู้ซื้อ/ผู้ขายดูประวัติย้อนหลังได้
CREATE TABLE shipment_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_id  UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    status       TEXT NOT NULL,
    note         TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_shipment_events_shipment ON shipment_events(shipment_id, created_at);
