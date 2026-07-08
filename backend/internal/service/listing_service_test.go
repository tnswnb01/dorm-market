package service

import (
	"context"
	"errors"
	"testing"

	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

func newTestListingService() (*ListingService, *fakeListingRepository) {
	listings := newFakeListingRepository()
	categories := &fakeCategoryRepository{categories: []models.Category{
		{ID: "cat-1", Name: "หนังสือ", Slug: "books"},
	}}
	return NewListingService(listings, categories), listings
}

func TestListingService_Create(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateListingInput
		wantErr error
	}{
		{
			name: "สร้างสำเร็จ",
			input: CreateListingInput{
				SellerID: "user-1", CategoryID: "cat-1",
				Title: "โต๊ะเรียนไม้", Price: 450, Condition: models.ConditionGood,
			},
			wantErr: nil,
		},
		{
			name: "ไม่ใส่ชื่อสินค้า",
			input: CreateListingInput{
				SellerID: "user-1", CategoryID: "cat-1", Title: "   ", Price: 450,
			},
			wantErr: ErrTitleRequired,
		},
		{
			name: "ราคาเป็น 0",
			input: CreateListingInput{
				SellerID: "user-1", CategoryID: "cat-1", Title: "ของบางอย่าง", Price: 0,
			},
			wantErr: ErrInvalidPrice,
		},
		{
			name: "ราคาติดลบ",
			input: CreateListingInput{
				SellerID: "user-1", CategoryID: "cat-1", Title: "ของบางอย่าง", Price: -100,
			},
			wantErr: ErrInvalidPrice,
		},
		{
			name: "ไม่เลือกหมวดหมู่",
			input: CreateListingInput{
				SellerID: "user-1", Title: "ของบางอย่าง", Price: 100,
			},
			wantErr: ErrInvalidCategory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestListingService()
			listing, err := svc.Create(context.Background(), tt.input)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ต้องการ error %v แต่ได้ %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("ไม่ควร error: %v", err)
			}
			if listing.Status != models.StatusAvailable {
				t.Errorf("สถานะเริ่มต้นต้องเป็น available ได้ %s", listing.Status)
			}
			if listing.ID == "" {
				t.Error("ต้องมี ID หลังสร้างสำเร็จ")
			}
		})
	}

	t.Run("ไม่ระบุ condition จะ default เป็น good", func(t *testing.T) {
		svc, _ := newTestListingService()
		listing, err := svc.Create(context.Background(), CreateListingInput{
			SellerID: "user-1", CategoryID: "cat-1", Title: "ของบางอย่าง", Price: 100,
		})
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if listing.Condition != models.ConditionGood {
			t.Errorf("condition default ต้องเป็น good ได้ %s", listing.Condition)
		}
	})
}

func TestListingService_Get(t *testing.T) {
	svc, _ := newTestListingService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, CreateListingInput{
		SellerID: "user-1", CategoryID: "cat-1", Title: "โต๊ะเรียนไม้", Price: 450,
	})

	t.Run("หาเจอ", func(t *testing.T) {
		got, err := svc.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if got.Title != "โต๊ะเรียนไม้" {
			t.Errorf("ได้ listing ผิดตัว")
		}
	})

	t.Run("ไม่พบ", func(t *testing.T) {
		_, err := svc.Get(ctx, "ไม่มีจริง")
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("ต้องการ ErrNotFound แต่ได้ %v", err)
		}
	})
}

