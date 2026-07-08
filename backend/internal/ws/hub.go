package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Client คือ 1 browser tab ที่เปิดห้องแชทหนึ่งห้องอยู่
type Client struct {
	conn           *websocket.Conn
	send           chan []byte
	conversationID string
	userID         string
}

// Hub เก็บ client ทั้งหมดที่กำลังเปิดอยู่ จัดกลุ่มตาม conversationID
// เพื่อ broadcast ข้อความใหม่ไปหาทุกคนที่อยู่ในห้องเดียวกันแบบ real-time
//
// ออกแบบให้รันในโปรเซสเดียว (single instance) — ถ้าจะ scale เป็นหลาย instance
// ทีหลัง ต้องเปลี่ยนไปใช้ Redis pub/sub แทนการเก็บ client ไว้ใน memory แบบนี้
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]bool // conversationID -> set of clients
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]map[*Client]bool)}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[c.conversationID] == nil {
		h.rooms[c.conversationID] = make(map[*Client]bool)
	}
	h.rooms[c.conversationID][c] = true
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.rooms[c.conversationID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.rooms, c.conversationID)
		}
	}
	close(c.send)
}

// Broadcast ส่ง payload (JSON-encoded message) ไปหาทุก client ที่เปิดห้องแชทนี้อยู่
// เรียกจาก handler หลังจากบันทึกข้อความลง DB สำเร็จแล้ว
func (h *Hub) Broadcast(conversationID string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ws: marshal broadcast payload ล้มเหลว: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[conversationID] {
		select {
		case c.send <- data:
		default:
			// client รับข้อความไม่ทัน (buffer เต็ม) — ปล่อยผ่าน ไม่ block ทั้งห้อง
			log.Printf("ws: client ช้าเกินไป ข้าม broadcast รอบนี้ (user %s)", c.userID)
		}
	}
}

// ServeClient รับ connection ที่ upgrade เป็น websocket แล้ว ผูกเข้ากับห้องแชท
// แล้ว loop อ่าน/เขียนจนกว่า connection จะปิด (เรียกจาก handler เป็น goroutine)
func (h *Hub) ServeClient(conn *websocket.Conn, conversationID, userID string, onMessage func(content string)) {
	client := &Client{
		conn:           conn,
		send:           make(chan []byte, 16),
		conversationID: conversationID,
		userID:         userID,
	}
	h.register(client)

	go client.writePump()
	client.readPump(h, onMessage)
}

func (c *Client) readPump(h *Hub, onMessage func(content string)) {
	defer func() {
		h.unregister(c)
		c.conn.Close()
	}()

	for {
		var incoming struct {
			Content string `json:"content"`
		}
		if err := c.conn.ReadJSON(&incoming); err != nil {
			// client ปิด tab หรือ network หลุด — ปิด connection แบบเงียบๆ ไม่ถือเป็น error ร้ายแรง
			break
		}
		onMessage(incoming.Content)
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for data := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			break
		}
	}
}
