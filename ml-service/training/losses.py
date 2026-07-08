"""
Supervised Contrastive Loss (SupCon) — Khosla et al. 2020 (https://arxiv.org/abs/2004.11362)

แนวคิด: ในหนึ่ง batch ถ้ามีหลายรูปที่ label เดียวกัน (เช่น หมวดหมู่เดียวกัน) ให้ embedding
ของรูปเหล่านั้นถูกดึงเข้าใกล้กัน ส่วนรูปที่ label ต่างกันถูกผลักออกจากกัน — ต่างจาก
self-supervised contrastive loss (SimCLR) ตรงที่ SupCon ใช้ label จริงแทนการสร้างคู่
augmentation เอง เหมาะกับกรณีเรามี label หมวดหมู่สินค้าอยู่แล้ว
"""

import torch
import torch.nn.functional as F


def supervised_contrastive_loss(
    embeddings: torch.Tensor, labels: torch.Tensor, temperature: float = 0.1
) -> torch.Tensor:
    """
    embeddings: (batch_size, dim) — ยังไม่ต้อง normalize มาก่อน ฟังก์ชันนี้ normalize ให้เอง
    labels: (batch_size,) — label จำนวนเต็ม เช่น category id
    """
    device = embeddings.device
    batch_size = embeddings.shape[0]

    embeddings = F.normalize(embeddings, dim=1)

    # similarity matrix: cosine similarity ของทุกคู่ใน batch, หารด้วย temperature
    similarity = torch.matmul(embeddings, embeddings.T) / temperature

    # กัน numerical overflow ตอน exponentiate (ลบค่า max ออกก่อน เป็นเทคนิคมาตรฐาน)
    similarity = similarity - similarity.max(dim=1, keepdim=True).values.detach()

    labels = labels.view(-1, 1)
    positive_mask = torch.eq(labels, labels.T).float().to(device)

    # ไม่นับตัวเองเป็น positive/negative ของตัวเอง
    self_mask = torch.eye(batch_size, device=device)
    positive_mask = positive_mask - self_mask
    logits_mask = 1 - self_mask

    exp_sim = torch.exp(similarity) * logits_mask
    log_prob = similarity - torch.log(exp_sim.sum(dim=1, keepdim=True) + 1e-12)

    # เฉลี่ย log-probability เฉพาะคู่ที่เป็น positive (label เดียวกัน) ของแต่ละตัวอย่าง
    num_positives = positive_mask.sum(dim=1)
    # ตัวอย่างที่ไม่มี positive เลยใน batch (label เดี่ยวโดดๆ) ให้ loss เป็น 0 ไปเลย ไม่หารด้วย 0
    safe_num_positives = torch.clamp(num_positives, min=1)
    mean_log_prob_pos = (positive_mask * log_prob).sum(dim=1) / safe_num_positives

    loss_per_sample = -mean_log_prob_pos
    loss_per_sample = loss_per_sample * (num_positives > 0).float()

    valid_count = (num_positives > 0).float().sum()
    if valid_count == 0:
        return torch.tensor(0.0, device=device, requires_grad=True)

    return loss_per_sample.sum() / valid_count
