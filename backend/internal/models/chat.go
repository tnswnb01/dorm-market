package models

import "time"

// Conversation คือห้องแชทระหว่างผู้ซื้อกับผู้ขายเกี่ยวกับสินค้าชิ้นหนึ่ง
// มีได้แค่ 1 ห้องต่อ 1 คู่ (listing, buyer) — ดู unique constraint ใน migration
type Conversation struct {
	ID            string      `json:"id"`
	ListingID     string      `json:"listingId"`
	Listing       *Listing    `json:"listing,omitempty"`
	BuyerID       string      `json:"buyerId"`
	SellerID      string      `json:"sellerId"`
	OtherParty    *PublicUser `json:"otherParty,omitempty"` // เติมตอน list ให้ frontend ไม่ต้องเดาเองว่าใครคือ "อีกฝ่าย"
	LastMessage   *Message    `json:"lastMessage,omitempty"`
	CreatedAt     time.Time   `json:"createdAt"`
	LastMessageAt time.Time   `json:"lastMessageAt"`
}

// Message คือข้อความหนึ่งข้อความในห้องแชท
type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"createdAt"`
}
