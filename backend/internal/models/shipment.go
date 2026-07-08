package models

import "time"

type ShipmentMethod string

const (
	ShipmentMethodPickup   ShipmentMethod = "pickup"
	ShipmentMethodDelivery ShipmentMethod = "delivery"
)

type ShipmentStatus string

const (
	ShipmentPending   ShipmentStatus = "pending"
	ShipmentShipped   ShipmentStatus = "shipped"
	ShipmentCompleted ShipmentStatus = "completed"
	ShipmentCancelled ShipmentStatus = "cancelled"
)

// Shipment คือการติดตามสถานะส่งมอบสินค้า 1 รายการ ผูกกับ conversation (คู่ผู้ซื้อ-ผู้ขายที่ตกลงกันแล้ว)
// เป็นระบบแบบ manual — ผู้ขายกรอก/อัปเดตสถานะเอง ไม่ได้ดึงข้อมูลจริงจาก API ขนส่ง
type Shipment struct {
	ID             string         `json:"id"`
	ConversationID string         `json:"conversationId"`
	Method         ShipmentMethod `json:"method"`
	CourierName    string         `json:"courierName,omitempty"`
	TrackingNumber string         `json:"trackingNumber,omitempty"`
	Status         ShipmentStatus `json:"status"`
	Events         []ShipmentEvent `json:"events,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// ShipmentEvent คือ 1 จุดใน timeline ประวัติการเปลี่ยนสถานะ
type ShipmentEvent struct {
	ID         string         `json:"id"`
	ShipmentID string         `json:"shipmentId"`
	Status     ShipmentStatus `json:"status"`
	Note       string         `json:"note"`
	CreatedAt  time.Time      `json:"createdAt"`
}
