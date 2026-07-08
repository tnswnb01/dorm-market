"""
เทรน ProjectionHead ด้วย Supervised Contrastive Loss บน CLIP embeddings ที่ precompute ไว้แล้ว

วิธีใช้ (หลังรัน prepare_data.py ได้ data/embeddings.npz แล้ว):
    python training/train.py --data data/embeddings.npz --epochs 30

ผลลัพธ์: ไฟล์ app/../training/projection_head.pt (หรือ path ที่ระบุด้วย --output)
เอาไปใช้กับ ClipEmbedder ได้ทันทีโดยตั้ง PROJECTION_HEAD_PATH ให้ชี้มาที่ไฟล์นี้
"""

import argparse
import sys
from pathlib import Path

import numpy as np
import torch
from sklearn.model_selection import train_test_split
from torch.utils.data import DataLoader, TensorDataset
from tqdm import tqdm

sys.path.insert(0, str(Path(__file__).parent.parent))

from app.projection_head import ProjectionHead  # noqa: E402
from training.losses import supervised_contrastive_loss  # noqa: E402


def evaluate_clustering_quality(embeddings: torch.Tensor, labels: torch.Tensor) -> float:
    """
    วัดคุณภาพ embedding แบบง่ายๆ: ค่าเฉลี่ย cosine similarity ภายในกลุ่มเดียวกัน
    ลบด้วยค่าเฉลี่ย cosine similarity ข้ามกลุ่ม — ยิ่งค่าสูง (บวกมาก) ยิ่งดี
    หมายถึง embedding ของกลุ่มเดียวกันอยู่ใกล้กัน และกลุ่มต่างกันอยู่ไกลกัน ตามที่ต้องการ
    """
    normed = torch.nn.functional.normalize(embeddings, dim=1)
    sim = normed @ normed.T

    same_mask = (labels.view(-1, 1) == labels.view(1, -1)) & ~torch.eye(
        len(labels), dtype=torch.bool
    )
    diff_mask = labels.view(-1, 1) != labels.view(1, -1)

    intra_sim = sim[same_mask].mean().item() if same_mask.any() else 0.0
    inter_sim = sim[diff_mask].mean().item() if diff_mask.any() else 0.0
    return intra_sim - inter_sim


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--data", required=True)
    parser.add_argument("--output", default="training/projection_head.pt")
    parser.add_argument("--epochs", type=int, default=30)
    parser.add_argument("--batch-size", type=int, default=64)
    parser.add_argument("--lr", type=float, default=1e-3)
    args = parser.parse_args()

    raw = np.load(args.data, allow_pickle=True)
    embeddings = torch.tensor(raw["embeddings"], dtype=torch.float32)
    labels = torch.tensor(raw["labels"], dtype=torch.long)
    label_names = raw["label_names"]

    print(f"โหลดข้อมูล: {len(embeddings)} ตัวอย่าง, {len(label_names)} หมวดหมู่")

    train_idx, val_idx = train_test_split(
        np.arange(len(embeddings)), test_size=0.2, stratify=labels.numpy(), random_state=42
    )
    train_ds = TensorDataset(embeddings[train_idx], labels[train_idx])
    train_loader = DataLoader(train_ds, batch_size=args.batch_size, shuffle=True, drop_last=True)

    val_embeddings, val_labels = embeddings[val_idx], labels[val_idx]

    model = ProjectionHead()
    optimizer = torch.optim.Adam(model.parameters(), lr=args.lr)

    print("\nคุณภาพ embedding ก่อนเทรน (แค่ raw CLIP, ยังไม่ผ่าน projection head):")
    baseline_quality = evaluate_clustering_quality(val_embeddings, val_labels)
    print(f"  intra-inter similarity gap: {baseline_quality:.4f}")

    for epoch in range(1, args.epochs + 1):
        model.train()
        total_loss = 0.0
        for batch_embeddings, batch_labels in tqdm(train_loader, desc=f"epoch {epoch}", leave=False):
            optimizer.zero_grad()
            projected = model(batch_embeddings)
            loss = supervised_contrastive_loss(projected, batch_labels)
            loss.backward()
            optimizer.step()
            total_loss += loss.item()

        avg_loss = total_loss / len(train_loader)

        model.eval()
        with torch.no_grad():
            val_projected = model(val_embeddings)
            val_quality = evaluate_clustering_quality(val_projected, val_labels)

        print(f"epoch {epoch:3d} | train loss {avg_loss:.4f} | val intra-inter gap {val_quality:.4f}")

    Path(args.output).parent.mkdir(parents=True, exist_ok=True)
    torch.save(model.state_dict(), args.output)
    print(f"\nบันทึก projection head ไว้ที่ {args.output}")
    print(f"เทียบผลก่อน/หลังเทรน: {baseline_quality:.4f} -> {val_quality:.4f}")
    if val_quality > baseline_quality:
        print("ดีขึ้น — projection head ช่วยแยกหมวดหมู่ได้ดีกว่า raw CLIP embedding")
    else:
        print("ยังไม่ดีขึ้น ลองเพิ่ม epoch, ปรับ learning rate, หรือเช็คว่าข้อมูลแต่ละหมวดหมู่พอไหม")


if __name__ == "__main__":
    main()
