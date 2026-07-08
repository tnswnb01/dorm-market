package service

import (
	"context"
	"errors"
	"log"
	"strings"

	"dormmarket/internal/mlservice"
	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

var (
	ErrTitleRequired       = errors.New("กรุณาระบุชื่อสินค้า")
	ErrInvalidPrice        = errors.New("ราคาต้องมากกว่า 0")
	ErrInvalidCategory     = errors.New("กรุณาเลือกหมวดหมู่")
	ErrNotOwner            = errors.New("คุณไม่ใช่เจ้าของประกาศนี้")
	ErrImageSearchDisabled = errors.New("ระบบยังไม่ได้เปิดใช้งานการค้นหาด้วยรูป")
)

// minSamplesForSuggestion คือจำนวนประกาศขั้นต่ำในหมวดหมู่+สภาพเดียวกันที่ต้องมี
// ก่อนจะกล้าแนะนำราคา (แก้ปัญหา cold start — ข้อมูลน้อยเกินไปแนะนำราคาจะไม่น่าเชื่อถือ)
const minSamplesForSuggestion = 3

type ListingService struct {
	listings   repository.ListingRepository
	categories repository.CategoryRepository
	embedder   mlservice.Client // nil ได้ถ้าไม่ได้ตั้งค่า ML_SERVICE_URL (image similarity search จะปิดไปเฉยๆ)
}

func NewListingService(listings repository.ListingRepository, categories repository.CategoryRepository) *ListingService {
	return &ListingService{listings: listings, categories: categories}
}

// WithEmbedder ต่อ ml-service เข้า service — เรียกแบบ optional หลังสร้าง ListingService แล้ว
// (deployment ที่ไม่ได้รัน ml-service ไว้ก็ควรใช้งานลงประกาศ/ค้นหาปกติได้ ไม่มี image search เฉยๆ)
func (s *ListingService) WithEmbedder(client mlservice.Client) *ListingService {
	s.embedder = client
	return s
}

type CreateListingInput struct {
	SellerID    string
	CategoryID  string
	Title       string
	Description string
	Condition   models.ListingCondition
	Price       int
}

func (s *ListingService) Create(ctx context.Context, in CreateListingInput) (*models.Listing, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, ErrTitleRequired
	}
	if in.Price <= 0 {
		return nil, ErrInvalidPrice
	}
	if strings.TrimSpace(in.CategoryID) == "" {
		return nil, ErrInvalidCategory
	}
	if in.Condition == "" {
		in.Condition = models.ConditionGood
	}

	listing := &models.Listing{
		SellerID:    in.SellerID,
		CategoryID:  in.CategoryID,
		Title:       strings.TrimSpace(in.Title),
		Description: strings.TrimSpace(in.Description),
		Condition:   in.Condition,
		Price:       in.Price,
		Status:      models.StatusAvailable,
	}

	// เติมราคาแนะนำไว้ในตัวประกาศเลย (best-effort — ถ้าคำนวณไม่สำเร็จหรือข้อมูลยังน้อย
	// เกินไป ก็ไม่ต้องทำให้การสร้างประกาศล้มเหลวไปด้วย)
	if suggestion, err := s.SuggestPrice(ctx, in.CategoryID, in.Condition); err == nil && suggestion != nil {
		listing.SuggestedPrice = &suggestion.SuggestedPrice
	}

	if err := s.listings.Create(ctx, listing); err != nil {
		return nil, err
	}
	return listing, nil
}

// SuggestPrice คืนราคาแนะนำแบบ rule-based คืน nil (ไม่ error) ถ้าข้อมูลยังน้อยเกินไป
// จะได้แยกแยะได้ระหว่าง "ไม่มีข้อมูลพอ" กับ "เกิดข้อผิดพลาดจริง"
func (s *ListingService) SuggestPrice(ctx context.Context, categoryID string, condition models.ListingCondition) (*models.PriceSuggestion, error) {
	if categoryID == "" {
		return nil, ErrInvalidCategory
	}
	if condition == "" {
		condition = models.ConditionGood
	}

	suggestion, err := s.listings.SuggestPrice(ctx, categoryID, condition)
	if err != nil {
		return nil, err
	}
	if suggestion.SampleSize < minSamplesForSuggestion {
		return nil, nil
	}
	return suggestion, nil
}

func (s *ListingService) Get(ctx context.Context, id string) (*models.Listing, error) {
	return s.listings.GetByID(ctx, id)
}

