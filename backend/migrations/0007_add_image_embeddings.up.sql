-- เปิดใช้ image similarity search จริง — เก็บ embedding vector ของแต่ละรูปสินค้า
-- ขนาด 256 มิติ ตรงกับ output ของ ProjectionHead ที่เทรนเอง (ml-service/training/)
-- (ไม่ใช่ 512 มิติแบบ raw CLIP — production ต้องโหลด projection head เสมอถึงจะได้มิติตรงกัน)

CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE listing_images ADD COLUMN embedding vector(256);

-- หมายเหตุ: ยังไม่สร้าง index (เช่น ivfflat/hnsw) เพราะข้อมูลยังน้อย sequential scan
-- เร็วพอสำหรับ scale ของตลาดในหอ ถ้าข้อมูลรูปเยอะขึ้นมาก (หลักหมื่น+) ค่อยเพิ่ม index ทีหลัง:
--   CREATE INDEX idx_listing_images_embedding ON listing_images
--     USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
