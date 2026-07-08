"""
ProjectionHead — ส่วนเดียวในระบบทั้งหมดที่ "เทรนเอง" จริงๆ

Import torch แบบ lazy (import อยู่ในฟังก์ชัน ไม่ใช่ top-level) เพื่อให้ FastAPI app
ที่รันด้วย MockEmbedder (ไม่ต้องมี torch) ยัง import module นี้ได้โดยไม่ error
"""

INPUT_DIM = 512  # ขนาด embedding ที่ออกมาจาก CLIP ViT-B/32
HIDDEN_DIM = 256
OUTPUT_DIM = 256  # ขนาด embedding สุดท้ายที่เก็บใน pgvector


def ProjectionHead():
    """
    สร้าง MLP 2 ชั้น: Linear -> ReLU -> Linear
    Input: CLIP embedding (512 มิติ, normalize แล้ว)
    Output: embedding โดเมนเฉพาะของเรา (256 มิติ, ยังไม่ normalize — normalize ทีหลังตอนใช้งาน)

    สถาปัตยกรรมเรียบง่ายตั้งใจ เพราะข้อมูลเทรนของเรา (สินค้ามือสองในหอ) มีจำนวนจำกัด
    โมเดลใหญ่เกินไปจะ overfit ง่าย — MLP เล็กๆ แบบนี้เพียงพอสำหรับ "ปรับ" พื้นที่ embedding
    ของ CLIP ให้เข้ากับโดเมนของเรา ไม่ใช่เรียนรู้ feature ใหม่ทั้งหมด
    """
    import torch.nn as nn

    return nn.Sequential(
        nn.Linear(INPUT_DIM, HIDDEN_DIM),
        nn.ReLU(),
        nn.Linear(HIDDEN_DIM, OUTPUT_DIM),
    )
