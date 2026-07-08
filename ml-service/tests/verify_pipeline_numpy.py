"""
พิสูจน์ว่ากลไกการเทรน (2-layer MLP + Supervised Contrastive Loss + gradient descent)
ทำงานถูกต้องจริง โดยไม่ต้องพึ่ง PyTorch/internet — ใช้ numerical gradient (finite
difference) แทน autograd ซึ่งเป็น gradient จริงทางคณิตศาสตร์ (ไม่ใช่ของปลอม)
เพียงแต่คำนวณช้ากว่า analytic backprop มาก จึงใช้ได้แค่กับโมเดล/ข้อมูลขนาดเล็กแบบนี้

ข้อมูลสังเคราะห์จำลองพฤติกรรมของ CLIP embeddings: แต่ละหมวดหมู่มี "จุดศูนย์กลาง" ของตัวเอง
ในปริภูมิเวกเตอร์ บวกด้วย noise แบบสุ่ม — ใกล้เคียงกับที่ CLIP embedding ของสินค้าใน
หมวดหมู่เดียวกันจะรวมกลุ่มกันหลวมๆ (แต่ไม่ perfect เพราะ CLIP ไม่ได้ถูกเทรนมาเพื่อ
domain นี้โดยเฉพาะ — นั่นคือเหตุผลที่เรา fine-tune ต่อ)

รัน: python tests/verify_pipeline_numpy.py
"""

import numpy as np

rng = np.random.default_rng(42)

# ---------- 1. สร้างข้อมูลสังเคราะห์ (จำลอง CLIP embeddings) ----------

INPUT_DIM = 64      # ย่อจาก CLIP จริง (512) ลงมาให้ numerical gradient คำนวณได้ทันเวลา
HIDDEN_DIM = 32
OUTPUT_DIM = 32
N_CLASSES = 5
SAMPLES_PER_CLASS = 40
NOISE_STD = 0.9      # noise สูงพอสมควร จำลองว่า raw CLIP ยังแยกหมวดหมู่ได้ไม่ดีนัก


def make_synthetic_dataset():
    centers = rng.normal(size=(N_CLASSES, INPUT_DIM))
    centers /= np.linalg.norm(centers, axis=1, keepdims=True)

    X, y = [], []
    for class_idx in range(N_CLASSES):
        noise = rng.normal(scale=NOISE_STD, size=(SAMPLES_PER_CLASS, INPUT_DIM))
        samples = centers[class_idx] + noise
        X.append(samples)
        y.extend([class_idx] * SAMPLES_PER_CLASS)
    return np.vstack(X).astype(np.float64), np.array(y)


# ---------- 2. โมเดล: 2-layer MLP (เหมือน ProjectionHead ใน production จริง) ----------


def init_params():
    return {
        "W1": rng.normal(scale=0.1, size=(INPUT_DIM, HIDDEN_DIM)),
        "b1": np.zeros(HIDDEN_DIM),
        "W2": rng.normal(scale=0.1, size=(HIDDEN_DIM, OUTPUT_DIM)),
        "b2": np.zeros(OUTPUT_DIM),
    }


def forward(params, X):
    h = np.maximum(0, X @ params["W1"] + params["b1"])  # ReLU
    out = h @ params["W2"] + params["b2"]
    return out


# ---------- 3. Supervised Contrastive Loss (สูตรเดียวกับ training/losses.py) ----------


def supervised_contrastive_loss(embeddings, labels, temperature=0.2):
    normed = embeddings / np.linalg.norm(embeddings, axis=1, keepdims=True)
    similarity = normed @ normed.T / temperature
    similarity -= similarity.max(axis=1, keepdims=True)

    labels = labels.reshape(-1, 1)
    positive_mask = (labels == labels.T).astype(np.float64)
    n = len(labels)
    self_mask = np.eye(n)
    positive_mask -= self_mask
    logits_mask = 1 - self_mask

    exp_sim = np.exp(similarity) * logits_mask
    log_prob = similarity - np.log(exp_sim.sum(axis=1, keepdims=True) + 1e-12)

    num_positives = positive_mask.sum(axis=1)
    safe_num_positives = np.clip(num_positives, 1, None)
    mean_log_prob_pos = (positive_mask * log_prob).sum(axis=1) / safe_num_positives

    loss_per_sample = -mean_log_prob_pos * (num_positives > 0)
    valid = (num_positives > 0).sum()
    return loss_per_sample.sum() / max(valid, 1)


def loss_fn(params, X, y):
    return supervised_contrastive_loss(forward(params, X), y)


# ---------- 4. Numerical gradient (finite difference) — คือ gradient จริง ไม่ใช่ของปลอม ----------


