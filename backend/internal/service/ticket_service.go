package service

import (
	"context"
	"errors"
	"strings"

	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

var (
	ErrTicketSubjectRequired = errors.New("กรุณาระบุหัวข้อปัญหา")
	ErrTicketMessageRequired = errors.New("กรุณาพิมพ์ข้อความ")
	ErrTicketAccessDenied    = errors.New("คุณไม่มีสิทธิ์เข้าถึง ticket นี้")
	ErrInvalidTicketStatus   = errors.New("สถานะ ticket ไม่ถูกต้อง")
)

type TicketService struct {
	tickets repository.TicketRepository
}

func NewTicketService(tickets repository.TicketRepository) *TicketService {
	return &TicketService{tickets: tickets}
}

type CreateTicketInput struct {
	UserID  string
	Subject string
	Message string
}

// CreateTicket เปิด ticket ใหม่พร้อมข้อความแรกในคราวเดียว
func (s *TicketService) CreateTicket(ctx context.Context, in CreateTicketInput) (*models.SupportTicket, error) {
	subject := strings.TrimSpace(in.Subject)
	message := strings.TrimSpace(in.Message)
	if subject == "" {
		return nil, ErrTicketSubjectRequired
	}
	if message == "" {
		return nil, ErrTicketMessageRequired
	}

	ticket := &models.SupportTicket{UserID: in.UserID, Subject: subject}
	if err := s.tickets.CreateTicket(ctx, ticket); err != nil {
		return nil, err
	}
	if err := s.tickets.AddMessage(ctx, &models.TicketMessage{TicketID: ticket.ID, SenderID: in.UserID, Body: message}); err != nil {
		return nil, err
	}
	return ticket, nil
}

func (s *TicketService) ListMine(ctx context.Context, userID string) ([]models.SupportTicket, error) {
	return s.tickets.ListForUser(ctx, userID)
}

// ListAll คือ inbox ของแอดมิน — status เป็น nil แปลว่าเอาทุกสถานะ
func (s *TicketService) ListAll(ctx context.Context, status *models.TicketStatus) ([]models.SupportTicket, error) {
	return s.tickets.ListAll(ctx, status)
}

// GetThread คืน ticket + ข้อความทั้งหมด — เข้าถึงได้เฉพาะเจ้าของ ticket หรือแอดมินเท่านั้น
func (s *TicketService) GetThread(ctx context.Context, ticketID, callerID string, isAdmin bool) (*models.SupportTicket, []models.TicketMessage, error) {
	ticket, err := s.tickets.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, nil, err
	}
	if !isAdmin && ticket.UserID != callerID {
		return nil, nil, ErrTicketAccessDenied
	}

	messages, err := s.tickets.ListMessages(ctx, ticketID)
	if err != nil {
		return nil, nil, err
	}
	return ticket, messages, nil
}

// AddMessage เพิ่มข้อความในกระทู้ — ถ้าแอดมินตอบ สถานะเปลี่ยนเป็น pending (รอผู้ใช้)
// ถ้าเจ้าของ ticket ทักเพิ่ม สถานะกลับเป็น open เสมอ แม้ ticket จะเคยถูกปิดไปแล้วก็ตาม (reopen อัตโนมัติ)
func (s *TicketService) AddMessage(ctx context.Context, ticketID, senderID string, isAdmin bool, body string) (*models.TicketMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrTicketMessageRequired
	}

	ticket, err := s.tickets.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if !isAdmin && ticket.UserID != senderID {
		return nil, ErrTicketAccessDenied
	}

	msg := &models.TicketMessage{TicketID: ticketID, SenderID: senderID, Body: body}
	if err := s.tickets.AddMessage(ctx, msg); err != nil {
		return nil, err
	}

	newStatus := models.TicketOpen
	if isAdmin {
		newStatus = models.TicketPending
	}
	if err := s.tickets.UpdateStatus(ctx, ticketID, newStatus); err != nil {
		return nil, err
	}
	return msg, nil
}

// UpdateStatus: แอดมินตั้งสถานะอะไรก็ได้ ส่วนเจ้าของ ticket ปิดเรื่องเองได้อย่างเดียว (reopen ต้องทักข้อความใหม่แทน)
func (s *TicketService) UpdateStatus(ctx context.Context, ticketID, callerID string, isAdmin bool, status models.TicketStatus) error {
	ticket, err := s.tickets.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	if !isAdmin {
		if ticket.UserID != callerID {
			return ErrTicketAccessDenied
		}
		if status != models.TicketClosed {
			return ErrInvalidTicketStatus
		}
	}
	if status != models.TicketOpen && status != models.TicketPending && status != models.TicketClosed {
		return ErrInvalidTicketStatus
	}
	return s.tickets.UpdateStatus(ctx, ticketID, status)
}
