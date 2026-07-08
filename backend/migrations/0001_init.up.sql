-- Phase 1 schema: users, categories, listings, listing_images
-- ตั้งใจเผื่อ column ที่ Phase 2 (ML) จะใช้ไว้แล้ว (suggested_price) เพื่อไม่ต้อง
-- migrate แบบ breaking change ทีหลัง

CREATE EXTENSION IF NOT EXISTS pgcrypto; -- สำหรับ gen_random_uuid()

CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    name           TEXT NOT NULL,
    dorm_building  TEXT NOT NULL DEFAULT '',
    avatar_url     TEXT NOT NULL DEFAULT '',
    trust_score    INTEGER NOT NULL DEFAULT 100,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE categories (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE
);

CREATE TABLE listings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id     UUID NOT NULL REFERENCES categories(id),
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    condition       TEXT NOT NULL DEFAULT 'good',
    price           INTEGER NOT NULL CHECK (price > 0),
    suggested_price INTEGER, -- เตรียมไว้สำหรับ Phase 2 (ML price suggestion), ยังไม่ใช้ตอนนี้
    status          TEXT NOT NULL DEFAULT 'available',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE listing_images (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id  UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    url         TEXT NOT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0
    -- หมายเหตุ: Phase 2 จะเพิ่ม column `embedding vector(512)` ตรงนี้
    -- สำหรับ image similarity search ด้วย pgvector extension
);

CREATE INDEX idx_listings_category ON listings(category_id);
CREATE INDEX idx_listings_status ON listings(status);
CREATE INDEX idx_listings_seller ON listings(seller_id);
CREATE INDEX idx_listing_images_listing ON listing_images(listing_id);

-- Seed หมวดหมู่เริ่มต้นที่ใช้บ่อยในหอ/มหาลัย
INSERT INTO categories (name, slug) VALUES
    ('หนังสือ/ตำรา', 'books'),
    ('เฟอร์นิเจอร์', 'furniture'),
    ('เครื่องใช้ไฟฟ้า', 'electronics'),
    ('เสื้อผ้า/แฟชั่น', 'clothing'),
    ('ของใช้ในห้อง', 'room-essentials'),
    ('จักรยาน/ยานพาหนะ', 'vehicles'),
    ('อื่นๆ', 'other');