def numerical_gradient(params, X, y, eps=1e-4):
    grads = {}
    base_loss = loss_fn(params, X, y)
    for key, arr in params.items():
        grad = np.zeros_like(arr)
        it = np.nditer(arr, flags=["multi_index"])
        for _ in it:
            idx = it.multi_index
            original = arr[idx]

            arr[idx] = original + eps
            loss_plus = loss_fn(params, X, y)

            arr[idx] = original - eps
            loss_minus = loss_fn(params, X, y)

            arr[idx] = original
            grad[idx] = (loss_plus - loss_minus) / (2 * eps)
        grads[key] = grad
    return grads, base_loss


# ---------- 5. Metric วัดคุณภาพ embedding (สูตรเดียวกับ training/train.py) ----------


def clustering_quality(embeddings, labels):
    normed = embeddings / np.linalg.norm(embeddings, axis=1, keepdims=True)
    sim = normed @ normed.T
    labels = labels.reshape(-1, 1)
    same_mask = (labels == labels.T) & ~np.eye(len(labels), dtype=bool)
    diff_mask = labels != labels.T
    intra = sim[same_mask].mean()
    inter = sim[diff_mask].mean()
    return intra - inter


# ---------- 6. Training loop ----------


def main():
    X, y = make_synthetic_dataset()

    # แบ่ง train/val แบบง่าย (stratified คร่าวๆ โดยสุ่ม index ต่อคลาส)
    train_idx, val_idx = [], []
    for c in range(N_CLASSES):
        class_indices = np.where(y == c)[0]
        rng.shuffle(class_indices)
        split = int(0.75 * len(class_indices))
        train_idx.extend(class_indices[:split])
        val_idx.extend(class_indices[split:])
    train_idx, val_idx = np.array(train_idx), np.array(val_idx)

    X_train, y_train = X[train_idx], y[train_idx]
    X_val, y_val = X[val_idx], y[val_idx]

    params = init_params()

    print("=" * 60)
    print("พิสูจน์ pipeline การเทรนด้วย numerical gradient (numpy ล้วน)")
    print("=" * 60)

    baseline_val_quality = clustering_quality(X_val, y_val)
    print(f"\nคุณภาพ embedding ก่อนเทรน (raw, ยังไม่ผ่าน MLP): {baseline_val_quality:.4f}")

    initial_projected_quality = clustering_quality(forward(params, X_val), y_val)
    print(f"คุณภาพ embedding ก่อนเทรน (ผ่าน MLP สุ่มเริ่มต้น): {initial_projected_quality:.4f}")

    lr = 0.5
    batch_size = 25
    epochs = 12
    losses = []

    for epoch in range(1, epochs + 1):
        perm = rng.permutation(len(X_train))
        epoch_losses = []

        for start in range(0, len(X_train), batch_size):
            batch_idx = perm[start : start + batch_size]
            X_batch, y_batch = X_train[batch_idx], y_train[batch_idx]

            grads, batch_loss = numerical_gradient(params, X_batch, y_batch)
            for key in params:
                params[key] -= lr * grads[key]

            epoch_losses.append(batch_loss)

        avg_loss = float(np.mean(epoch_losses))
        losses.append(avg_loss)
        val_quality = clustering_quality(forward(params, X_val), y_val)
        print(f"epoch {epoch:2d} | train loss {avg_loss:.4f} | val intra-inter gap {val_quality:.4f}")

    final_val_quality = clustering_quality(forward(params, X_val), y_val)

    print("\n" + "=" * 60)
    print("สรุปผล")
    print("=" * 60)
    print(f"Loss ลดลงจาก {losses[0]:.4f} -> {losses[-1]:.4f}  ({'ลดลงจริง ✓' if losses[-1] < losses[0] else 'ไม่ลด ✗'})")
    print(
        f"Clustering quality (val): {initial_projected_quality:.4f} -> {final_val_quality:.4f} "
        f"({'ดีขึ้น ✓' if final_val_quality > initial_projected_quality else 'แย่ลง ✗'})"
    )

    assert losses[-1] < losses[0], "loss ควรลดลงหลังเทรน"
    assert final_val_quality > initial_projected_quality, "clustering quality ควรดีขึ้นหลังเทรน"
    print("\n✅ pipeline ถูกต้อง — โครงสร้างเดียวกันนี้ (แค่ scale ขึ้นเป็น CLIP 512 มิติจริง")
    print("   และใช้ PyTorch autograd แทน numerical gradient) คือสิ่งที่ training/train.py ทำ")


if __name__ == "__main__":
    main()
