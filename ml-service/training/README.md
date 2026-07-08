# Training Pipeline — Fine-tune Image Similarity สำหรับ DormMarket

เอกสารนี้อธิบายวิธีเทรน **ProjectionHead** (ส่วนเดียวที่เทรนเองจริงๆ) บน CLIP embeddings
ทั้งหมดนี้ต้องรันที่เครื่องคุณเอง (มี internet ปกติ) เพราะแซนด์บ็อกซ์ที่ใช้พัฒนาโค้ดนี้
ต่อเน็ตออกไปโหลด CLIP weights/dataset จริงไม่ได้

## แนวคิดโดยสรุป

```
รูปภาพ → CLIP ViT-B/32 (frozen, pretrained) → 512-dim → ProjectionHead (เทรนเอง) → 256-dim
```

เทรนด้วย **Supervised Contrastive Loss**: ดึง embedding ของสินค้าหมวดหมู่เดียวกันเข้าใกล้กัน
ผลักหมวดหมู่ต่างกันออกจากกัน — ทำให้ similarity search แม่นขึ้นสำหรับโดเมนตลาดมือสองโดยเฉพาะ
(ต่างจาก CLIP เปล่าๆ ที่ถูกเทรนมาให้เข้าใจภาพทั่วไป ไม่ได้ specialize กับของมือสอง)

**พิสูจน์แล้วว่า pipeline ถูกต้อง:** รัน `python tests/verify_pipeline_numpy.py` ดูได้เลย
(ใช้ numpy + numerical gradient จำลอง proof-of-concept ขนาดเล็ก ไม่ต้องมี torch/internet)
เห็นชัดว่า loss ลดจริงและ embedding จัดกลุ่มตามหมวดหมู่ได้ดีขึ้นจริงหลังเทรน

## ขั้นตอนที่ 1: ติดตั้ง dependencies

```bash
cd ml-service
pip install -r requirements.txt
```

## ขั้นตอนที่ 2: หา dataset

