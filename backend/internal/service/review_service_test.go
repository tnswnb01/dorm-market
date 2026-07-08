package service

import (
	"context"
	"errors"
	"testing"

	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

// setupSoldListingWithConversation คือ scenario ที่ใช้ซ้ำหลายเทส: มีประกาศที่ขายแล้ว
// และผู้ซื้อเคยทักแชทกับผู้ขายไว้จริง (เงื่อนไขพื้นฐานที่ทำให้รีวิวได้)
func setupSoldListingWithConversation(t *testing.T) (*ReviewService, *fakeListingRepository, *fakeReviewRepository, *models.Listing) {
	t.Helper()
	ctx := context.Background()

	listings := newFakeListingRepository()
	chat := newFakeChatRepository()
	reviews := newFakeReviewRepository()
	users := newFakeUserRepository()
	svc := NewReviewService(reviews, listings, chat, users)

	listing := &models.Listing{SellerID: "seller-1", CategoryID: "cat-1", Title: "โต๊ะ", Price: 100, Status: models.StatusAvailable}
	if err := listings.Create(ctx, listing); err != nil {
		t.Fatalf("เตรียมข้อมูลไม่สำเร็จ: %v", err)
	}
	if err := listings.UpdateStatus(ctx, listing.ID, "seller-1", models.StatusSold); err != nil {
		t.Fatalf("เตรียมข้อมูลไม่สำเร็จ: %v", err)
	}
	if _, err := chat.GetOrCreateConversation(ctx, listing.ID, "buyer-1", "seller-1"); err != nil {
		t.Fatalf("เตรียมข้อมูลไม่สำเร็จ: %v", err)
	}

	return svc, listings, reviews, listing
}

func TestReviewService_Create(t *testing.T) {
	t.Run("รีวิวสำเร็จ และคำนวณ trust score ใหม่", func(t *testing.T) {
		svc, _, reviews, listing := setupSoldListingWithConversation(t)
		ctx := context.Background()

		review, err := svc.Create(ctx, CreateReviewInput{
			ListingID: listing.ID, ReviewerID: "buyer-1", Rating: 4, Comment: "ดีมาก ของตรงปก",
		})
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if review.RevieweeID != "seller-1" {
			t.Errorf("reviewee ต้องเป็นผู้ขาย ได้ %s", review.RevieweeID)
		}

		if got := reviews.trustScores["seller-1"]; got != 80 { // 4/5*100 = 80
			t.Errorf("trust score ควรเป็น 80 ได้ %d", got)
		}
	})

	t.Run("คะแนนนอกช่วง 1-5 ไม่ผ่าน", func(t *testing.T) {
		svc, _, _, listing := setupSoldListingWithConversation(t)
		ctx := context.Background()

		for _, rating := range []int{0, 6, -1} {
			_, err := svc.Create(ctx, CreateReviewInput{
				ListingID: listing.ID, ReviewerID: "buyer-1", Rating: rating,
			})
			if !errors.Is(err, ErrInvalidRating) {
				t.Errorf("rating=%d ต้องการ ErrInvalidRating แต่ได้ %v", rating, err)
			}
		}
	})

	t.Run("ประกาศยังไม่ขาย รีวิวไม่ได้", func(t *testing.T) {
		listings := newFakeListingRepository()
		chat := newFakeChatRepository()
		reviews := newFakeReviewRepository()
		users := newFakeUserRepository()
		svc := NewReviewService(reviews, listings, chat, users)
		ctx := context.Background()

		listing := &models.Listing{SellerID: "seller-1", CategoryID: "cat-1", Title: "โต๊ะ", Price: 100, Status: models.StatusAvailable}
		listings.Create(ctx, listing)
		chat.GetOrCreateConversation(ctx, listing.ID, "buyer-1", "seller-1")

		_, err := svc.Create(ctx, CreateReviewInput{ListingID: listing.ID, ReviewerID: "buyer-1", Rating: 5})
		if !errors.Is(err, ErrListingNotSold) {
			t.Fatalf("ต้องการ ErrListingNotSold แต่ได้ %v", err)
		}
	})

	t.Run("รีวิวประกาศตัวเองไม่ได้", func(t *testing.T) {
		svc, _, _, listing := setupSoldListingWithConversation(t)
		ctx := context.Background()

		_, err := svc.Create(ctx, CreateReviewInput{ListingID: listing.ID, ReviewerID: "seller-1", Rating: 5})
		if !errors.Is(err, ErrCannotReviewSelf) {
			t.Fatalf("ต้องการ ErrCannotReviewSelf แต่ได้ %v", err)
		}
	})

	t.Run("ไม่เคยทักแชทมาก่อน รีวิวไม่ได้ (กันรีวิวปลอม)", func(t *testing.T) {
		listings := newFakeListingRepository()
		chat := newFakeChatRepository()
		reviews := newFakeReviewRepository()
		users := newFakeUserRepository()
		svc := NewReviewService(reviews, listings, chat, users)
		ctx := context.Background()

		listing := &models.Listing{SellerID: "seller-1", CategoryID: "cat-1", Title: "โต๊ะ", Price: 100, Status: models.StatusAvailable}
		listings.Create(ctx, listing)
		listings.UpdateStatus(ctx, listing.ID, "seller-1", models.StatusSold)
		// สังเกตว่าไม่ได้สร้าง conversation ไว้เลย

		_, err := svc.Create(ctx, CreateReviewInput{ListingID: listing.ID, ReviewerID: "random-person", Rating: 5})
		if !errors.Is(err, ErrNotEligibleReview) {
			t.Fatalf("ต้องการ ErrNotEligibleReview แต่ได้ %v", err)
		}
	})

	t.Run("รีวิวซ้ำประกาศเดิมไม่ได้", func(t *testing.T) {
		svc, _, _, listing := setupSoldListingWithConversation(t)
		ctx := context.Background()

		_, err := svc.Create(ctx, CreateReviewInput{ListingID: listing.ID, ReviewerID: "buyer-1", Rating: 5})
		if err != nil {
			t.Fatalf("รีวิวครั้งแรกไม่ควร error: %v", err)
		}

		_, err = svc.Create(ctx, CreateReviewInput{ListingID: listing.ID, ReviewerID: "buyer-1", Rating: 3})
		if !errors.Is(err, ErrAlreadyReviewed) {
			t.Fatalf("ต้องการ ErrAlreadyReviewed แต่ได้ %v", err)
		}
	})

	t.Run("ประกาศไม่มีอยู่จริง", func(t *testing.T) {
		svc, _, _, _ := setupSoldListingWithConversation(t)
		ctx := context.Background()

		_, err := svc.Create(ctx, CreateReviewInput{ListingID: "ไม่มีจริง", ReviewerID: "buyer-1", Rating: 5})
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("ต้องการ ErrNotFound แต่ได้ %v", err)
		}
	})

	t.Run("รีวิวหลายคน trust score เฉลี่ยถูกต้อง", func(t *testing.T) {
		svc, listings, reviews, _ := setupSoldListingWithConversation(t)
		ctx := context.Background()

		// สร้างอีก 1 ประกาศ ให้ buyer อีกคนซื้อจากผู้ขายเดียวกัน จะได้เทสค่าเฉลี่ยจากรีวิวหลายอัน
		listing2 := &models.Listing{SellerID: "seller-1", CategoryID: "cat-1", Title: "เก้าอี้", Price: 50, Status: models.StatusAvailable}
		listings.Create(ctx, listing2)
		listings.UpdateStatus(ctx, listing2.ID, "seller-1", models.StatusSold)

		svc.chat.GetOrCreateConversation(ctx, listing2.ID, "buyer-2", "seller-1")

		listing1ID := listings.byID
		var listing1 *models.Listing
		for _, l := range listing1ID {
			if l.Title == "โต๊ะ" {
				listing1 = l
			}
		}

		svc.Create(ctx, CreateReviewInput{ListingID: listing1.ID, ReviewerID: "buyer-1", Rating: 4})
		svc.Create(ctx, CreateReviewInput{ListingID: listing2.ID, ReviewerID: "buyer-2", Rating: 2})

		// เฉลี่ย (4+2)/2 = 3 ดาว -> trust score = 3/5*100 = 60
		if got := reviews.trustScores["seller-1"]; got != 60 {
			t.Errorf("trust score เฉลี่ยควรเป็น 60 ได้ %d", got)
		}
	})
}

