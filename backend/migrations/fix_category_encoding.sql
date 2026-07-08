-- แก้ไขชื่อหมวดหมู่ที่ตัวอักษรไทยเสียหาย (กลายเป็น "?") จากปัญหา encoding ตอนรัน
-- migration ผ่าน PowerShell (Get-Content ไม่รักษา UTF-8 ให้ถูกต้องตอน pipe ไปยัง docker exec)
--
-- แก้แค่คอลัมน์ name เท่านั้น เพราะ slug (ภาษาอังกฤษ) ไม่ได้เสียหาย ใช้ slug เป็นตัวอ้างอิงได้ปลอดภัย

UPDATE categories SET name = 'หนังสือ/ตำรา' WHERE slug = 'books';
UPDATE categories SET name = 'เฟอร์นิเจอร์' WHERE slug = 'furniture';
UPDATE categories SET name = 'เครื่องใช้ไฟฟ้า' WHERE slug = 'electronics';
UPDATE categories SET name = 'เสื้อผ้า/แฟชั่น' WHERE slug = 'clothing';
UPDATE categories SET name = 'ของใช้ในห้อง' WHERE slug = 'room-essentials';
UPDATE categories SET name = 'จักรยาน/ยานพาหนะ' WHERE slug = 'vehicles';
UPDATE categories SET name = 'อื่นๆ' WHERE slug = 'other';