func (s *ListingService) List(ctx context.Context, filter models.ListingFilter) ([]models.Listing, error) {
	if filter.Status == "" && filter.SellerID == "" {
		filter.Status = models.StatusAvailable
	}
	return s.listings.List(ctx, filter)
}

// AddImage บันทึกรูปสินค้า + คำนวณ embedding สำหรับ image similarity search แบบ best-effort
// (ถ้า ml-service ไม่ได้ตั้งค่าไว้ หรือเรียกไม่สำเร็จ ก็ยังบันทึกรูปได้ปกติ แค่ไม่มี
// embedding ให้ค้นหาด้วยรูปสำหรับรูปนี้เท่านั้น — ไม่ทำให้การอัปโหลดรูปทั้งหมดล้มเหลว)
func (s *ListingService) AddImage(ctx context.Context, listingID, sellerID string, imageData []byte, filename, url string, sortOrder int) error {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil {
		return err
	}
	if listing.SellerID != sellerID {
		return ErrNotOwner
	}

	img := &models.ListingImage{
		ListingID: listingID,
		URL:       url,
		SortOrder: sortOrder,
	}
	if err := s.listings.AddImage(ctx, img); err != nil {
		return err
	}

	if s.embedder != nil {
		embedding, err := s.embedder.Embed(ctx, imageData, filename)
		if err != nil {
			log.Printf("คำนวณ embedding ไม่สำเร็จสำหรับรูป %s (ไม่กระทบการอัปโหลดรูป): %v", img.ID, err)
		} else if err := s.listings.SetImageEmbedding(ctx, img.ID, embedding); err != nil {
			log.Printf("บันทึก embedding ไม่สำเร็จสำหรับรูป %s: %v", img.ID, err)
		}
	}

	return nil
}

// SearchByImage หาประกาศที่มีรูปคล้ายกับรูปที่ส่งเข้ามามากที่สุด — คืน ErrImageSearchDisabled
// ถ้าไม่ได้ตั้งค่า ml-service ไว้ (แยก error นี้ออกมาให้ handler ตอบ 501 ที่สื่อความหมายชัดเจน)
func (s *ListingService) SearchByImage(ctx context.Context, imageData []byte, filename string) ([]models.Listing, error) {
	if s.embedder == nil {
		return nil, ErrImageSearchDisabled
	}
	embedding, err := s.embedder.Embed(ctx, imageData, filename)
	if err != nil {
		return nil, err
	}
	return s.listings.SearchBySimilarListings(ctx, embedding, 20)
}

func (s *ListingService) UpdateStatus(ctx context.Context, id, sellerID string, status models.ListingStatus) error {
	return s.listings.UpdateStatus(ctx, id, sellerID, status)
}

type UpdateListingInput struct {
	ID          string
	SellerID    string
	CategoryID  string
	Title       string
	Description string
	Condition   models.ListingCondition
	Price       int
}

// Update แก้ไขข้อมูลประกาศ — ใช้ validation ชุดเดียวกับตอนสร้าง
func (s *ListingService) Update(ctx context.Context, in UpdateListingInput) (*models.Listing, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, ErrTitleRequired
	}
	if in.Price <= 0 {
		return nil, ErrInvalidPrice
	}
	if strings.TrimSpace(in.CategoryID) == "" {
		return nil, ErrInvalidCategory
	}
	if in.Condition == "" {
		in.Condition = models.ConditionGood
	}

	listing := &models.Listing{
		ID:          in.ID,
		SellerID:    in.SellerID,
		CategoryID:  in.CategoryID,
		Title:       strings.TrimSpace(in.Title),
		Description: strings.TrimSpace(in.Description),
		Condition:   in.Condition,
		Price:       in.Price,
	}
	if err := s.listings.Update(ctx, listing); err != nil {
		return nil, err
	}
	return s.listings.GetByID(ctx, in.ID)
}

// Delete คือ soft delete — เจ้าของเท่านั้นลบได้ ข้อมูลเดิม (แชท/รีวิว/shipment) ยังอยู่ครบ
// แค่ถูกซ่อนจากหน้าค้นหา/รายการเท่านั้น
func (s *ListingService) Delete(ctx context.Context, id, sellerID string) error {
	return s.listings.SoftDelete(ctx, id, sellerID)
}

func (s *ListingService) ListCategories(ctx context.Context) ([]models.Category, error) {
	return s.categories.List(ctx)
}