แนะนำ **[Fashion Product Images (Small)](https://www.kaggle.com/datasets/paramaggarwal/fashion-product-images-small)**
บน Kaggle — มีรูป ~44,000 รูป แบ่งหมวดหมู่ชัดเจน (เสื้อผ้า, รองเท้า, กระเป๋า, เครื่องประดับ ฯลฯ)
ขนาดไฟล์เล็ก (~600MB) ดาวน์โหลดเร็ว ตรงกับหมวดหมู่ "เสื้อผ้า/แฟชั่น" ในระบบเราได้เลย

**ถ้าอยากได้ครอบคลุมหมวดหมู่อื่นด้วย** (หนังสือ, เฟอร์นิเจอร์, เครื่องใช้ไฟฟ้า) ลองรวมกับ:
- [Amazon Products Dataset](https://www.kaggle.com/datasets/asaniczka/amazon-products-dataset-2023-1-4m-products) — สินค้าหลากหลายหมวดหมู่ กรองเอาเฉพาะหมวดที่ต้องการ
- หรือถ่ายรูปสินค้าตัวอย่างเองสัก 30-50 รูปต่อหมวดหมู่ (คุณภาพจะตรงกับของจริงในหอมากกว่า dataset สาธารณะ)

### ไฟล์ที่โหลดจาก Kaggle ไม่ได้จัดโฟลเดอร์ตามหมวดหมู่ให้ทันที — ใช้สคริปต์นี้ช่วยจัดให้

Dataset จาก Kaggle ส่วนใหญ่ (รวมถึง Fashion Product Images) มาเป็นรูปกองรวมกันในโฟลเดอร์
เดียว + ไฟล์ `styles.csv` บอกหมวดหมู่แยกต่างหาก **ไม่ใช่โครงสร้างที่ `prepare_data.py`
ต้องการโดยตรง** ใช้สคริปต์ `sort_kaggle_data.py` ที่เตรียมไว้ให้ จัดเรียงรูปเข้าโฟลเดอร์ตาม
หมวดหมู่อัตโนมัติแทนการลากแยกเอง:

```bash
python training/sort_kaggle_data.py \
    --csv fashion-dataset/styles.csv \
    --images-dir fashion-dataset/images \
    --output data/raw \
    --category-column masterCategory \
    --limit-per-category 300
```

จะได้โฟลเดอร์ `data/raw/<หมวดหมู่>/` พร้อมใช้กับ `prepare_data.py` ทันที (ข้ามขั้นตอน "จัด
โครงสร้างโฟลเดอร์เอง" ด้านล่างไปได้เลย) ปรับ `--category-column` เป็น `subCategory` แทนได้
ถ้าอยากได้หมวดหมู่ละเอียดกว่า (ดูชื่อคอลัมน์ที่มีจริงได้จากการเปิด `styles.csv` ด้วย
Excel/Notepad)

**หมายเหตุ:** dataset นี้เป็นสินค้าแฟชั่นล้วน หมวดหมู่ที่ได้ (Apparel, Footwear, Accessories
ฯลฯ) จะไม่ตรงกับหมวดหมู่จริงในตลาดของเรา (หนังสือ, เฟอร์นิเจอร์ ฯลฯ) เสียทีเดียว — ใช้ได้ดีสำหรับ
**พิสูจน์ว่า pipeline ทำงานถูกต้อง** (โมเดลแยกเสื้อผ้ากับรองเท้าออกจากกันได้จริงไหม) แต่ถ้าอยากได้
โมเดลที่แม่นกับของจริงในหอ ควรผสมรูปสินค้าจริงจากตลาดของเราเข้าไปด้วยทีหลัง

ดาวน์โหลดแล้วจัดโครงสร้างโฟลเดอร์แบบนี้ (ชื่อโฟลเดอร์ย่อย = ชื่อหมวดหมู่) **ถ้าไม่ได้ใช้สคริปต์
ด้านบน** (เช่น จัดเรียงเองหรือใช้ dataset ที่มีโครงสร้างพร้อมอยู่แล้ว):

```
ml-service/data/raw/
├── books/
│   ├── img001.jpg
│   └── ...
├── furniture/
│   └── ...
├── electronics/
│   └── ...
└── clothing/
    └── ...
```

## ขั้นตอนที่ 3: Precompute CLIP embeddings

```bash
python training/prepare_data.py --input data/raw --output data/embeddings.npz --limit-per-class 500
```

ครั้งแรกจะโหลด CLIP pretrained weights จาก HuggingFace (ใช้เวลาสักพัก ~350MB)
หลังจากนั้นจะเดินไฟล์ทุกรูปในแต่ละหมวดหมู่ แปลงเป็น embedding แล้วเก็บเป็นไฟล์เดียว (`.npz`)

## ขั้นตอนที่ 4: เทรน

```bash
python training/train.py --data data/embeddings.npz --epochs 30
```

จะ print loss และ "clustering quality" (ค่าความต่างระหว่าง similarity ภายในกลุ่มเดียวกัน
กับข้ามกลุ่ม — ยิ่งสูงยิ่งดี) ทุก epoch ให้ดูว่าค่าดีขึ้นเรื่อยๆ ถ้า loss ไม่ลดลงเลยหลังผ่านไป
สัก 5-10 epoch ลองปรับ `--lr` ให้เล็กลง หรือเช็คว่าแต่ละหมวดหมู่มีรูปมากพอ (อย่างน้อย ~50 รูป)

ผลลัพธ์: `training/projection_head.pt`

## ขั้นตอนที่ 5: ใช้งานจริงกับ ml-service

```bash
export EMBEDDER_MODE=clip
export PROJECTION_HEAD_PATH=training/projection_head.pt
uvicorn app.main:app --port 8001
```

ตอนนี้ `/embed` endpoint จะใช้ CLIP + projection head ที่เทรนเองแทน MockEmbedder แล้ว

## ทำไมออกแบบแบบนี้ (สรุปสั้นๆ สำหรับตอนอธิบายให้คนอื่นฟัง)

- **ไม่เทรน CLIP ใหม่ทั้งหมด** เพราะต้องใช้รูปหลายร้อยล้านคู่ ไม่มีใครทำเองได้จริงในระดับโปรเจกต์นี้
- **เทรนแค่ projection head เล็กๆ** (2-layer MLP) ต่อท้าย frozen CLIP — วิธีนี้เรียกว่า
  "linear probing" หรือ "lightweight fine-tuning" เป็นเทคนิคมาตรฐานในวงการ (ใช้กันจริงใน
  production หลายที่ ไม่ใช่ทางลัดที่ด้อยคุณภาพ)
- **ใช้ label หมวดหมู่ที่มีอยู่แล้ว** (weak supervision) แทนที่จะต้องมานั่ง label คู่รูป
  "เหมือน/ไม่เหมือน" เองทีละคู่ ซึ่งจะใช้เวลานานมากและ subjective
- **precompute embedding ก่อนเทรน** เพราะ backbone freeze ไว้ ค่าไม่เปลี่ยนทุก epoch —
  ประหยัดเวลาเทรนได้มาก (ไม่ต้องรัน CLIP forward pass ซ้ำ)