// นี่คือ business logic ที่สำคัญที่สุดของ ListingService: ตอนดูตลาดรวม (browse)
// ต้องเห็นแค่ของที่ยังขายอยู่ (available) แต่ตอนเจ้าของเปิดดู "ประกาศของฉัน" (ระบุ SellerID)
// ต้องเห็นทุกสถานะ ไม่งั้นของที่ขายไปแล้วจะหายไปจากหน้าประวัติของตัวเอง
func TestListingService_List_DefaultStatusFilter(t *testing.T) {
	svc, repo := newTestListingService()
	ctx := context.Background()

	available, _ := svc.Create(ctx, CreateListingInput{
		SellerID: "user-1", CategoryID: "cat-1", Title: "ของที่ยังขายอยู่", Price: 100,
	})
	sold, _ := svc.Create(ctx, CreateListingInput{
		SellerID: "user-1", CategoryID: "cat-1", Title: "ของที่ขายไปแล้ว", Price: 200,
	})
	if err := repo.UpdateStatus(ctx, sold.ID, "user-1", models.StatusSold); err != nil {
		t.Fatalf("เตรียมข้อมูลเทสไม่สำเร็จ: %v", err)
	}

	t.Run("browse ทั่วไป (ไม่ระบุ filter) เห็นแค่ available", func(t *testing.T) {
		got, err := svc.List(ctx, models.ListingFilter{})
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if len(got) != 1 || got[0].ID != available.ID {
			t.Errorf("ควรเห็นแค่ประกาศที่ available 1 รายการ ได้ %d รายการ", len(got))
		}
	})

	t.Run("ระบุ SellerID (ประกาศของฉัน) เห็นทุกสถานะ", func(t *testing.T) {
		got, err := svc.List(ctx, models.ListingFilter{SellerID: "user-1"})
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("ควรเห็นทั้ง 2 รายการไม่ว่าสถานะไหน ได้ %d รายการ", len(got))
		}
	})

	t.Run("ระบุ Status ตรงๆ จะไม่ถูก override โดย default", func(t *testing.T) {
		got, err := svc.List(ctx, models.ListingFilter{Status: models.StatusSold})
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if len(got) != 1 || got[0].ID != sold.ID {
			t.Errorf("ควรเห็นแค่ประกาศที่ sold 1 รายการ ได้ %d รายการ", len(got))
		}
	})
}

func TestListingService_AddImage(t *testing.T) {
	svc, _ := newTestListingService()
	ctx := context.Background()

	listing, _ := svc.Create(ctx, CreateListingInput{
		SellerID: "owner-1", CategoryID: "cat-1", Title: "โต๊ะเรียนไม้", Price: 450,
	})

	t.Run("เจ้าของเพิ่มรูปได้", func(t *testing.T) {
		err := svc.AddImage(ctx, listing.ID, "owner-1", []byte("fake-image-bytes"), "a.jpg", "/uploads/a.jpg", 0)
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
	})

	t.Run("คนอื่นเพิ่มรูปไม่ได้", func(t *testing.T) {
		err := svc.AddImage(ctx, listing.ID, "someone-else", []byte("x"), "b.jpg", "/uploads/b.jpg", 0)
		if !errors.Is(err, ErrNotOwner) {
			t.Fatalf("ต้องการ ErrNotOwner แต่ได้ %v", err)
		}
	})

	t.Run("listing ไม่มีอยู่จริง", func(t *testing.T) {
		err := svc.AddImage(ctx, "ไม่มีจริง", "owner-1", []byte("x"), "c.jpg", "/uploads/c.jpg", 0)
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("ต้องการ ErrNotFound แต่ได้ %v", err)
		}
	})
}

// เทสส่วนที่เชื่อมกับ ml-service (image similarity search) — ใช้ fakeEmbedderClient
// แทน ml-service จริง เพราะสภาพแวดล้อมที่รันเทสนี้เรียก ml-service จริงไม่ได้
func TestListingService_AddImage_ComputesEmbedding(t *testing.T) {
	listings := newFakeListingRepository()
	categories := &fakeCategoryRepository{categories: []models.Category{{ID: "cat-1", Name: "หนังสือ", Slug: "books"}}}
	embedder := &fakeEmbedderClient{}
	svc := NewListingService(listings, categories).WithEmbedder(embedder)
	ctx := context.Background()

	listing, _ := svc.Create(ctx, CreateListingInput{
		SellerID: "owner-1", CategoryID: "cat-1", Title: "โต๊ะ", Price: 100,
	})

	err := svc.AddImage(ctx, listing.ID, "owner-1", []byte("some-image-data"), "photo.jpg", "/uploads/photo.jpg", 0)
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if embedder.callCount != 1 {
		t.Errorf("ควรเรียก embedder 1 ครั้ง เรียกไปจริง %d ครั้ง", embedder.callCount)
	}

	images := listings.images[listing.ID]
	if len(images) != 1 {
		t.Fatalf("ควรมีรูป 1 รูป")
	}
	if _, ok := listings.embeddings[images[0].ID]; !ok {
		t.Error("ควรบันทึก embedding ของรูปไว้ด้วย")
	}
}

