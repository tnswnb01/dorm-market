package mlservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Client คือ interface กลางสำหรับเรียกใช้ ml-service — แยกเป็น interface เพื่อให้ service
// layer เทสได้ง่ายด้วย fake client (ดู fakes_test.go ฝั่ง service) โดยไม่ต้องมี ml-service
// รันจริงตอนเทส เหมือน pattern เดียวกับ GoogleVerifier
type Client interface {
	Embed(ctx context.Context, imageData []byte, filename string) ([]float32, error)
}

// HTTPClient คือ implementation จริง เรียก FastAPI ml-service ผ่าน HTTP
// (ดู ml-service/app/main.py endpoint POST /embed)
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type embedResponse struct {
	Embedding []float32 `json:"embedding"`
	Dim       int       `json:"dim"`
}

func (c *HTTPClient) Embed(ctx context.Context, imageData []byte, filename string) ([]float32, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("สร้าง multipart request ไม่สำเร็จ: %w", err)
	}
	if _, err := part.Write(imageData); err != nil {
		return nil, fmt.Errorf("เขียนข้อมูลรูปลง request ไม่สำเร็จ: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embed", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("เชื่อมต่อ ml-service ไม่สำเร็จ (เช็คว่ารันอยู่ที่ %s หรือไม่): %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ml-service ตอบกลับ status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed embedResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("แปลง response จาก ml-service ไม่สำเร็จ: %w", err)
	}
	return parsed.Embedding, nil
}
