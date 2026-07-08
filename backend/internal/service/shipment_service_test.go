package service

import (
	"context"
	"errors"
	"testing"

	"dormmarket/internal/models"
)

func newTestShipmentService() (*ShipmentService, *fakeChatRepository, *fakeListingRepository) {
	chat := newFakeChatRepository()
	listings := newFakeListingRepository()
	shipments := newFakeShipmentRepository()
	return NewShipmentService(shipments, chat), chat, listings
}

func setupConversation(t *testing.T, chat *fakeChatRepository, listings *fakeListingRepository) *models.Conversation {
	t.Helper()
	ctx := context.Background()
	listing := &models.Listing{SellerID: "seller-1", CategoryID: "cat-1", Title: "โต๊ะ", Price: 100, Status: models.StatusAvailable}
	listings.Create(ctx, listing)
	conv, err := chat.GetOrCreateConversation(ctx, listing.ID, "buyer-1", "seller-1")
	if err != nil {
		t.Fatalf("เตรียมข้อมูลไม่สำเร็จ: %v", err)
	}
	return conv
}

func TestShipmentService_Create(t *testing.T) {
	t.Run("สร้างแบบนัดรับ (pickup) สำเร็จ", func(t *testing.T) {
		svc, chat, listings := newTestShipmentService()
		conv := setupConversation(t, chat, listings)

		shipment, err := svc.Create(context.Background(), CreateShipmentInput{
			ConversationID: conv.ID, SellerID: "seller-1", Method: models.ShipmentMethodPickup,
		})
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if shipment.Status != models.ShipmentPending {
			t.Errorf("สถานะเริ่มต้นต้องเป็น pending ได้ %s", shipment.Status)
		}
	})

	t.Run("สร้างแบบ delivery ต้องมีชื่อขนส่ง+เลข tracking", func(t *testing.T) {
		svc, chat, listings := newTestShipmentService()
		conv := setupConversation(t, chat, listings)

		_, err := svc.Create(context.Background(), CreateShipmentInput{
			ConversationID: conv.ID, SellerID: "seller-1", Method: models.ShipmentMethodDelivery,
		})
		if !errors.Is(err, ErrCourierRequired) {
			t.Fatalf("ต้องการ ErrCourierRequired แต่ได้ %v", err)
		}

		shipment, err := svc.Create(context.Background(), CreateShipmentInput{
			ConversationID: conv.ID, SellerID: "seller-1", Method: models.ShipmentMethodDelivery,
			CourierName: "Kerry", TrackingNumber: "TH1234567",
		})
		if err != nil {
			t.Fatalf("ไม่ควร error เมื่อกรอกครบ: %v", err)
		}
		if shipment.CourierName != "Kerry" {
			t.Errorf("courier name ผิด")
		}
	})

	t.Run("วิธีจัดส่งไม่ถูกต้อง", func(t *testing.T) {
		svc, chat, listings := newTestShipmentService()
		conv := setupConversation(t, chat, listings)

		_, err := svc.Create(context.Background(), CreateShipmentInput{
			ConversationID: conv.ID, SellerID: "seller-1", Method: "teleport",
		})
		if !errors.Is(err, ErrInvalidShipmentMethod) {
			t.Fatalf("ต้องการ ErrInvalidShipmentMethod แต่ได้ %v", err)
		}
	})

	t.Run("ผู้ซื้อสร้าง shipment ไม่ได้ (ต้องเป็นผู้ขายเท่านั้น)", func(t *testing.T) {
		svc, chat, listings := newTestShipmentService()
		conv := setupConversation(t, chat, listings)

		_, err := svc.Create(context.Background(), CreateShipmentInput{
			ConversationID: conv.ID, SellerID: "buyer-1", Method: models.ShipmentMethodPickup,
		})
		if !errors.Is(err, ErrOnlySellerCanManage) {
			t.Fatalf("ต้องการ ErrOnlySellerCanManage แต่ได้ %v", err)
		}
	})

	t.Run("สร้างซ้ำในสนทนาเดิมไม่ได้", func(t *testing.T) {
		svc, chat, listings := newTestShipmentService()
		conv := setupConversation(t, chat, listings)
		ctx := context.Background()

		svc.Create(ctx, CreateShipmentInput{ConversationID: conv.ID, SellerID: "seller-1", Method: models.ShipmentMethodPickup})
		_, err := svc.Create(ctx, CreateShipmentInput{ConversationID: conv.ID, SellerID: "seller-1", Method: models.ShipmentMethodPickup})
		if !errors.Is(err, ErrShipmentExists) {
			t.Fatalf("ต้องการ ErrShipmentExists แต่ได้ %v", err)
		}
	})
}

func TestShipmentService_UpdateStatus(t *testing.T) {
	svc, chat, listings := newTestShipmentService()
	conv := setupConversation(t, chat, listings)
	ctx := context.Background()
	svc.Create(ctx, CreateShipmentInput{ConversationID: conv.ID, SellerID: "seller-1", Method: models.ShipmentMethodPickup})

	t.Run("ผู้ขายอัปเดตสถานะได้ และบันทึก timeline", func(t *testing.T) {
		shipment, err := svc.UpdateStatus(ctx, conv.ID, "seller-1", models.ShipmentShipped, "นัดรับที่หน้าหอ A เวลา 18:00")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if shipment.Status != models.ShipmentShipped {
			t.Errorf("สถานะต้องเปลี่ยนเป็น shipped ได้ %s", shipment.Status)
		}
		if len(shipment.Events) != 2 { // สร้าง + อัปเดตครั้งนี้
			t.Errorf("ควรมี 2 events (สร้าง+อัปเดต) ได้ %d", len(shipment.Events))
		}
	})

	t.Run("ผู้ซื้ออัปเดตสถานะไม่ได้", func(t *testing.T) {
		_, err := svc.UpdateStatus(ctx, conv.ID, "buyer-1", models.ShipmentCompleted, "")
		if !errors.Is(err, ErrOnlySellerCanManage) {
			t.Fatalf("ต้องการ ErrOnlySellerCanManage แต่ได้ %v", err)
		}
	})

	t.Run("สถานะไม่ถูกต้อง", func(t *testing.T) {
		_, err := svc.UpdateStatus(ctx, conv.ID, "seller-1", "flying", "")
		if !errors.Is(err, ErrInvalidShipmentStatus) {
			t.Fatalf("ต้องการ ErrInvalidShipmentStatus แต่ได้ %v", err)
		}
	})
}

func TestShipmentService_Get(t *testing.T) {
	svc, chat, listings := newTestShipmentService()
	conv := setupConversation(t, chat, listings)
	ctx := context.Background()
	svc.Create(ctx, CreateShipmentInput{ConversationID: conv.ID, SellerID: "seller-1", Method: models.ShipmentMethodPickup})

	t.Run("ผู้ซื้อดูได้", func(t *testing.T) {
		_, err := svc.Get(ctx, conv.ID, "buyer-1")
		if err != nil {
			t.Errorf("ไม่ควร error: %v", err)
		}
	})

	t.Run("ผู้ขายดูได้", func(t *testing.T) {
		_, err := svc.Get(ctx, conv.ID, "seller-1")
		if err != nil {
			t.Errorf("ไม่ควร error: %v", err)
		}
	})

	t.Run("คนนอกดูไม่ได้", func(t *testing.T) {
		_, err := svc.Get(ctx, conv.ID, "random-person")
		if !errors.Is(err, ErrNotConversationParty) {
			t.Fatalf("ต้องการ ErrNotConversationParty แต่ได้ %v", err)
		}
	})
}
