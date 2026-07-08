package models

import "time"

type ListingCondition string

const (
	ConditionNew     ListingCondition = "new"
	ConditionLikeNew ListingCondition = "like_new"
	ConditionGood    ListingCondition = "good"
	ConditionWorn    ListingCondition = "worn"
)

type ListingStatus string

const (
	StatusAvailable ListingStatus = "available"
	StatusReserved  ListingStatus = "reserved"
	StatusSold      ListingStatus = "sold"
)

// Listing คือประกาศขายสินค้าหนึ่งชิ้น
//
// SuggestedPrice เป็น nullable และยังไม่ถูกใช้งานใน Phase 1 (เตรียมไว้สำหรับ
// ML price-suggestion ใน Phase 2 เพื่อไม่ต้อง migrate schema เพิ่มทีหลัง)
type Listing struct {
	ID             string           `json:"id"`
	SellerID       string           `json:"sellerId"`
	Seller         *PublicUser      `json:"seller,omitempty"`
	CategoryID     string           `json:"categoryId"`
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	Condition      ListingCondition `json:"condition"`
	Price          int              `json:"price"` // หน่วยเป็นบาท เก็บเป็นจำนวนเต็มเพื่อเลี่ยงปัญหา float
	SuggestedPrice *int             `json:"suggestedPrice,omitempty"`
	Status         ListingStatus    `json:"status"`
	Images         []ListingImage   `json:"images,omitempty"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`
	DeletedAt      *time.Time       `json:"deletedAt,omitempty"` // nil = ยังไม่ถูกลบ
}

type ListingImage struct {
	ID        string `json:"id"`
	ListingID string `json:"listingId"`
	URL       string `json:"url"`
	SortOrder int    `json:"sortOrder"`
}

// PriceSuggestion คือผลลัพธ์การแนะนำราคาแบบ rule-based (Phase 2 ส่วนแรก)
// คำนวณจากราคาเฉลี่ยของประกาศอื่นในหมวดหมู่+สภาพเดียวกันที่มีอยู่แล้วในระบบ
type PriceSuggestion struct {
	SuggestedPrice int `json:"suggestedPrice"`
	MinPrice       int `json:"minPrice"`
	MaxPrice       int `json:"maxPrice"`
	SampleSize     int `json:"sampleSize"`
}

// ListingFilter รวมเงื่อนไขค้นหา/กรองที่ endpoint GET /api/listings รองรับ
type ListingFilter struct {
	CategoryID string
	SellerID   string
	Search     string
	MinPrice   *int
	MaxPrice   *int
	Status     ListingStatus
	Limit      int
	Offset     int
}