func TestReviewService_CanReview(t *testing.T) {
	svc, _, _, listing := setupSoldListingWithConversation(t)
	ctx := context.Background()

	if !svc.CanReview(ctx, listing.ID, "buyer-1") {
		t.Error("buyer-1 ควรมีสิทธิ์รีวิว")
	}
	if svc.CanReview(ctx, listing.ID, "seller-1") {
		t.Error("ผู้ขายรีวิวตัวเองไม่ได้")
	}
	if svc.CanReview(ctx, listing.ID, "random-person") {
		t.Error("คนที่ไม่เคยคุยด้วยไม่ควรมีสิทธิ์")
	}

	svc.Create(ctx, CreateReviewInput{ListingID: listing.ID, ReviewerID: "buyer-1", Rating: 5})
	if svc.CanReview(ctx, listing.ID, "buyer-1") {
		t.Error("รีวิวไปแล้วไม่ควรรีวิวซ้ำได้อีก")
	}
}

func TestReviewService_ListForUser(t *testing.T) {
	svc, _, _, listing := setupSoldListingWithConversation(t)
	ctx := context.Background()

	svc.Create(ctx, CreateReviewInput{ListingID: listing.ID, ReviewerID: "buyer-1", Rating: 5, Comment: "เยี่ยม"})

	got, err := svc.ListForUser(ctx, "seller-1")
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if len(got) != 1 || got[0].Comment != "เยี่ยม" {
		t.Errorf("ได้รีวิวไม่ตรง: %+v", got)
	}
}
