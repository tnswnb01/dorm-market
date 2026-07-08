# DormMarket — ตลาดมือสองในหอ/มหาลัย

Marketplace ซื้อขายของมือสองสำหรับนักศึกษาในหอ/มหาลัยเดียวกัน

**สถานะ:** Phase 1 (core marketplace) เสร็จสมบูรณ์ — Phase 2 (AI) กำลังทำอยู่:
price suggestion เสร็จแล้ว, image similarity search มี pipeline ครบแล้วรอเทรนจริง,
fraud detection และ real-time chat ยังไม่เริ่ม (ดูรายละเอียดท้ายไฟล์นี้)

## Stack

| ส่วน | เทคโนโลยี |
|---|---|
| Backend | Go 1.22 (standard library เป็นหลัก + `lib/pq`, `golang.org/x/crypto/bcrypt`) |
| Frontend | React + Vite + React Router + Tailwind CSS |
| Database | PostgreSQL 16 |
| Auth | JWT (เขียนเอง, HS256) + bcrypt |

## โครงสร้างโปรเจกต์

```
dormmarket/
├── docker-compose.yml         # Postgres สำหรับ local dev
├── backend/
│   ├── cmd/api/main.go        # entry point — wiring เท่านั้น ไม่มี logic
│   ├── internal/
│   │   ├── config/            # โหลดค่าจาก environment variable
│   │   ├── database/          # เปิด connection pool
│   │   ├── models/            # struct ของข้อมูล (User, Listing, Category)
│   │   ├── auth/              # JWT, bcrypt, middleware
│   │   ├── repository/        # เข้าถึง DB โดยตรง (SQL อยู่ที่นี่ที่เดียว)
│   │   ├── service/           # business logic + validation
│   │   ├── handler/           # แปลง HTTP request/response
│   │   └── router/            # รวม route ทั้งหมด
│   ├── migrations/            # SQL schema
│   └── data/uploads/          # รูปที่อัปโหลด (local disk)
├── ml-service/                 # Python — image embedding สำหรับ similarity search (Phase 2)
│   ├── app/                    # FastAPI service (Embedder abstraction: Mock/CLIP)
│   ├── training/                # เทรน ProjectionHead จริง (ดู ml-service/training/README.md)
│   └── tests/                  # พิสูจน์ pipeline ด้วย numpy ล้วน ไม่ต้องมี torch
└── frontend/
    ├── tailwind.config.js       # สี/ฟอนต์ทั้งหมดตั้งไว้ที่นี่ (theme.extend)
    └── src/
        ├── api/                # เรียก backend (1 ไฟล์ต่อ 1 resource)
        ├── context/            # AuthContext (สถานะ login)
        ├── hooks/              # useConversationSocket (WebSocket)
        ├── components/         # UI ที่ใช้ซ้ำ
        ├── pages/              # 1 ไฟล์ต่อ 1 หน้า (ใช้ Tailwind utility classes)
        └── index.css           # @tailwind directives เท่านั้น ไม่มี custom class แล้ว
```

**หลักการจัดโครงสร้าง (ทำไมถึงแบ่งแบบนี้):**
- Backend เป็น **layered architecture**: `handler → service → repository → database`
  ชั้นบนไม่รู้จักรายละเอียดของชั้นล่าง (เช่น handler ไม่รู้ว่า data เก็บใน Postgres หรือที่ไหน)
  ทำให้แก้/เพิ่มฟีเจอร์โดยไม่กระทบชั้นอื่น และเขียน unit test แต่ละชั้นแยกกันได้ง่าย
- SQL query ทั้งหมดอยู่ใน `repository/` ที่เดียว — ถ้าจะ optimize query หรือเปลี่ยน DB ทีหลัง แก้ที่เดียวจบ
- Frontend แยก `api/` ออกจาก `pages/` ชัดเจน — หน้าไม่ต้องรู้เรื่อง fetch/URL ตรงๆ

## วิธีรัน (ครั้งแรก)

ต้องมี **Go 1.22+**, **Node.js 18+**, และ **Docker** (สำหรับรัน Postgres)

### 1) เปิด Postgres

```bash
docker compose up -d
```

### 2) รัน migration (สร้างตาราง + seed หมวดหมู่)

