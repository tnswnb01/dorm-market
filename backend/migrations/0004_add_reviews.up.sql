-- ระบบรีวิว/ให้คะแนนหลังซื้อขาย — ทำให้ trust_score มีความหมายจริง (จากเดิมทุกคนติด 100 ตายตัว)
--
-- เงื่อนไขที่ enforce ผ่าน application logic (ไม่ใช่ DB constraint เพราะต้องเช็คสถานะ listing
-- และประวัติการคุยด้วย ซึ่ง SQL constraint ตรงๆ ทำไม่ได้):
--   - รีวิวได้เฉพาะตอน listing สถานะ 'sold'
--   - ผู้รีวิวต้องเคยมี conversation กับผู้ขายเกี่ยวกับ listing นั้นจริง (กันรีวิวปลอม)
--   - รีวิวเจ้าของประกาศตัวเองไม่ได้

CREATE TABLE reviews (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id   UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    reviewer_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reviewee_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating       INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment      TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- 1 คนรีวิวได้แค่ 1 ครั้งต่อ 1 ประกาศ
    UNIQUE (listing_id, reviewer_id),
    CHECK (reviewer_id != reviewee_id)
);

CREATE INDEX idx_reviews_reviewee ON reviews(reviewee_id);
CREATE INDEX idx_reviews_listing ON reviews(listing_id);
