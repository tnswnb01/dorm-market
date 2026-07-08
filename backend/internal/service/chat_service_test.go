package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

func newTestChatService() (*ChatService, *fakeListingRepository) {
	listings := newFakeListingRepository()
	chat := newFakeChatRepository()
	return NewChatService(chat, listings), listings
}

func TestChatService_StartConversation(t *testing.T) {
	svc, listings := newTestChatService()
	ctx := context.Background()

	listing := &models.Listing{SellerID: "seller-1", CategoryID: "cat-1", Title: "โต๊ะ", Price: 100, Status: models.StatusAvailable}
	listings.Create(ctx, listing)

	t.Run("เริ่มแชทสำเร็จ", func(t *testing.T) {
		conv, err := svc.StartConversation(ctx, listing.ID, "buyer-1")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if conv.SellerID != "seller-1" || conv.BuyerID != "buyer-1" {
			t.Errorf("ได้ conversation ผิด: %+v", conv)
		}
	})

	t.Run("กดซ้ำได้ห้องเดิม (idempotent)", func(t *testing.T) {
		conv1, _ := svc.StartConversation(ctx, listing.ID, "buyer-1")
		conv2, _ := svc.StartConversation(ctx, listing.ID, "buyer-1")
		if conv1.ID != conv2.ID {
			t.Errorf("กดซ้ำควรได้ ID เดิม ได้ %s กับ %s", conv1.ID, conv2.ID)
		}
	})

	t.Run("เจ้าของประกาศทักหาตัวเองไม่ได้", func(t *testing.T) {
		_, err := svc.StartConversation(ctx, listing.ID, "seller-1")
		if !errors.Is(err, ErrCannotMessageSelf) {
			t.Fatalf("ต้องการ ErrCannotMessageSelf แต่ได้ %v", err)
		}
	})

	t.Run("ประกาศไม่มีอยู่จริง", func(t *testing.T) {
		_, err := svc.StartConversation(ctx, "ไม่มีจริง", "buyer-1")
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("ต้องการ ErrNotFound แต่ได้ %v", err)
		}
	})

	t.Run("ผู้ซื้อคนละคนได้ห้องแชทคนละห้อง", func(t *testing.T) {
		convA, _ := svc.StartConversation(ctx, listing.ID, "buyer-A")
		convB, _ := svc.StartConversation(ctx, listing.ID, "buyer-B")
		if convA.ID == convB.ID {
			t.Error("ผู้ซื้อต่างคนกันต้องได้ห้องแชทคนละห้อง")
		}
	})
}

