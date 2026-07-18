-- ระบบรายงาน — รายงานได้ทั้งประกาศ (listing) และผู้ใช้ (user) จากหน้าเดียวกัน
-- แยกกันด้วย target_type + CHECK บังคับว่าต้องมี FK ตรงกับ target_type เป๊ะๆ
-- (ตั้งใจไม่ CASCADE ลบ report ตอนลบ user/listing เพื่อเก็บประวัติไว้ให้แอดมินตรวจสอบย้อนหลังได้
-- แม้ของที่ถูกรายงานจะถูกลบ/แบนไปแล้ว)

CREATE TABLE reports (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type        TEXT NOT NULL CHECK (target_type IN ('listing', 'user')),
    target_listing_id  UUID REFERENCES listings(id),
    target_user_id     UUID REFERENCES users(id),
    reason             TEXT NOT NULL CHECK (reason IN ('scam', 'inappropriate', 'harassment', 'spam', 'other')),
    description        TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'resolved', 'dismissed')),
    resolution_action  TEXT CHECK (resolution_action IN ('ban_user', 'remove_listing', 'none')),
    resolution_note    TEXT NOT NULL DEFAULT '',
    resolved_by        UUID REFERENCES users(id),
    resolved_at        TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (
        (target_type = 'listing' AND target_listing_id IS NOT NULL AND target_user_id IS NULL) OR
        (target_type = 'user' AND target_user_id IS NOT NULL AND target_listing_id IS NULL)
    ),
    CHECK (target_user_id IS NULL OR reporter_id != target_user_id)
);

CREATE INDEX idx_reports_status ON reports(status);
CREATE INDEX idx_reports_target_listing ON reports(target_listing_id);
CREATE INDEX idx_reports_target_user ON reports(target_user_id);
