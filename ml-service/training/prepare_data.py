"""
เตรียมข้อมูลเทรน — เดินไฟล์ในโฟลเดอร์ dataset (โครงสร้าง 1 โฟลเดอร์ย่อยต่อ 1 หมวดหมู่)
แล้ว precompute CLIP embedding ของทุกรูป เก็บเป็น .npy cache ไว้

ทำไมต้อง precompute ก่อนแยกจากขั้นตอนเทรน:
CLIP backbone ถูก freeze ไว้ (ไม่เทรน) ดังนั้นค่า embedding ของแต่ละรูปจะไม่เปลี่ยนไม่ว่า
จะเทรนกี่ epoch ก็ตาม — คำนวณครั้งเดียวเก็บไว้ ทำให้ loop เทรน projection head เร็วขึ้นมาก
(ไม่ต้องรัน CLIP forward pass ซ้ำทุก epoch)

วิธีใช้:
    1. ดาวน์โหลด dataset (ดู training/README.md แนะนำ dataset ที่เหมาะกับโปรเจกต์นี้)
    2. จัดโครงสร้างโฟลเดอร์ให้เป็นแบบนี้:
         data/raw/<category_name>/<image files...>
    3. รัน: python training/prepare_data.py --input data/raw --output data/embeddings.npz

ต้องรันที่เครื่องที่มี internet ปกติ (โหลด CLIP weights ครั้งแรก) และติดตั้ง
requirements.txt ให้ครบก่อน (โดยเฉพาะ torch, open_clip_torch)
"""

import argparse
import sys
from pathlib import Path

import numpy as np
from PIL import Image
from tqdm import tqdm

sys.path.insert(0, str(Path(__file__).parent.parent))  # ให้ import app.* ได้


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True, help="โฟลเดอร์ dataset (1 โฟลเดอร์ย่อย = 1 หมวดหมู่)")
    parser.add_argument("--output", default="data/embeddings.npz", help="ไฟล์ผลลัพธ์")
    parser.add_argument("--limit-per-class", type=int, default=500, help="จำกัดจำนวนรูปต่อหมวดหมู่ (กันข้อมูลไม่สมดุลกันเกินไป)")
    args = parser.parse_args()

    from app.embedder import ClipEmbedder  # import ตรงนี้ ไม่ใช่ top-level เพราะต้องมี torch

    embedder = ClipEmbedder()  # ไม่ใส่ projection head — เราต้องการ raw CLIP embedding มาเทรน head เอง

    input_dir = Path(args.input)
    class_dirs = sorted([d for d in input_dir.iterdir() if d.is_dir()])
    if not class_dirs:
        print(f"ไม่พบโฟลเดอร์หมวดหมู่ย่อยใน {input_dir} — เช็คโครงสร้างโฟลเดอร์อีกที")
        sys.exit(1)

    print(f"พบ {len(class_dirs)} หมวดหมู่: {[d.name for d in class_dirs]}")

    embeddings = []
    labels = []
    label_names = []

    for class_idx, class_dir in enumerate(class_dirs):
        label_names.append(class_dir.name)
        image_paths = sorted(
            [p for p in class_dir.iterdir() if p.suffix.lower() in (".jpg", ".jpeg", ".png", ".webp")]
        )[: args.limit_per_class]

        for path in tqdm(image_paths, desc=class_dir.name):
            try:
                image = Image.open(path)
                image.load()
            except Exception as e:
                print(f"  ข้ามไฟล์เสีย {path}: {e}")
                continue

            vec = embedder.embed(image)
            embeddings.append(vec)
            labels.append(class_idx)

    embeddings = np.stack(embeddings)
    labels = np.array(labels)

    Path(args.output).parent.mkdir(parents=True, exist_ok=True)
    np.savez(args.output, embeddings=embeddings, labels=labels, label_names=label_names)

    print(f"\nเสร็จแล้ว: {len(embeddings)} รูป, {len(class_dirs)} หมวดหมู่")
    print(f"บันทึกไว้ที่ {args.output}")


if __name__ == "__main__":
    main()
