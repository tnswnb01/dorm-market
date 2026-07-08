"""
DormMarket ML Service — แปลงรูปภาพเป็น embedding vector สำหรับ image similarity search

รันด้วย: uvicorn app.main:app --reload --port 8001
ตั้ง EMBEDDER_MODE=clip เมื่อพร้อมใช้ CLIP จริง (ค่า default คือ mock สำหรับ dev/test)
"""

import io

from fastapi import FastAPI, File, HTTPException, UploadFile
from PIL import Image
from pydantic import BaseModel

from app.embedder import get_embedder

app = FastAPI(title="DormMarket ML Service")

# โหลด embedder ครั้งเดียวตอน start service (โหลด CLIP model ทุก request จะช้ามาก)
embedder = get_embedder()


class EmbedResponse(BaseModel):
    embedding: list[float]
    dim: int


class HealthResponse(BaseModel):
    status: str
    embedder_mode: str


@app.get("/health", response_model=HealthResponse)
def health():
    import os

    return HealthResponse(status="ok", embedder_mode=os.getenv("EMBEDDER_MODE", "mock"))


@app.post("/embed", response_model=EmbedResponse)
async def embed_image(file: UploadFile = File(...)):
    raw = await file.read()
    if len(raw) == 0:
        raise HTTPException(status_code=400, detail="ไฟล์ว่างเปล่า")

    try:
        image = Image.open(io.BytesIO(raw))
        image.load()
    except Exception:
        # ไม่เชื่อ content-type header อย่างเดียว เพราะ client บางตัว (เช่น Go multipart
        # writer) ไม่ได้ตั้งค่า header นี้ให้ถูกต้องเสมอไป — ลองเปิดไฟล์จริงแทน
        raise HTTPException(status_code=400, detail="เปิดไฟล์รูปไม่สำเร็จ ไฟล์อาจไม่ใช่รูปภาพหรือเสียหาย")

    vector = embedder.embed(image)
    return EmbedResponse(embedding=vector.tolist(), dim=len(vector))
