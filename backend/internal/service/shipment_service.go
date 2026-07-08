package service

import (
	"context"
	"errors"
	"strings"

	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

var (
	ErrInvalidShipmentMethod = errors.New("วิธีจัดส่งต้องเป็น pickup หรือ delivery เท่านั้น")
	ErrCourierRequired       = errors.New("กรุณาระบุชื่อขนส่งและเลข tracking สำหรับการจัดส่งแบบ delivery")
	ErrShipmentExists        = errors.New("มีรายการติดตามสินค้าสำหรับการสนทนานี้อยู่แล้ว")
	ErrOnlySellerCanManage   = errors.New("เฉพาะผู้ขายเท่านั้นที่จัดการการจัดส่งได้")
	ErrInvalidShipmentStatus = errors.New("สถานะไม่ถูกต้อง")
)

var validShipmentStatuses = map[models.ShipmentStatus]bool{
	models.ShipmentPending:   true,
	models.ShipmentShipped:   true,
	models.ShipmentCompleted: true,
	models.ShipmentCancelled: true,
}

type ShipmentService struct {
	shipments repository.ShipmentRepository
	chat      repository.ChatRepository
}

func NewShipmentService(shipments repository.ShipmentRepository, chat repository.ChatRepository) *ShipmentService {
	return &ShipmentService{shipments: shipments, chat: chat}
}

type CreateShipmentInput struct {
	ConversationID string
	SellerID       string
	Method         models.ShipmentMethod
	CourierName    string
	TrackingNumber string
}

func (s *ShipmentService) Create(ctx context.Context, in CreateShipmentInput) (*models.Shipment, error) {
	conv, err := s.chat.GetConversation(ctx, in.ConversationID)
	if err != nil {
		return nil, err
	}
	if conv.SellerID != in.SellerID {
		return nil, ErrOnlySellerCanManage
	}

	if in.Method != models.ShipmentMethodPickup && in.Method != models.ShipmentMethodDelivery {
		return nil, ErrInvalidShipmentMethod
	}
	if in.Method == models.ShipmentMethodDelivery {
		if strings.TrimSpace(in.CourierName) == "" || strings.TrimSpace(in.TrackingNumber) == "" {
			return nil, ErrCourierRequired
		}
	}

	if _, err := s.shipments.GetByConversationID(ctx, in.ConversationID); err == nil {
		return nil, ErrShipmentExists
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	shipment := &models.Shipment{
		ConversationID: in.ConversationID,
		Method:         in.Method,
		CourierName:    strings.TrimSpace(in.CourierName),
		TrackingNumber: strings.TrimSpace(in.TrackingNumber),
		Status:         models.ShipmentPending,
	}
	if err := s.shipments.Create(ctx, shipment); err != nil {
		return nil, err
	}
	return shipment, nil
}

// Get คืนข้อมูล shipment พร้อม timeline — ทั้งผู้ซื้อและผู้ขายในการสนทนานั้นดูได้
func (s *ShipmentService) Get(ctx context.Context, conversationID, requesterID string) (*models.Shipment, error) {
	conv, err := s.chat.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.BuyerID != requesterID && conv.SellerID != requesterID {
		return nil, ErrNotConversationParty
	}

	return s.shipments.GetByConversationID(ctx, conversationID)
}

func (s *ShipmentService) UpdateStatus(ctx context.Context, conversationID, sellerID string, status models.ShipmentStatus, note string) (*models.Shipment, error) {
	if !validShipmentStatuses[status] {
		return nil, ErrInvalidShipmentStatus
	}

	conv, err := s.chat.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.SellerID != sellerID {
		return nil, ErrOnlySellerCanManage
	}

	shipment, err := s.shipments.GetByConversationID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if err := s.shipments.UpdateStatus(ctx, shipment.ID, status, strings.TrimSpace(note)); err != nil {
		return nil, err
	}

	return s.shipments.GetByConversationID(ctx, conversationID)
}