```bash
docker exec -i dormmarket-postgres psql -U dormmarket -d dormmarket < backend/migrations/0001_init.up.sql
docker exec -i dormmarket-postgres psql -U dormmarket -d dormmarket < backend/migrations/0002_add_chat.up.sql
docker exec -i dormmarket-postgres psql -U dormmarket -d dormmarket < backend/migrations/0003_add_google_auth.up.sql
docker exec -i dormmarket-postgres psql -U dormmarket -d dormmarket < backend/migrations/0004_add_reviews.up.sql
docker exec -i dormmarket-postgres psql -U dormmarket -d dormmarket < backend/migrations/0005_add_shipments.up.sql
docker exec -i dormmarket-postgres psql -U dormmarket -d dormmarket < backend/migrations/0006_add_listing_soft_delete.up.sql
docker exec -i dormmarket-postgres psql -U dormmarket -d dormmarket < backend/migrations/0007_add_image_embeddings.up.sql
```

### 3) รัน backend

```bash
cd backend
go mod tidy
go run ./cmd/api
```

จะรันที่ `http://localhost:8080` (ใช้ค่า default เชื่อม Postgres จาก docker-compose อัตโนมัติ
ถ้าอยากเปลี่ยนค่า ตั้ง environment variable `DATABASE_URL`, `JWT_SECRET`, `PORT` ได้)

### 4) รัน frontend (เปิด terminal อีกอัน)

```bash
cd frontend
npm install
npm run dev
```

เปิด `http://localhost:5173`

## ตั้งค่า Google Login (ไม่บังคับ — ไม่ตั้งก็ใช้อีเมล/รหัสผ่านได้ตามปกติ)

ระบบรองรับ login แบบ **hybrid**: อีเมล/รหัสผ่าน กับ Google ใช้คู่กันได้ ถ้าไม่อยากเปิด
Google Login ก็ข้ามหัวข้อนี้ไปเลย ระบบเดิมทำงานปกติ

### ขั้นที่ 1: ขอ Google Client ID

