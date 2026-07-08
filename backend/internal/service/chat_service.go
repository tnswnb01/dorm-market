package service

import (
	"context"
	"errors"
	"strings"

	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

var (
	ErrCannotMessageSelf  = errors.New("ไม่สามารถทักแชทหาประกาศของตัวเองได้")
	ErrEmptyMessage       = errors.New("ข้อความว่างเปล่า")
	ErrMessageTooLong     = errors.New("ข้อความยาวเกินไป (สูงสุด 2000 ตัวอักษร)")
	ErrNotConversationParty = errors.New("คุณไม่ได้เป็นส่วนหนึ่งของการสนทนานี้")
)

const maxMessageLength = 2000

type ChatService struct {
	chat     repository.ChatRepository
	listings repository.ListingRepository
}

func NewChatService(chat repository.ChatRepository, listings repository.ListingRepository) *ChatService {
	return &ChatService{chat: chat, listings: listings}
}

// StartConversation เริ่มห้องแชทสำหรับ (listing, buyer) คู่นี้ — ถ้ามีอยู่แล้วคืนห้องเดิม
// (idempotent — กด "ติดต่อผู้ขาย" ซ้ำกี่ครั้งก็ได้ห้องเดิม ไม่สร้างซ้ำ)
func (s *ChatService) StartConversation(ctx context.Context, listingID, buyerID string) (*models.Conversation, error) {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if listing.SellerID == buyerID {
		return nil, ErrCannotMessageSelf
	}

	return s.chat.GetOrCreateConversation(ctx, listingID, buyerID, listing.SellerID)
}

func (s *ChatService) ListConversations(ctx context.Context, userID string) ([]models.Conversation, error) {
	return s.chat.ListConversationsForUser(ctx, userID)
}

// GetConversationDetails คืนข้อมูล conversation เดี่ยวพร้อม listing แนบมาด้วย
// ใช้ตอนเปิดหน้าแชท เพื่อให้ frontend รู้ว่า user คนปัจจุบันเป็นผู้ซื้อหรือผู้ขาย
// (เทียบ userID กับ conversation.SellerID เอาเอง) และโชว์ชื่อสินค้าที่กำลังคุยถึง
func (s *ChatService) GetConversationDetails(ctx context.Context, conversationID, requesterID string) (*models.Conversation, error) {
	conv, err := s.authorize(ctx, conversationID, requesterID)
	if err != nil {
		return nil, err
	}

	listing, err := s.listings.GetByID(ctx, conv.ListingID)
	if err == nil {
		conv.Listing = listing
	}

	return conv, nil
}

// authorize ตรวจว่า userID เป็นผู้ซื้อหรือผู้ขายในห้องแชทนี้จริง — ทุก endpoint ที่แตะห้องแชท
// ต้องเรียกฟังก์ชันนี้ก่อนเสมอ กันคนอื่นแอบอ่าน/ส่งข้อความในห้องที่ตัวเองไม่เกี่ยวข้อง
func (s *ChatService) authorize(ctx context.Context, conversationID, userID string) (*models.Conversation, error) {
	conv, err := s.chat.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.BuyerID != userID && conv.SellerID != userID {
		return nil, ErrNotConversationParty
	}
	return conv, nil
}

func (s *ChatService) ListMessages(ctx context.Context, conversationID, requesterID string) ([]models.Message, error) {
	if _, err := s.authorize(ctx, conversationID, requesterID); err != nil {
		return nil, err
	}
	return s.chat.ListMessages(ctx, conversationID)
}

func (s *ChatService) SendMessage(ctx context.Context, conversationID, senderID, content string) (*models.Message, error) {
	if _, err := s.authorize(ctx, conversationID, senderID); err != nil {
		return nil, err
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrEmptyMessage
	}
	if len(content) > maxMessageLength {
		return nil, ErrMessageTooLong
	}

	msg := &models.Message{
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
	}
	if err := s.chat.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// CanAccessConversation ใช้ตอน WebSocket handshake (ก่อน upgrade connection)
// เพื่อเช็คสิทธิ์ก่อนปล่อยให้เข้าห้องแชท real-time
func (s *ChatService) CanAccessConversation(ctx context.Context, conversationID, userID string) bool {
	_, err := s.authorize(ctx, conversationID, userID)
	return err == nil
}