// ถ้า ml-service เรียกไม่สำเร็จ (network ล่ม ฯลฯ) การอัปโหลดรูปต้องยังสำเร็จอยู่
// (ฟีเจอร์หลักคือ "มีรูปในประกาศ" ไม่ควรพังเพราะ AI feature เสริมเรียกไม่ได้)
func TestListingService_AddImage_EmbedderFailureDoesNotBlockUpload(t *testing.T) {
	listings := newFakeListingRepository()
	categories := &fakeCategoryRepository{categories: []models.Category{{ID: "cat-1", Name: "หนังสือ", Slug: "books"}}}
	embedder := &fakeEmbedderClient{err: errors.New("ml-service ล่ม")}
	svc := NewListingService(listings, categories).WithEmbedder(embedder)
	ctx := context.Background()

	listing, _ := svc.Create(ctx, CreateListingInput{
		SellerID: "owner-1", CategoryID: "cat-1", Title: "โต๊ะ", Price: 100,
	})

	err := svc.AddImage(ctx, listing.ID, "owner-1", []byte("some-image-data"), "photo.jpg", "/uploads/photo.jpg", 0)
	if err != nil {
		t.Fatalf("embedder พังไม่ควรทำให้ AddImage fail: %v", err)
	}

	images := listings.images[listing.ID]
	if len(images) != 1 {
		t.Fatalf("รูปควรถูกบันทึกอยู่ดี แม้ embedder จะพัง")
	}
}

func TestListingService_SearchByImage(t *testing.T) {
	t.Run("ปิดใช้งานถ้าไม่ได้ตั้งค่า embedder", func(t *testing.T) {
		listings := newFakeListingRepository()
		categories := &fakeCategoryRepository{}
		svc := NewListingService(listings, categories) // ไม่เรียก WithEmbedder

		_, err := svc.SearchByImage(context.Background(), []byte("x"), "x.jpg")
		if !errors.Is(err, ErrImageSearchDisabled) {
			t.Fatalf("ต้องการ ErrImageSearchDisabled แต่ได้ %v", err)
		}
	})

	t.Run("ค้นหารูปเดียวกันเป๊ะต้องเจอประกาศเดิมเป็นอันดับ 1", func(t *testing.T) {
		listings := newFakeListingRepository()
		categories := &fakeCategoryRepository{categories: []models.Category{{ID: "cat-1", Name: "หนังสือ", Slug: "books"}}}
		embedder := &fakeEmbedderClient{}
		svc := NewListingService(listings, categories).WithEmbedder(embedder)
		ctx := context.Background()

		target, _ := svc.Create(ctx, CreateListingInput{SellerID: "s1", CategoryID: "cat-1", Title: "เป้าหมาย", Price: 100})
		svc.AddImage(ctx, target.ID, "s1", []byte("target-image-content"), "t.jpg", "/uploads/t.jpg", 0)

		other, _ := svc.Create(ctx, CreateListingInput{SellerID: "s2", CategoryID: "cat-1", Title: "อื่นๆ", Price: 200})
		svc.AddImage(ctx, other.ID, "s2", []byte("completely-different-content-xyz"), "o.jpg", "/uploads/o.jpg", 0)

		// ค้นหาด้วยรูปเดียวกับที่ "เป้าหมาย" ใช้เป๊ะ — fakeEmbedderClient deterministic
		// ตาม byte ของรูป รูปเดียวกัน = embedding เดียวกัน = คะแนนคล้ายสูงสุด
		results, err := svc.SearchByImage(ctx, []byte("target-image-content"), "query.jpg")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("ควรเจอผลลัพธ์อย่างน้อย 1 รายการ")
		}
		if results[0].ID != target.ID {
			t.Errorf("อันดับ 1 ควรเป็นประกาศที่ใช้รูปเดียวกัน ได้ %s แทน", results[0].Title)
		}
	})

	t.Run("ไม่รวมประกาศที่ถูกลบหรือขายไปแล้ว", func(t *testing.T) {
		listings := newFakeListingRepository()
		categories := &fakeCategoryRepository{categories: []models.Category{{ID: "cat-1", Name: "หนังสือ", Slug: "books"}}}
		embedder := &fakeEmbedderClient{}
		svc := NewListingService(listings, categories).WithEmbedder(embedder)
		ctx := context.Background()

		sold, _ := svc.Create(ctx, CreateListingInput{SellerID: "s1", CategoryID: "cat-1", Title: "ขายแล้ว", Price: 100})
		svc.AddImage(ctx, sold.ID, "s1", []byte("same-content"), "a.jpg", "/uploads/a.jpg", 0)
		svc.UpdateStatus(ctx, sold.ID, "s1", models.StatusSold)

		results, err := svc.SearchByImage(ctx, []byte("same-content"), "query.jpg")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		for _, r := range results {
			if r.ID == sold.ID {
				t.Error("ไม่ควรเจอประกาศที่ขายไปแล้วในผลค้นหา")
			}
		}
	})
}

