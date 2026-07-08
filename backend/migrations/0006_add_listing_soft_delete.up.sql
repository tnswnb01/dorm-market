-- รองรับการลบประกาศแบบ soft delete — ไม่ hard delete จริงเพราะจะทำลาย
-- ประวัติแชท/รีวิว/shipment ที่ผูกกับ listing นี้ไปด้วย (FK cascade)
-- deleted_at ไม่ null = ถูกลบแล้ว จะถูกซ่อนจากหน้าค้นหา/รายการ แต่ยังเข้าถึงตรงๆ ผ่าน ID ได้
-- (จำเป็นสำหรับหน้าแชท/ประวัติการซื้อที่อ้างอิงถึง listing นี้อยู่)

ALTER TABLE listings ADD COLUMN deleted_at TIMESTAMPTZ;
