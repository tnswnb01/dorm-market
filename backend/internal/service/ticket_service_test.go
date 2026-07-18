package service

import (
	"context"
	"errors"
	"testing"

	"dormmarket/internal/models"
)

func TestTicketService_CreateTicket(t *testing.T) {
	tickets := newFakeTicketRepository()
	svc := NewTicketService(tickets)
	ctx := context.Background()

	ticket, err := svc.CreateTicket(ctx, CreateTicketInput{UserID: "user-1", Subject: "จ่ายเงินแล้วไม่ได้ของ", Message: "ซื้อโต๊ะไปแล้วผู้ขายเงียบหาย"})
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if ticket.Status != models.TicketOpen {
		t.Errorf("สถานะเริ่มต้นต้องเป็น open ได้ %s", ticket.Status)
	}
	if len(tickets.messages[ticket.ID]) != 1 {
		t.Errorf("ต้องมีข้อความแรกติดมาด้วย ได้ %d ข้อความ", len(tickets.messages[ticket.ID]))
	}

	t.Run("subject ว่างไม่ได้", func(t *testing.T) {
		_, err := svc.CreateTicket(ctx, CreateTicketInput{UserID: "user-1", Subject: "  ", Message: "ข้อความ"})
		if !errors.Is(err, ErrTicketSubjectRequired) {
			t.Fatalf("ต้องการ ErrTicketSubjectRequired แต่ได้ %v", err)
		}
	})

	t.Run("message ว่างไม่ได้", func(t *testing.T) {
		_, err := svc.CreateTicket(ctx, CreateTicketInput{UserID: "user-1", Subject: "หัวข้อ", Message: ""})
		if !errors.Is(err, ErrTicketMessageRequired) {
			t.Fatalf("ต้องการ ErrTicketMessageRequired แต่ได้ %v", err)
		}
	})
}

func TestTicketService_GetThread_AccessControl(t *testing.T) {
	tickets := newFakeTicketRepository()
	svc := NewTicketService(tickets)
	ctx := context.Background()

	ticket, _ := svc.CreateTicket(ctx, CreateTicketInput{UserID: "user-1", Subject: "หัวข้อ", Message: "ข้อความ"})

	t.Run("เจ้าของ ticket เข้าถึงได้", func(t *testing.T) {
		_, _, err := svc.GetThread(ctx, ticket.ID, "user-1", false)
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
	})

	t.Run("แอดมินเข้าถึงได้เสมอ", func(t *testing.T) {
		_, _, err := svc.GetThread(ctx, ticket.ID, "admin-1", true)
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
	})

	t.Run("คนอื่นเข้าถึงไม่ได้", func(t *testing.T) {
		_, _, err := svc.GetThread(ctx, ticket.ID, "random-person", false)
		if !errors.Is(err, ErrTicketAccessDenied) {
			t.Fatalf("ต้องการ ErrTicketAccessDenied แต่ได้ %v", err)
		}
	})
}

func TestTicketService_AddMessage_StatusTransitions(t *testing.T) {
	tickets := newFakeTicketRepository()
	svc := NewTicketService(tickets)
	ctx := context.Background()

	ticket, _ := svc.CreateTicket(ctx, CreateTicketInput{UserID: "user-1", Subject: "หัวข้อ", Message: "ข้อความแรก"})

	t.Run("แอดมินตอบ -> สถานะเป็น pending", func(t *testing.T) {
		_, err := svc.AddMessage(ctx, ticket.ID, "admin-1", true, "กำลังตรวจสอบให้ครับ")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if tickets.byID[ticket.ID].Status != models.TicketPending {
			t.Errorf("สถานะต้องเป็น pending ได้ %s", tickets.byID[ticket.ID].Status)
		}
	})

	t.Run("เจ้าของทักเพิ่ม -> สถานะกลับเป็น open", func(t *testing.T) {
		_, err := svc.AddMessage(ctx, ticket.ID, "user-1", false, "ขอบคุณครับ รอผลอยู่")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if tickets.byID[ticket.ID].Status != models.TicketOpen {
			t.Errorf("สถานะต้องเป็น open ได้ %s", tickets.byID[ticket.ID].Status)
		}
	})

	t.Run("ticket ปิดแล้ว เจ้าของทักใหม่ reopen อัตโนมัติ", func(t *testing.T) {
		tickets.UpdateStatus(ctx, ticket.ID, models.TicketClosed)
		_, err := svc.AddMessage(ctx, ticket.ID, "user-1", false, "ขอเปิดเรื่องใหม่")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if tickets.byID[ticket.ID].Status != models.TicketOpen {
			t.Errorf("ticket ต้อง reopen เป็น open ได้ %s", tickets.byID[ticket.ID].Status)
		}
	})

	t.Run("คนอื่นทักไม่ได้", func(t *testing.T) {
		_, err := svc.AddMessage(ctx, ticket.ID, "random-person", false, "แทรกได้ไหม")
		if !errors.Is(err, ErrTicketAccessDenied) {
			t.Fatalf("ต้องการ ErrTicketAccessDenied แต่ได้ %v", err)
		}
	})
}

func TestTicketService_UpdateStatus(t *testing.T) {
	tickets := newFakeTicketRepository()
	svc := NewTicketService(tickets)
	ctx := context.Background()

	t.Run("เจ้าของปิด ticket ตัวเองได้", func(t *testing.T) {
		ticket, _ := svc.CreateTicket(ctx, CreateTicketInput{UserID: "user-1", Subject: "หัวข้อ", Message: "ข้อความ"})
		if err := svc.UpdateStatus(ctx, ticket.ID, "user-1", false, models.TicketClosed); err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
	})

	t.Run("เจ้าของเปิด ticket ตัวเองใหม่ผ่านทางนี้ไม่ได้", func(t *testing.T) {
		ticket, _ := svc.CreateTicket(ctx, CreateTicketInput{UserID: "user-1", Subject: "หัวข้อ", Message: "ข้อความ"})
		err := svc.UpdateStatus(ctx, ticket.ID, "user-1", false, models.TicketOpen)
		if !errors.Is(err, ErrInvalidTicketStatus) {
			t.Fatalf("ต้องการ ErrInvalidTicketStatus แต่ได้ %v", err)
		}
	})

	t.Run("แอดมินตั้งสถานะอะไรก็ได้", func(t *testing.T) {
		ticket, _ := svc.CreateTicket(ctx, CreateTicketInput{UserID: "user-1", Subject: "หัวข้อ", Message: "ข้อความ"})
		if err := svc.UpdateStatus(ctx, ticket.ID, "admin-1", true, models.TicketPending); err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
	})

	t.Run("คนอื่นแก้สถานะไม่ได้", func(t *testing.T) {
		ticket, _ := svc.CreateTicket(ctx, CreateTicketInput{UserID: "user-1", Subject: "หัวข้อ", Message: "ข้อความ"})
		err := svc.UpdateStatus(ctx, ticket.ID, "random-person", false, models.TicketClosed)
		if !errors.Is(err, ErrTicketAccessDenied) {
			t.Fatalf("ต้องการ ErrTicketAccessDenied แต่ได้ %v", err)
		}
	})
}
