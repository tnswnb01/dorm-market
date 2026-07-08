package service

import (
	"context"
	"errors"
	"strings"

	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

var (
	ErrInvalidRating      = errors.New("คะแนนต้องอยู่ระหว่าง 1-5 ดาว")
	ErrListingNotSold     = errors.New("รีวิวได้เฉพาะประกาศที่ขายสำเร็จแล้วเท่านั้น")
	ErrCannotReviewSelf   = errors.New("รีวิวประกาศของตัวเองไม่ได้")
	ErrNotEligibleReview  = errors.New("คุณต้องเคยทักแชทกับผู้ขายก่อนถึงจะรีวิวได้")
	ErrAlreadyReviewed    = errors.New("คุณรีวิวประกาศนี้ไปแล้ว")
)

type ReviewService struct {
	reviews  repository.ReviewRepository
	listings repository.ListingRepository
	chat     repository.ChatRepository
	users    repository.UserRepository
}

func NewReviewService(
	reviews repository.ReviewRepository,
	listings repository.ListingRepository,
	chat repository.ChatRepository,
	users repository.UserRepository,
) *ReviewService {
	return &ReviewService{reviews: reviews, listings: listings, chat: chat, users: users}
}

type CreateReviewInput struct {
	ListingID  string
	ReviewerID string
	Rating     int
	Comment    string
}

// Create สร้างรีวิวใหม่ หลังผ่านเงื่อนไขครบทุกข้อ:
//  1. rating ต้องอยู่ 1-5
//  2. listing ต้องมีอยู่จริงและสถานะ "sold" แล้ว
//  3. ผู้รีวิวต้องไม่ใช่เจ้าของประกาศเอง
//  4. ผู้รีวิวต้องเคยมี conversation กับผู้ขายเกี่ยวกับประกาศนี้จริง (กันรีวิวปลอมจากคนไม่เกี่ยวข้อง)
//  5. ยังไม่เคยรีวิวประกาศนี้มาก่อน
//
// สำเร็จแล้วจะคำนวณ trust_score ของผู้ขายใหม่ทันที จากค่าเฉลี่ยดาวทั้งหมดที่เคยได้รับ
func (s *ReviewService) Create(ctx context.Context, in CreateReviewInput) (*models.Review, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return nil, ErrInvalidRating
	}

	listing, err := s.listings.GetByID(ctx, in.ListingID)
	if err != nil {
		return nil, err
	}
	if listing.Status != models.StatusSold {
		return nil, ErrListingNotSold
	}
	if listing.SellerID == in.ReviewerID {
		return nil, ErrCannotReviewSelf
	}

	hasConversation, err := s.chat.HasConversation(ctx, in.ListingID, in.ReviewerID)
	if err != nil {
		return nil, err
	}
	if !hasConversation {
		return nil, ErrNotEligibleReview
	}

	alreadyReviewed, err := s.reviews.HasReviewed(ctx, in.ListingID, in.ReviewerID)
	if err != nil {
		return nil, err
	}
	if alreadyReviewed {
		return nil, ErrAlreadyReviewed
	}

	review := &models.Review{
		ListingID:  in.ListingID,
		ReviewerID: in.ReviewerID,
		RevieweeID: listing.SellerID,
		Rating:     in.Rating,
		Comment:    strings.TrimSpace(in.Comment),
	}
	if err := s.reviews.Create(ctx, review); err != nil {
		return nil, err
	}

	if err := s.reviews.RecomputeTrustScore(ctx, listing.SellerID); err != nil {
		return nil, err
	}

	return review, nil
}

func (s *ReviewService) ListForUser(ctx context.Context, userID string) ([]models.Review, error) {
	return s.reviews.ListForUser(ctx, userID)
}

// CanReview บอกว่า user นี้มีสิทธิ์รีวิว listing นี้ไหม (ใช้โชว์/ซ่อนปุ่ม "เขียนรีวิว" ฝั่ง frontend)
// คืน false เฉยๆ ถ้าไม่มีสิทธิ์ ไม่ error เพราะเป็นเรื่องปกติ (เช่นยังไม่เคยคุยกับผู้ขาย)
func (s *ReviewService) CanReview(ctx context.Context, listingID, userID string) bool {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil || listing.Status != models.StatusSold || listing.SellerID == userID {
		return false
	}

	hasConversation, err := s.chat.HasConversation(ctx, listingID, userID)
	if err != nil || !hasConversation {
		return false
	}

	alreadyReviewed, err := s.reviews.HasReviewed(ctx, listingID, userID)
	if err != nil || alreadyReviewed {
		return false
	}

	return true
}