func TestListingService_UpdateStatus(t *testing.T) {
	svc, _ := newTestListingService()
	ctx := context.Background()

	listing, _ := svc.Create(ctx, CreateListingInput{
		SellerID: "owner-1", CategoryID: "cat-1", Title: "โต๊ะเรียนไม้", Price: 450,
	})

	t.Run("เจ้าของเปลี่ยนสถานะได้", func(t *testing.T) {
		err := svc.UpdateStatus(ctx, listing.ID, "owner-1", models.StatusSold)
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
	})

	t.Run("คนอื่นเปลี่ยนสถานะไม่ได้", func(t *testing.T) {
		err := svc.UpdateStatus(ctx, listing.ID, "someone-else", models.StatusReserved)
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("ต้องการ ErrNotFound (ไม่บอกตรงๆ ว่า listing มีอยู่แต่ไม่ใช่เจ้าของ) แต่ได้ %v", err)
		}
	})
}

func TestListingService_ListCategories(t *testing.T) {
	svc, _ := newTestListingService()
	got, err := svc.ListCategories(context.Background())
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "books" {
		t.Errorf("ได้หมวดหมู่ไม่ตรงกับที่ seed ไว้")
	}
}

func TestListingService_Update(t *testing.T) {
	svc, _ := newTestListingService()
	ctx := context.Background()

	listing, _ := svc.Create(ctx, CreateListingInput{
		SellerID: "owner-1", CategoryID: "cat-1", Title: "โต๊ะเก่า", Price: 100,
	})

	t.Run("เจ้าของแก้ไขได้", func(t *testing.T) {
		updated, err := svc.Update(ctx, UpdateListingInput{
			ID: listing.ID, SellerID: "owner-1", CategoryID: "cat-1",
			Title: "โต๊ะเก่า (ลดราคา)", Price: 80, Condition: models.ConditionWorn,
		})
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if updated.Title != "โต๊ะเก่า (ลดราคา)" || updated.Price != 80 {
			t.Errorf("ข้อมูลหลังแก้ไขไม่ตรง: %+v", updated)
		}
	})

	t.Run("คนอื่นแก้ไขไม่ได้", func(t *testing.T) {
		_, err := svc.Update(ctx, UpdateListingInput{
			ID: listing.ID, SellerID: "someone-else", CategoryID: "cat-1", Title: "แอบแก้", Price: 1,
		})
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("ต้องการ ErrNotFound แต่ได้ %v", err)
		}
	})

	t.Run("validation เดียวกับตอนสร้าง", func(t *testing.T) {
		_, err := svc.Update(ctx, UpdateListingInput{
			ID: listing.ID, SellerID: "owner-1", CategoryID: "cat-1", Title: "", Price: 80,
		})
		if !errors.Is(err, ErrTitleRequired) {
			t.Fatalf("ต้องการ ErrTitleRequired แต่ได้ %v", err)
		}
	})

	t.Run("ประกาศไม่มีอยู่จริง", func(t *testing.T) {
		_, err := svc.Update(ctx, UpdateListingInput{
			ID: "ไม่มีจริง", SellerID: "owner-1", CategoryID: "cat-1", Title: "x", Price: 1,
		})
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("ต้องการ ErrNotFound แต่ได้ %v", err)
		}
	})
}