1. เข้า [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. สร้างโปรเจกต์ใหม่ (หรือใช้โปรเจกต์เดิม)
3. ไปที่ **APIs & Services → Credentials → Create Credentials → OAuth client ID**
4. เลือก Application type = **Web application**
5. ใส่ **Authorized JavaScript origins**: `http://localhost:5173` (หรือ domain จริงตอน deploy)
6. กด Create จะได้ **Client ID** มา (รูปแบบ `xxxxx.apps.googleusercontent.com`)

### ขั้นที่ 2: ตั้งค่าทั้งสองฝั่ง (ต้องใช้ Client ID เดียวกัน)

**Backend** — ตั้ง environment variable ก่อนรัน:
```cmd
set GOOGLE_CLIENT_ID=xxxxx.apps.googleusercontent.com
go run ./cmd/api
```
เห็น log `✅ เปิดใช้งาน Google Login แล้ว` แปลว่าถูกต้อง

**Frontend** — สร้างไฟล์ `frontend/.env` (คัดลอกจาก `.env.example`):
```
VITE_GOOGLE_CLIENT_ID=xxxxx.apps.googleusercontent.com
```
แล้ว restart `npm run dev` ใหม่ (Vite อ่านค่า env แค่ตอน start เท่านั้น)

### วิธีทำงาน (สรุปสั้นๆ)

1. Frontend ใช้ Google Identity Services แสดงปุ่ม "Sign in with Google" ได้ **ID token** (JWT ที่ Google เซ็นมาให้) กลับมา ไม่ต้อง redirect ไปมา
2. ส่ง ID token ไปที่ `POST /api/auth/google`
3. Backend verify signature ของ token กับ public key ของ Google เอง (เขียนเอง ไม่พึ่ง library ภายนอก ดู `internal/auth/google.go`) แล้ว find-or-create user
4. ถ้าอีเมลตรงกับบัญชีที่เคยสมัครด้วยรหัสผ่านไว้ก่อน จะ**ผูกให้อัตโนมัติ** — login ทางไหนก็เข้าบัญชีเดิม

**ทดสอบแล้วครบ:** unit test ของ Google token verification (signature ปลอม, aud ไม่ตรง, token หมดอายุ, อีเมลยังไม่ verify) และ business logic (สร้างบัญชีใหม่, ผูกบัญชีเดิม, login ซ้ำได้ user เดิม) ผ่านหมด — ดูรายละเอียดใน `internal/auth/google_test.go` และ `internal/service/auth_service_test.go`

## Testing

Unit test ครอบคลุม **service layer** (business logic ทั้งหมด: validation, ownership check,
default filter logic, chat authorization) และ **auth package** (JWT generate/parse, password
hashing) โดยใช้ fake in-memory repository แทน Postgres จริง — รันเร็ว ไม่ต้องมี DB ก็เทสได้

```bash
cd backend
go test ./... -v          # รันทุกเทส แบบละเอียด
go test ./... -cover      # ดู coverage
```

Handler/repository/router ยังไม่มี test เพราะต้องพึ่ง HTTP request จริง/DB จริง
(เหมาะกับ integration test แยกต่างหาก ไม่ใช่ unit test) — ถ้าต้องการเพิ่มทีหลัง
แนะนำใช้ `httptest` package สำหรับ handler และ Docker test container สำหรับ repository

## API Endpoints (Phase 1)

| Method | Path | ต้อง login | คำอธิบาย |
|---|---|---|---|
| POST | `/api/auth/register` | ❌ | สมัครสมาชิก |
| POST | `/api/auth/login` | ❌ | เข้าสู่ระบบ |
| POST | `/api/auth/google` | ❌ | เข้าสู่ระบบ/สมัครด้วย Google ID token (ดูหัวข้อ "ตั้งค่า Google Login") |
| GET | `/api/auth/me` | ✅ | ข้อมูลตัวเอง |
| GET | `/api/categories` | ❌ | รายการหมวดหมู่ |
| GET | `/api/listings` | ❌ | ค้นหา/กรองประกาศ (`?search=&categoryId=&sellerId=&minPrice=&maxPrice=`) |
| GET | `/api/listings/suggest-price` | ❌ | แนะนำราคา (`?categoryId=&condition=`) — rule-based, ต้องมีข้อมูลอย่างน้อย 3 รายการในหมวด+สภาพเดียวกัน |
| POST | `/api/listings/search-by-image` | ❌ | ค้นหาประกาศด้วยรูปภาพ (ต้องตั้งค่า `ML_SERVICE_URL` ก่อน) |
| GET | `/api/listings/{id}` | ❌ | รายละเอียดประกาศ |
| POST | `/api/listings` | ✅ | สร้างประกาศใหม่ |
| PUT | `/api/listings/{id}` | ✅ (เจ้าของเท่านั้น) | แก้ไขประกาศ |
| DELETE | `/api/listings/{id}` | ✅ (เจ้าของเท่านั้น) | ลบประกาศ (soft delete — ซ่อนจากรายการ ข้อมูลเดิมไม่หาย) |
| POST | `/api/listings/{id}/images` | ✅ (เจ้าของเท่านั้น) | อัปโหลดรูป |
| PATCH | `/api/listings/{id}/status` | ✅ (เจ้าของเท่านั้น) | เปลี่ยนสถานะ (available/reserved/sold) |
| POST | `/api/reviews` | ✅ | เขียนรีวิวผู้ขาย (ต้องซื้อขายสำเร็จ + เคยทักแชทจริง) |
| GET | `/api/users/{id}/reviews` | ❌ | ดูรีวิวทั้งหมดที่ user คนนี้เคยได้รับ |
| GET | `/api/listings/{id}/can-review` | ✅ | เช็คว่ามีสิทธิ์รีวิว listing นี้ไหม (โชว์/ซ่อนปุ่มฝั่ง frontend) |
| POST | `/api/conversations/{id}/shipment` | ✅ (ผู้ขายเท่านั้น) | เริ่มติดตามการส่งมอบสินค้า (`method: pickup\|delivery`) |
| GET | `/api/conversations/{id}/shipment` | ✅ (คู่สนทนาเท่านั้น) | ดูสถานะ + timeline การจัดส่ง |
| PATCH | `/api/conversations/{id}/shipment/status` | ✅ (ผู้ขายเท่านั้น) | อัปเดตสถานะ (pending → shipped/completed → cancelled) |

ทดสอบผ่านแล้วด้วย `curl` ครบทุก endpoint รวมถึง validation error (อีเมลซ้ำ, รหัสผ่านสั้นเกินไป,
สร้างประกาศโดยไม่ login ฯลฯ)

## แก้ไข/ลบประกาศ + ประวัติการซื้อสินค้า

**แก้ไข/ลบประกาศ** — เฉพาะเจ้าของ ลบเป็น **soft delete** (คอลัมน์ `deleted_at`) ไม่ใช่ลบจริง
เพราะ hard delete จะทำลายประวัติแชท/รีวิว/shipment ที่ผูกกับ listing นั้นไปด้วย (FK cascade)
ประกาศที่ถูกลบจะหายจากหน้าค้นหา/รายการทันที แต่ยังเข้าถึงตรงๆ ผ่าน ID ได้ (จำเป็นสำหรับหน้าแชท/
ประวัติการซื้อที่อ้างอิงถึง listing นั้นอยู่)

**ประวัติการซื้อสินค้า** — ไม่มี endpoint ใหม่ ใช้ข้อมูล `GET /api/conversations` เดิมที่มีอยู่แล้ว
กรองฝั่ง frontend เอา (เฉพาะ conversation ที่ user เป็นผู้ซื้อ) แบ่งเป็น "ซื้อสำเร็จแล้ว"
(listing สถานะ `sold`) กับ "กำลังดำเนินการ" (สถานะอื่น) — เลือกออกแบบแบบนี้เพราะระบบยังไม่มี
แนวคิด "order" แยกจาก listing โดยตรง การอ้างอิงจาก conversation จึงเพียงพอและไม่ต้องเพิ่ม
ตารางใหม่

## ระบบติดตามการส่งมอบสินค้า (Shipment Tracking)

เป็นระบบ **manual** — ผู้ขายกรอก/อัปเดตสถานะเอง ไม่ได้เชื่อมกับ API ขนส่งจริง (Kerry/Flash/
ไปรษณีย์ไทย ฯลฯ) เพราะแต่ละเจ้าต้องขอ API key ผ่าน business account ต่างหาก และแซนด์บ็อกซ์ที่
พัฒนาโค้ดนี้เรียก API ภายนอกพวกนี้ไม่ได้ — ถ้าอยากต่อยอดเป็นแบบดึงสถานะจริงอัตโนมัติทีหลัง
ทำได้โดยเพิ่ม adapter เรียก API ของขนส่งแต่ละเจ้า มาแทนที่จุดที่ผู้ขายกดอัปเดตสถานะเอง

**กลไก:**
- 1 conversation (คู่ผู้ซื้อ-ผู้ขาย-สินค้า) มีได้แค่ 1 shipment
- เลือกวิธีได้ 2 แบบ: **นัดรับเอง (pickup)** หรือ **ส่งขนส่ง (delivery** — ต้องกรอกชื่อขนส่ง+เลข
  tracking)
- สถานะ: `pending` → `shipped` (เฉพาะ delivery) → `completed`, หรือยกเลิกเป็น `cancelled` ได้
  จาก `pending`/`shipped`
- ทุกครั้งที่เปลี่ยนสถานะ จะบันทึกเป็น event เก็บ timeline ให้ทั้งผู้ซื้อและผู้ขายดูย้อนหลังได้
- เฉพาะผู้ขายอัปเดตสถานะได้ ผู้ซื้อดูอย่างเดียว

แสดงผลใน `ChatPage` — ฝั่งผู้ขายเห็นฟอร์มเริ่มติดตาม + ปุ่มเปลี่ยนสถานะ ฝั่งผู้ซื้อเห็น timeline
อย่างเดียว ทดสอบแล้วครบทั้ง unit test (10 เคส ครอบคลุมทุกเงื่อนไข) และ end-to-end กับ Postgres
จริง

## Reviews & Trust Score

แก้ปัญหาที่ `trust_score` เดิมทุกคนติดค่า default 100 ตายตัว ไม่เคยเปลี่ยนเลย ตอนนี้คำนวณจาก
รีวิวจริงแล้ว:

- รีวิวได้เฉพาะตอน listing สถานะ **sold** และผู้รีวิวต้อง**เคยทักแชทกับผู้ขายจริง** (เช็คจาก
  ตาราง `conversations`) — กันรีวิวปลอมจากคนที่ไม่เกี่ยวข้องกับการซื้อขายจริง
- 1 คนรีวิวได้ 1 ครั้งต่อ 1 ประกาศเท่านั้น
- สูตรคำนวณ: `trust_score = round(ค่าเฉลี่ยดาวทั้งหมด / 5 * 100)` เช่น เฉลี่ย 4 ดาว → trust score 80
- คำนวณใหม่ทันทีทุกครั้งที่มีรีวิวใหม่เข้ามา (ไม่ใช่ batch job รันทีหลัง)

ทดสอบแล้วครบทั้ง unit test (fake repository, ครอบคลุมทุกเงื่อนไข) และ end-to-end กับ Postgres
จริง (สร้าง user จริง 2 คน → ทักแชท → mark ขาย → รีวิว → เช็คว่า trust score เปลี่ยนจริงตาม
สูตรที่คำนวณด้วย SQL `ROUND()` ตรงกับที่ทดสอบไว้)

## Phase 2 — สถานะปัจจุบัน

### ✅ Price Suggestion (เสร็จแล้ว, rule-based)

`GET /api/listings/suggest-price?categoryId=X&condition=Y` คำนวณราคาเฉลี่ยจากประกาศอื่น
ในหมวดหมู่+สภาพเดียวกันที่มีอยู่แล้ว ต้องมีอย่างน้อย 3 รายการถึงจะแนะนำ (กัน cold start —
ข้อมูลน้อยเกินไปแนะนำราคาจะไม่น่าเชื่อถือ) ตอนสร้างประกาศใหม่ ระบบจะเติม `suggestedPrice`
ให้อัตโนมัติถ้ามีข้อมูลพอ

### ✅ Image Similarity Search (เสร็จสมบูรณ์ — เทรนจริงแล้ว + เชื่อม backend แล้ว)

อยู่ใน `ml-service/` — สถาปัตยกรรม: **frozen CLIP (pretrained) + ProjectionHead ที่เทรนเอง**
ด้วย Supervised Contrastive Loss (ดูรายละเอียดเต็มใน `ml-service/README.md` และ
`ml-service/training/README.md`)

- ✅ FastAPI service พร้อม `/embed` endpoint
- ✅ Training pipeline เต็มรูปแบบ (PyTorch) — **เทรนจริงแล้วด้วย Kaggle dataset**
  (Fashion Product Images) ผลลัพธ์: intra-inter similarity gap ดีขึ้นจาก 0.08 → 1.04
  (ดีขึ้นเกือบ 13 เท่า) ได้ไฟล์ `training/projection_head.pt`
- ✅ เชื่อมกับ Go backend สมบูรณ์:
  - `listing_images.embedding vector(256)` (migration `0007_add_image_embeddings.up.sql`)
    ใช้ **pgvector** extension — เก็บ embedding ของทุกรูปที่อัปโหลด
  - `internal/mlservice/` — HTTP client เรียก ml-service ตอนอัปโหลดรูป (แบบ **best-effort**:
    ถ้า ml-service ไม่ได้ตั้งค่าไว้หรือเรียกไม่สำเร็จ การอัปโหลดรูปยังสำเร็จอยู่ปกติ แค่ไม่มี
    embedding ให้ค้นหาด้วยรูปสำหรับรูปนั้น)
  - `POST /api/listings/search-by-image` — อัปโหลดรูป 1 รูป คืนประกาศที่มีรูปคล้ายที่สุด
    (เรียง cosine distance จาก pgvector, กรองเฉพาะประกาศที่ยัง available และไม่ถูกลบ)
  - Frontend: ปุ่มกล้อง 📷 ข้างช่องค้นหาในหน้าแรก อัปโหลดรูปแล้วเห็นผลลัพธ์ทันที

**ทดสอบครบทุกระดับ:**
- Unit test (fake embedder client) — คำนวณ embedding สำเร็จ, ล้มเหลวแบบ best-effort ไม่กระทบ
  การอัปโหลด, ปิดใช้งานถ้าไม่ตั้งค่า, ค้นหารูปเดียวกันเป๊ะต้องได้ประกาศเดิมเป็นอันดับ 1,
  ไม่รวมประกาศที่ลบ/ขายแล้ว
- **End-to-end เต็มรูปแบบกับ Postgres + pgvector จริง + ml-service จริง**: อัปโหลดรูปเก้าอี้
  สีต่างกัน 2 ประกาศ → ค้นหาด้วยรูปสีเดียวกันเป๊ะ → ได้ประกาศที่ตรงเป็นอันดับ 1 จริง, ตรวจสอบ
  ค่า `vector_dims(embedding) = 256` ในฐานข้อมูลตรงตามที่ออกแบบ

**⚠️ สำคัญมากก่อนรันที่เครื่องคุณ:** `docker-compose.yml` เปลี่ยน image จาก `postgres:16-alpine`
เป็น **`pgvector/pgvector:pg16`** (มี extension `vector` ติดตั้งมาให้แล้ว) — ถ้า container เดิม
ที่รันอยู่ยังเป็น image เก่า ต้องรัน `docker compose down -v` (ลบ volume เดิมทิ้ง เพราะเป็นข้อมูล
ทดสอบ ไม่มีอะไรสำคัญ) แล้ว `docker compose up -d` ใหม่ จากนั้น **รัน migration ทั้งหมดใหม่ตั้งแต่
0001 ถึง 0007** เพราะเป็นฐานข้อมูลใหม่

**การตั้งค่าที่ต้องทำก่อนใช้งานจริง:**
```bash
# รัน ml-service (ต้องเทรน projection head ไว้ก่อน ดู ml-service/training/README.md)
cd ml-service
export EMBEDDER_MODE=clip
export PROJECTION_HEAD_PATH=training/projection_head.pt
uvicorn app.main:app --port 8001

# ตั้งค่า backend ให้รู้จัก ml-service
cd backend
export ML_SERVICE_URL=http://localhost:8001
go run ./cmd/api
```
เห็น log `✅ เปิดใช้งาน image similarity search แล้ว` แปลว่าเชื่อมสำเร็จ ถ้าไม่ได้ตั้งค่า
`ML_SERVICE_URL` ไว้ ระบบลงประกาศ/ค้นหาปกติยังใช้งานได้เหมือนเดิมทุกอย่าง แค่ปุ่ม "ค้นหาด้วยรูป"
จะไม่ทำงาน (ตอบ 501)

### ⏳ ยังไม่เริ่ม

- **Fraud detection** — เริ่มจาก rule-based scoring ก่อน (ราคาต่ำผิดปกติ, บัญชีใหม่, รูปซ้ำ —
  ใช้ image embedding ที่มีอยู่แล้วเช็ค duplicate ได้เลย)

### ✅ Real-time Chat (เสร็จแล้ว ทดสอบ end-to-end จริงแล้ว)

WebSocket ผูกกับแต่ละ conversation (1 conversation ต่อ 1 คู่ listing+buyer) — ทดสอบแล้วด้วย
WebSocket client จริง 2 ตัวพร้อมกัน (จำลองผู้ซื้อ+ผู้ขาย) ยืนยันว่า broadcast แบบ real-time
ทำงานถูกต้อง รวมถึงเทส security ครบ (คนนอกอ่าน/ส่งข้อความในห้องที่ไม่เกี่ยวข้องไม่ได้
ทั้งฝั่ง REST และ WebSocket)

**Endpoints เพิ่มเติม:**

| Method | Path | ต้อง login | คำอธิบาย |
|---|---|---|---|
| POST | `/api/conversations` | ✅ | เริ่ม/เปิดห้องแชทกับผู้ขาย (`{listingId}`) — idempotent |
| GET | `/api/conversations` | ✅ | รายการห้องแชททั้งหมด (inbox) |
| GET | `/api/conversations/{id}/messages` | ✅ (คู่สนทนาเท่านั้น) | ประวัติแชท |
| GET | `/ws/conversations/{id}?token=...` | ✅ (คู่สนทนาเท่านั้น) | WebSocket — ส่ง/รับข้อความ real-time |

**หมายเหตุ:** WebSocket auth ผ่าน query param `token` แทน `Authorization` header เพราะ
browser WebSocket API มาตรฐานตั้งค่า custom header ตอน handshake ไม่ได้ — เป็นข้อจำกัดของ
WebSocket spec เอง

**ข้อจำกัดที่ควรรู้ก่อนใช้งานจริงระดับ production:** Hub เก็บ connection ไว้ใน memory ของ
โปรเซสเดียว (`internal/ws/hub.go`) ถ้า deploy หลาย instance พร้อมกัน (horizontal scaling)
ข้อความจะ broadcast ไปหาแค่ client ที่เชื่อมกับ instance เดียวกัน ต้องเปลี่ยนไปใช้ Redis
pub/sub แทนถ้าจะ scale เกิน 1 instance

## ทำไม schema ถึงมี column ที่ยังไม่ใช้ (เช่น `suggested_price`)

ตั้งใจเผื่อไว้ตั้งแต่ Phase 1 เพื่อไม่ต้อง migrate schema แบบ breaking change ทีหลัง
(`suggested_price` ใช้งานจริงแล้วตอนนี้ ส่วน embedding column ของ `listing_images` ยังไม่ได้เพิ่ม
เพราะยังไม่รู้ขนาด vector ที่แน่นอน — จะเพิ่มตอนเชื่อม ml-service เข้ากับ backend จริง)
