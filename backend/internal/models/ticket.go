package models

import "time"

type TicketStatus string

const (
	TicketOpen    TicketStatus = "open"
	TicketPending TicketStatus = "pending"
	TicketClosed  TicketStatus = "closed"
)

// SupportTicket คือกระทู้ที่ผู้ใช้เปิดขอความช่วยเหลือจากแอดมิน แยกจากระบบแชทซื้อขายเดิม
type SupportTicket struct {
	ID        string       `json:"id"`
	UserID    string       `json:"userId"`
	User      *PublicUser  `json:"user,omitempty"`
	Subject   string       `json:"subject"`
	Status    TicketStatus `json:"status"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

// TicketMessage คือ 1 ข้อความในกระทู้ (จากผู้ใช้เจ้าของ ticket หรือแอดมิน)
type TicketMessage struct {
	ID        string      `json:"id"`
	TicketID  string      `json:"ticketId"`
	SenderID  string      `json:"senderId"`
	Sender    *PublicUser `json:"sender,omitempty"`
	Body      string      `json:"body"`
	CreatedAt time.Time   `json:"createdAt"`
}
