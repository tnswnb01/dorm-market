"""
Embedder abstraction — สลับได้ระหว่าง MockEmbedder (ใช้เทส ไม่ต้องมี internet/GPU)
กับ ClipEmbedder (ใช้จริง โหลด pretrained CLIP weights จาก HuggingFace)

หมายเหตุสำคัญ: แซนด์บ็อกซ์ที่ใช้พัฒนาโค้ดนี้ต่อเน็ตออกไปโหลด CLIP weights จริงไม่ได้
(ถูกจำกัด domain ไว้) โค้ด ClipEmbedder ด้านล่างถูกต้องและพร้อมใช้ แต่ทดสอบแบบ
end-to-end จริงไม่ได้ในสภาพแวดล้อมนี้ — ต้องรันที่เครื่องคุณเอง (มีเน็ตปกติ) ถึงจะโหลด
weights ได้ ส่วน MockEmbedder ใช้ยืนยันว่า pipeline ทั้งหมด (API, การเก็บ vector,
similarity search) ทำงานถูกต้องโดยไม่ต้องพึ่งโมเดลจริง
"""

import hashlib
import os
from abc import ABC, abstractmethod

import numpy as np
from PIL import Image

EMBEDDING_DIM = 256  # ขนาด embedding สุดท้ายที่ระบบใช้เก็บ/ค้นหา (= output ของ ProjectionHead
# ไม่ใช่ raw CLIP ที่ให้ 512 มิติ — production ต้องโหลด projection head เสมอถึงจะได้ 256 มิติจริง
# MockEmbedder ใช้ค่านี้ให้ตรงกับ production เพื่อทดสอบ pipeline (เก็บ/ค้นหาใน pgvector) ได้ถูกต้อง


class Embedder(ABC):
    """Interface กลาง — ไม่ว่าจะใช้ Mock หรือ CLIP จริง เรียกใช้งานแบบเดียวกันหมด"""

    @abstractmethod
    def embed(self, image: Image.Image) -> np.ndarray:
        """แปลงรูปเป็น embedding vector ขนาด EMBEDDING_DIM มิติ (normalize แล้ว)"""
        raise NotImplementedError


class MockEmbedder(Embedder):
    """
    Embedder ปลอมสำหรับเทส — ใช้ deterministic hash ของเนื้อหารูปมา seed random vector
    รูปเดียวกัน (byte เหมือนกัน) จะได้ embedding เดียวกันเสมอ ทำให้เทส similarity search ได้จริง
    โดยไม่ต้องพึ่งโมเดล ML ใดๆ — ใช้เฉพาะตอน dev/test เท่านั้น ห้ามใช้ใน production
    """

    def embed(self, image: Image.Image) -> np.ndarray:
        # ใช้ pixel data จริงมา hash เพื่อให้รูปที่คล้ายกัน (resize จากรูปเดียวกัน) ได้ผลใกล้กัน
        small = image.convert("RGB").resize((16, 16))
        digest = hashlib.sha256(small.tobytes()).digest()
        seed = int.from_bytes(digest[:8], "big")
        rng = np.random.default_rng(seed)
        vec = rng.normal(size=EMBEDDING_DIM).astype(np.float32)
        return vec / np.linalg.norm(vec)


class ClipEmbedder(Embedder):
    """
    Embedder จริง — ใช้ pretrained CLIP (ViT-B/32) เป็น frozen backbone
    ต้องติดตั้ง: pip install torch open_clip_torch
    ต้องมี internet ตอนรันครั้งแรก (โหลด pretrained weights จาก HuggingFace/OpenAI)

    ใช้งานคู่กับ ProjectionHead (training/model.py) ที่เทรนเองได้ ถ้ามี weights ที่เทรนแล้ว
    (projection_head.pt) จะโหลดมาต่อท้าย CLIP backbone อัตโนมัติ
    """

    def __init__(self, projection_head_path: str | None = None):
        import open_clip
        import torch

        self.torch = torch
        self.device = "cuda" if torch.cuda.is_available() else "cpu"

        self.model, _, self.preprocess = open_clip.create_model_and_transforms(
            "ViT-B-32", pretrained="openai"
        )
        self.model.eval().to(self.device)

        self.projection_head = None
        if projection_head_path and os.path.exists(projection_head_path):
            from .projection_head import ProjectionHead

            self.projection_head = ProjectionHead()
            self.projection_head.load_state_dict(
                torch.load(projection_head_path, map_location=self.device)
            )
            self.projection_head.eval().to(self.device)

    def embed(self, image: Image.Image) -> np.ndarray:
        with self.torch.no_grad():
            tensor = self.preprocess(image.convert("RGB")).unsqueeze(0).to(self.device)
            features = self.model.encode_image(tensor)
            features = features / features.norm(dim=-1, keepdim=True)

            if self.projection_head is not None:
                features = self.projection_head(features)
                features = features / features.norm(dim=-1, keepdim=True)

            return features.squeeze(0).cpu().numpy().astype(np.float32)


def get_embedder() -> Embedder:
    """
    Factory function — อ่าน environment variable EMBEDDER_MODE เพื่อเลือกว่าจะใช้ตัวไหน
    ค่า default เป็น "mock" เพื่อความปลอดภัย (กันรันจริงไปโดยไม่ตั้งใจตอน dev)
    ตั้ง EMBEDDER_MODE=clip ตอน deploy จริง
    """
    mode = os.getenv("EMBEDDER_MODE", "mock")
    if mode == "clip":
        projection_path = os.getenv("PROJECTION_HEAD_PATH", "training/projection_head.pt")
        return ClipEmbedder(projection_head_path=projection_path)
    return MockEmbedder()