func TestChatService_SendMessage_And_ListMessages(t *testing.T) {
	svc, listings := newTestChatService()
	ctx := context.Background()

	listing := &models.Listing{SellerID: "seller-1", CategoryID: "cat-1", Title: "โต๊ะ", Price: 100, Status: models.StatusAvailable}
	listings.Create(ctx, listing)
	conv, _ := svc.StartConversation(ctx, listing.ID, "buyer-1")

	t.Run("ผู้ซื้อส่งข้อความได้", func(t *testing.T) {
		msg, err := svc.SendMessage(ctx, conv.ID, "buyer-1", "สนใจโต๊ะตัวนี้ครับ")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if msg.Content != "สนใจโต๊ะตัวนี้ครับ" {
			t.Errorf("เนื้อหาข้อความผิด")
		}
	})

	t.Run("ผู้ขายส่งข้อความได้", func(t *testing.T) {
		_, err := svc.SendMessage(ctx, conv.ID, "seller-1", "ยังอยู่ครับ นัดดูของได้เลย")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
	})

	t.Run("คนนอกส่งข้อความไม่ได้", func(t *testing.T) {
		_, err := svc.SendMessage(ctx, conv.ID, "random-person", "แอบส่ง")
		if !errors.Is(err, ErrNotConversationParty) {
			t.Fatalf("ต้องการ ErrNotConversationParty แต่ได้ %v", err)
		}
	})

	t.Run("ข้อความว่างเปล่าส่งไม่ได้", func(t *testing.T) {
		_, err := svc.SendMessage(ctx, conv.ID, "buyer-1", "   ")
		if !errors.Is(err, ErrEmptyMessage) {
			t.Fatalf("ต้องการ ErrEmptyMessage แต่ได้ %v", err)
		}
	})

	t.Run("ข้อความยาวเกินไปส่งไม่ได้", func(t *testing.T) {
		tooLong := strings.Repeat("a", maxMessageLength+1)
		_, err := svc.SendMessage(ctx, conv.ID, "buyer-1", tooLong)
		if !errors.Is(err, ErrMessageTooLong) {
			t.Fatalf("ต้องการ ErrMessageTooLong แต่ได้ %v", err)
		}
	})

	t.Run("อ่านประวัติแชทได้ครบ เรียงลำดับถูก", func(t *testing.T) {
		messages, err := svc.ListMessages(ctx, conv.ID, "buyer-1")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if len(messages) != 2 {
			t.Fatalf("ควรมี 2 ข้อความ ได้ %d", len(messages))
		}
		if messages[0].Content != "สนใจโต๊ะตัวนี้ครับ" {
			t.Errorf("ลำดับข้อความผิด")
		}
	})

	t.Run("คนนอกอ่านประวัติแชทไม่ได้", func(t *testing.T) {
		_, err := svc.ListMessages(ctx, conv.ID, "random-person")
		if !errors.Is(err, ErrNotConversationParty) {
			t.Fatalf("ต้องการ ErrNotConversationParty แต่ได้ %v", err)
		}
	})
}

func TestChatService_CanAccessConversation(t *testing.T) {
	svc, listings := newTestChatService()
	ctx := context.Background()

	listing := &models.Listing{SellerID: "seller-1", CategoryID: "cat-1", Title: "โต๊ะ", Price: 100, Status: models.StatusAvailable}
	listings.Create(ctx, listing)
	conv, _ := svc.StartConversation(ctx, listing.ID, "buyer-1")

	if !svc.CanAccessConversation(ctx, conv.ID, "buyer-1") {
		t.Error("ผู้ซื้อควรเข้าถึงได้")
	}
	if !svc.CanAccessConversation(ctx, conv.ID, "seller-1") {
		t.Error("ผู้ขายควรเข้าถึงได้")
	}
	if svc.CanAccessConversation(ctx, conv.ID, "random-person") {
		t.Error("คนนอกไม่ควรเข้าถึงได้")
	}
}

func TestChatService_ListConversations(t *testing.T) {
	svc, listings := newTestChatService()
	ctx := context.Background()

	listingA := &models.Listing{SellerID: "seller-1", CategoryID: "cat-1", Title: "โต๊ะ", Price: 100, Status: models.StatusAvailable}
	listingB := &models.Listing{SellerID: "seller-2", CategoryID: "cat-1", Title: "เก้าอี้", Price: 50, Status: models.StatusAvailable}
	listings.Create(ctx, listingA)
	listings.Create(ctx, listingB)

	svc.StartConversation(ctx, listingA.ID, "buyer-1") // buyer-1 คุยกับ seller-1
	svc.StartConversation(ctx, listingB.ID, "buyer-1") // buyer-1 คุยกับ seller-2 ด้วย
	svc.StartConversation(ctx, listingA.ID, "buyer-2") // buyer-2 คุยกับ seller-1 (ไม่เกี่ยวกับ buyer-1)

	got, err := svc.ListConversations(ctx, "buyer-1")
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("buyer-1 ควรมี 2 ห้องแชท ได้ %d", len(got))
	}

	gotSeller, err := svc.ListConversations(ctx, "seller-1")
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if len(gotSeller) != 2 {
		t.Errorf("seller-1 ควรเห็น 2 ห้องแชท (จาก buyer-1 และ buyer-2) ได้ %d", len(gotSeller))
	}
}