func TestListingService_Delete(t *testing.T) {
	svc, _ := newTestListingService()
	ctx := context.Background()

	listing, _ := svc.Create(ctx, CreateListingInput{
		SellerID: "owner-1", CategoryID: "cat-1", Title: "ของที่จะลบ", Price: 100,
	})

	t.Run("คนอื่นลบไม่ได้", func(t *testing.T) {
		err := svc.Delete(ctx, listing.ID, "someone-else")
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("ต้องการ ErrNotFound แต่ได้ %v", err)
		}
	})

	t.Run("เจ้าของลบได้ และหายไปจากรายการค้นหา", func(t *testing.T) {
		err := svc.Delete(ctx, listing.ID, "owner-1")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}

		results, err := svc.List(ctx, models.ListingFilter{SellerID: "owner-1"})
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		for _, l := range results {
			if l.ID == listing.ID {
				t.Error("ประกาศที่ถูกลบไม่ควรโผล่ในรายการอีก")
			}
		}
	})

	t.Run("ลบซ้ำไม่ได้ (idempotent guard)", func(t *testing.T) {
		err := svc.Delete(ctx, listing.ID, "owner-1")
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("ลบไปแล้วครั้งที่สองควรได้ ErrNotFound แต่ได้ %v", err)
		}
	})
}

// เทส cold-start threshold: ต้องมีข้อมูลอย่างน้อย minSamplesForSuggestion รายการ
// ในหมวดหมู่+สภาพเดียวกัน ถึงจะกล้าแนะนำราคา ไม่งั้นข้อมูลน้อยเกินไปจะแนะนำมั่ว
func TestListingService_SuggestPrice(t *testing.T) {
	svc, _ := newTestListingService()
	ctx := context.Background()

	t.Run("ข้อมูลน้อยเกินไป (ยังไม่มีประกาศเลย) ไม่แนะนำราคา", func(t *testing.T) {
		got, err := svc.SuggestPrice(ctx, "cat-1", models.ConditionGood)
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if got != nil {
			t.Errorf("ข้อมูลน้อยเกินไปควรได้ nil (ไม่ใช่ error) แต่ได้ %+v", got)
		}
	})

	// สร้างประกาศให้ครบ threshold (minSamplesForSuggestion = 3) ในหมวด+สภาพเดียวกัน
	prices := []int{100, 200, 300}
	for _, p := range prices {
		_, err := svc.Create(ctx, CreateListingInput{
			SellerID: "user-1", CategoryID: "cat-1",
			Title: "ของทดสอบ", Price: p, Condition: models.ConditionGood,
		})
		if err != nil {
			t.Fatalf("เตรียมข้อมูลเทสไม่สำเร็จ: %v", err)
		}
	}

	t.Run("ข้อมูลครบ threshold แล้ว แนะนำราคาเฉลี่ยถูกต้อง", func(t *testing.T) {
		got, err := svc.SuggestPrice(ctx, "cat-1", models.ConditionGood)
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if got == nil {
			t.Fatal("ควรได้คำแนะนำราคาแล้ว")
		}
		wantAvg := (100 + 200 + 300) / 3
		if got.SuggestedPrice != wantAvg {
			t.Errorf("ราคาแนะนำผิด ต้องการ %d ได้ %d", wantAvg, got.SuggestedPrice)
		}
		if got.SampleSize != 3 {
			t.Errorf("sample size ต้องเป็น 3 ได้ %d", got.SampleSize)
		}
	})

	t.Run("สภาพสินค้าต่างกัน ไม่เอามาปนกันคำนวณ", func(t *testing.T) {
		got, err := svc.SuggestPrice(ctx, "cat-1", models.ConditionWorn)
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if got != nil {
			t.Errorf("หมวด+สภาพนี้ยังไม่มีข้อมูล ควรได้ nil แต่ได้ %+v", got)
		}
	})

	t.Run("ไม่ระบุหมวดหมู่ ต้อง error", func(t *testing.T) {
		_, err := svc.SuggestPrice(ctx, "", models.ConditionGood)
		if !errors.Is(err, ErrInvalidCategory) {
			t.Fatalf("ต้องการ ErrInvalidCategory แต่ได้ %v", err)
		}
	})
}
