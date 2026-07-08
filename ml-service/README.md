# DormMarket ML Service

แปลงรูปภาพสินค้าเป็น embedding vector สำหรับ image similarity search (Phase 2)

## โครงสร้าง

```
ml-service/
├── app/
│   ├── main.py              # FastAPI app — endpoint /embed, /health
│   ├── embedder.py          # Embedder abstraction: MockEmbedder (เทส) / ClipEmbedder (จริง)
│   └── projection_head.py   # โมเดล MLP เล็กๆ ที่เทรนเอง ต่อท้าย frozen CLIP
├── training/
│   ├── prepare_data.py      # precompute CLIP embeddings จาก dataset
│   ├── train.py             # เทรน ProjectionHead จริง (PyTorch)
│   ├── losses.py            # Supervised Contrastive Loss
│   └── README.md            # คู่มือเทรนแบบละเอียด (แนะนำ dataset, ขั้นตอนทีละสเต็ป)
└── tests/
    └── verify_pipeline_numpy.py   # พิสูจน์กลไกการเทรนด้วย numpy ล้วน ไม่ต้องมี torch
```

## รันแบบเร็ว (dev/test — ไม่ต้องมี internet, ไม่ต้องมี torch)

```bash
pip install -r requirements.txt   # ใช้แค่ fastapi, numpy, pillow ก็พอสำหรับโหมดนี้
EMBEDDER_MODE=mock uvicorn app.main:app --port 8001
```

`MockEmbedder` แปลงรูปเป็น embedding แบบ deterministic (รูปเดียวกัน = embedding เดียวกันเสมอ)
ใช้ยืนยันว่า API, การเก็บ vector, และ similarity search endpoint ทำงานถูกต้อง โดยไม่ต้องพึ่ง
โมเดล ML จริง — **ห้ามใช้ใน production**

## รันแบบจริง (production — ต้องมี internet, ต้องเทรน projection head ก่อน)

ดู `training/README.md` สำหรับขั้นตอนเทรนแบบละเอียด สรุปสั้นๆ:

```bash
pip install torch open_clip_torch scikit-learn tqdm   # dependency ที่หนักกว่า
python training/prepare_data.py --input data/raw --output data/embeddings.npz
python training/train.py --data data/embeddings.npz
EMBEDDER_MODE=clip PROJECTION_HEAD_PATH=training/projection_head.pt uvicorn app.main:app --port 8001
```

## เทส

```bash
python tests/verify_pipeline_numpy.py
```
