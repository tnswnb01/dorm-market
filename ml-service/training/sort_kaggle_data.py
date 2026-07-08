"""
จัดเรียงรูปจาก Kaggle "Fashion Product Images (Small)" dataset เข้าโฟลเดอร์ตามหมวดหมู่
ให้ตรงกับโครงสร้างที่ prepare_data.py ต้องการ (1 โฟลเดอร์ = 1 หมวดหมู่)

Dataset นี้มาเป็น:
    fashion-dataset/
    ├── images/          <- รูปกองรวมกัน ชื่อไฟล์เป็นเลข id เช่น 12345.jpg
    └── styles.csv       <- ตารางบอกว่ารูปไหนเป็นหมวดหมู่อะไร

สคริปต์นี้อ่าน styles.csv แล้ว copy รูปแต่ละใบไปไว้ในโฟลเดอร์ data/raw/<หมวดหมู่>/
ให้อัตโนมัติ ไม่ต้องลากแยกเอง

วิธีใช้:
    python training/sort_kaggle_data.py \
        --csv path/to/styles.csv \
        --images-dir path/to/images \
        --output data/raw \
        --category-column masterCategory \
        --limit-per-category 300

ถ้าไม่รู้ชื่อคอลัมน์ในไฟล์ csv ของตัวเอง เปิดไฟล์ styles.csv ด้วย Excel/Notepad
ดูหัวตารางได้เลย — dataset ตัวอย่างที่แนะนำใน README มีคอลัมน์ 'masterCategory'
(หมวดกว้างๆ เช่น Apparel, Footwear, Accessories) และ 'subCategory' (หมวดย่อยกว่า)
เลือกใช้อันไหนก็ได้แล้วแต่อยากให้หมวดหมู่กว้างหรือละเอียดแค่ไหน
"""

import argparse
import csv
import shutil
import sys
from collections import defaultdict
from pathlib import Path


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--csv", required=True, help="path ไปที่ styles.csv")
    parser.add_argument("--images-dir", required=True, help="path ไปที่โฟลเดอร์ images/ ที่มีรูปกองรวมกัน")
    parser.add_argument("--output", default="data/raw", help="โฟลเดอร์ปลายทาง (default: data/raw)")
    parser.add_argument("--id-column", default="id", help="ชื่อคอลัมน์ id ในไฟล์ csv (default: id)")
    parser.add_argument(
        "--category-column", default="masterCategory",
        help="ชื่อคอลัมน์หมวดหมู่ที่จะใช้แบ่งโฟลเดอร์ (default: masterCategory)",
    )
    parser.add_argument("--limit-per-category", type=int, default=300, help="จำกัดจำนวนรูปต่อหมวดหมู่")
    parser.add_argument(
        "--image-ext", default=".jpg",
        help="นามสกุลไฟล์รูป (default: .jpg — เปลี่ยนถ้า dataset ใช้ .png)",
    )
    args = parser.parse_args()

    csv_path = Path(args.csv)
    images_dir = Path(args.images_dir)
    output_dir = Path(args.output)

    if not csv_path.exists():
        print(f"ไม่พบไฟล์ {csv_path} — เช็ค path ที่ใส่ไว้อีกที")
        sys.exit(1)
    if not images_dir.exists():
        print(f"ไม่พบโฟลเดอร์ {images_dir} — เช็ค path ที่ใส่ไว้อีกที")
        sys.exit(1)

    counts = defaultdict(int)
    copied = 0
    skipped_missing = 0

    # Kaggle บาง dataset มีบรรทัดที่ column ไม่ครบ (แถวเสีย) ใช้ on_bad_lines กันพังทั้งไฟล์
    with open(csv_path, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)

        if args.id_column not in reader.fieldnames or args.category_column not in reader.fieldnames:
            print(f"ไม่พบคอลัมน์ '{args.id_column}' หรือ '{args.category_column}' ในไฟล์ csv")
            print(f"คอลัมน์ที่มีจริง: {reader.fieldnames}")
            sys.exit(1)

        for row in reader:
            category = row[args.category_column].strip()
            item_id = row[args.id_column].strip()
            if not category or not item_id:
                continue

            if counts[category] >= args.limit_per_category:
                continue

            src = images_dir / f"{item_id}{args.image_ext}"
            if not src.exists():
                skipped_missing += 1
                continue

            dest_dir = output_dir / category
            dest_dir.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, dest_dir / src.name)

            counts[category] += 1
            copied += 1

    print(f"\nจัดเรียงเสร็จแล้ว: copy ไปทั้งหมด {copied} รูป")
    print(f"หมวดหมู่ที่เจอ ({len(counts)} หมวด):")
    for category, n in sorted(counts.items(), key=lambda x: -x[1]):
        print(f"  {category}: {n} รูป")
    if skipped_missing:
        print(f"\n(ข้ามไป {skipped_missing} รายการ เพราะหารูปที่ตรงกับ id ไม่เจอในโฟลเดอร์ images — ปกติ ไม่ต้องกังวล)")


if __name__ == "__main__":
    main()
