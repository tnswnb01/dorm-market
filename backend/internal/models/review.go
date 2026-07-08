package models

import "time"

// Review คือคะแนน+ความเห็นที่ผู้ซื้อให้กับผู้ขาย หลังซื้อขายสำเร็จ (listing สถานะ sold แล้ว)
type Review struct {
	ID         string      `json:"id"`
	ListingID  string      `json:"listingId"`
	ReviewerID string      `json:"reviewerId"`
	Reviewer   *PublicUser `json:"reviewer,omitempty"`
	RevieweeID string      `json:"revieweeId"`
	Rating     int         `json:"rating"` // 1-5
	Comment    string      `json:"comment"`
	CreatedAt  time.Time   `json:"createdAt"`
}
