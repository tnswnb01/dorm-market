package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"dormmarket/internal/auth"
	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

// ไฟล์นี้รวม fake repository (in-memory) ไว้ใช้เทส service layer โดยไม่ต้องต่อ Postgres จริง
// เพราะ service ขึ้นกับ interface (repository.UserRepository ฯลฯ) ไม่ใช่ struct ตรงๆ
// จึงสลับมาใช้ fake ตัวนี้แทนตอนเทสได้ทันที

// ---------- fakeUserRepository ----------

type fakeUserRepository struct {
	byEmail map[string]*models.User
	byID    map[string]*models.User
	nextID  int

	// เปิดให้ inject error เพื่อจำลอง DB ล่มได้ในเทสที่ต้องการ
	forceGetByEmailErr error
	forceCreateErr     error
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		byEmail: make(map[string]*models.User),
		byID:    make(map[string]*models.User),
	}
}

func (r *fakeUserRepository) Create(ctx context.Context, u *models.User) error {
	if r.forceCreateErr != nil {
		return r.forceCreateErr
	}
	r.nextID++
	u.ID = fmt.Sprintf("user-%d", r.nextID)
	u.TrustScore = 100
	if u.Role == "" {
		u.Role = models.RoleUser
	}
	u.CreatedAt = time.Now()

	r.byEmail[u.Email] = u
	r.byID[u.ID] = u
	return nil
}

func (r *fakeUserRepository) Ban(ctx context.Context, userID, reason, adminID string) error {
	u, ok := r.byID[userID]
	if !ok {
		return repository.ErrNotFound
	}
	u.IsBanned = true
	u.BanReason = reason
	return nil
}

func (r *fakeUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if r.forceGetByEmailErr != nil {
		return nil, r.forceGetByEmailErr
	}
	u, ok := r.byEmail[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (r *fakeUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (r *fakeUserRepository) GetByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	for _, u := range r.byID {
		if u.GoogleID != nil && *u.GoogleID == googleID {
			return u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *fakeUserRepository) LinkGoogleID(ctx context.Context, userID string, googleID string) error {
	u, ok := r.byID[userID]
	if !ok {
		return repository.ErrNotFound
	}
	u.GoogleID = &googleID
	return nil
}

// ---------- fakeListingRepository ----------

type fakeListingRepository struct {
	byID       map[string]*models.Listing
	images     map[string][]models.ListingImage
	embeddings map[string][]float32 // key: image ID
	nextID     int

	forceGetByIDErr error
}

func newFakeListingRepository() *fakeListingRepository {
	return &fakeListingRepository{
		byID:       make(map[string]*models.Listing),
		images:     make(map[string][]models.ListingImage),
		embeddings: make(map[string][]float32),
	}
}

func (r *fakeListingRepository) Create(ctx context.Context, l *models.Listing) error {
	r.nextID++
	l.ID = fmt.Sprintf("listing-%d", r.nextID)
	l.CreatedAt = time.Now()
	l.UpdatedAt = time.Now()
	r.byID[l.ID] = l
	return nil
}

func (r *fakeListingRepository) GetByID(ctx context.Context, id string) (*models.Listing, error) {
	if r.forceGetByIDErr != nil {
		return nil, r.forceGetByIDErr
	}
	l, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	// คืน copy พร้อม images แนบไปด้วย เลียนแบบพฤติกรรมของ postgres implementation จริง
	cp := *l
	cp.Images = r.images[id]
	return &cp, nil
}

func (r *fakeListingRepository) List(ctx context.Context, filter models.ListingFilter) ([]models.Listing, error) {
	var out []models.Listing
	for _, l := range r.byID {
		if l.DeletedAt != nil {
			continue
		}
		if filter.Status != "" && l.Status != filter.Status {
			continue
		}
		if filter.SellerID != "" && l.SellerID != filter.SellerID {
			continue
		}
		if filter.CategoryID != "" && l.CategoryID != filter.CategoryID {
			continue
		}
		out = append(out, *l)
	}
	return out, nil
}

func (r *fakeListingRepository) AddImage(ctx context.Context, img *models.ListingImage) error {
	r.nextID++
	img.ID = fmt.Sprintf("image-%d", r.nextID)
	r.images[img.ListingID] = append(r.images[img.ListingID], *img)
	return nil
}

func (r *fakeListingRepository) ListImages(ctx context.Context, listingID string) ([]models.ListingImage, error) {
	return r.images[listingID], nil
}

func (r *fakeListingRepository) UpdateStatus(ctx context.Context, id string, sellerID string, status models.ListingStatus) error {
	l, ok := r.byID[id]
	if !ok || l.SellerID != sellerID {
		return repository.ErrNotFound
	}
	l.Status = status
	return nil
}

func (r *fakeListingRepository) Update(ctx context.Context, updated *models.Listing) error {
	l, ok := r.byID[updated.ID]
	if !ok || l.SellerID != updated.SellerID || l.DeletedAt != nil {
		return repository.ErrNotFound
	}
	l.Title = updated.Title
	l.Description = updated.Description
	l.CategoryID = updated.CategoryID
	l.Condition = updated.Condition
	l.Price = updated.Price
	return nil
}

func (r *fakeListingRepository) SoftDelete(ctx context.Context, id string, sellerID string) error {
	l, ok := r.byID[id]
	if !ok || l.SellerID != sellerID || l.DeletedAt != nil {
		return repository.ErrNotFound
	}
	now := time.Now()
	l.DeletedAt = &now
	return nil
}

func (r *fakeListingRepository) AdminSoftDelete(ctx context.Context, id string) error {
	l, ok := r.byID[id]
	if !ok || l.DeletedAt != nil {
		return repository.ErrNotFound
	}
	now := time.Now()
	l.DeletedAt = &now
	return nil
}

func (r *fakeListingRepository) SuggestPrice(ctx context.Context, categoryID string, condition models.ListingCondition) (*models.PriceSuggestion, error) {
	var prices []int
	for _, l := range r.byID {
		if l.CategoryID == categoryID && l.Condition == condition {
			prices = append(prices, l.Price)
		}
	}
	if len(prices) == 0 {
		return &models.PriceSuggestion{}, nil
	}
	sum, min, max := 0, prices[0], prices[0]
	for _, p := range prices {
		sum += p
		if p < min {
			min = p
		}
		if p > max {
			max = p
		}
	}
	return &models.PriceSuggestion{
		SuggestedPrice: sum / len(prices),
		MinPrice:       min,
		MaxPrice:       max,
		SampleSize:     len(prices),
	}, nil
}

func (r *fakeListingRepository) SetImageEmbedding(ctx context.Context, imageID string, embedding []float32) error {
	r.embeddings[imageID] = embedding
	return nil
}

// SearchBySimilarListings แบบง่ายสำหรับเทส — ใช้ cosine similarity ธรรมดา (ไม่ต้องพึ่ง pgvector)
// เรียงจากใกล้เคียงที่สุดไปน้อยที่สุด แล้วตัดตาม limit
func (r *fakeListingRepository) SearchBySimilarListings(ctx context.Context, embedding []float32, limit int) ([]models.Listing, error) {
	type scored struct {
		listing models.Listing
		score   float64
	}
	var results []scored

	for listingID, imgs := range r.images {
		listing, ok := r.byID[listingID]
		if !ok || listing.DeletedAt != nil || listing.Status != models.StatusAvailable {
			continue
		}
		bestScore := -2.0 // ต่ำกว่า cosine similarity ที่เป็นไปได้ทุกค่า (-1 ถึง 1)
		for _, img := range imgs {
			vec, ok := r.embeddings[img.ID]
			if !ok {
				continue
			}
			score := cosineSimilarity(embedding, vec)
			if score > bestScore {
				bestScore = score
			}
		}
		if bestScore > -2.0 {
			results = append(results, scored{listing: *listing, score: bestScore})
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })

	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}
	out := make([]models.Listing, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, results[i].listing)
	}
	return out, nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return -2.0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return -2.0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// ---------- fakeCategoryRepository ----------

type fakeCategoryRepository struct {
	categories []models.Category
}

func (r *fakeCategoryRepository) List(ctx context.Context) ([]models.Category, error) {
	return r.categories, nil
}

// ---------- fakeChatRepository ----------

type fakeChatRepository struct {
	conversations map[string]*models.Conversation // key: id
	byListingBuyer map[string]*models.Conversation // key: listingID+"|"+buyerID
	messages      map[string][]models.Message      // key: conversationID
	nextID        int
}

func newFakeChatRepository() *fakeChatRepository {
	return &fakeChatRepository{
		conversations:  make(map[string]*models.Conversation),
		byListingBuyer: make(map[string]*models.Conversation),
		messages:       make(map[string][]models.Message),
	}
}

func (r *fakeChatRepository) GetOrCreateConversation(ctx context.Context, listingID, buyerID, sellerID string) (*models.Conversation, error) {
	key := listingID + "|" + buyerID
	if existing, ok := r.byListingBuyer[key]; ok {
		return existing, nil
	}
	r.nextID++
	conv := &models.Conversation{
		ID:            fmt.Sprintf("conv-%d", r.nextID),
		ListingID:     listingID,
		BuyerID:       buyerID,
		SellerID:      sellerID,
		CreatedAt:     time.Now(),
		LastMessageAt: time.Now(),
	}
	r.conversations[conv.ID] = conv
	r.byListingBuyer[key] = conv
	return conv, nil
}

func (r *fakeChatRepository) GetConversation(ctx context.Context, id string) (*models.Conversation, error) {
	c, ok := r.conversations[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return c, nil
}

func (r *fakeChatRepository) HasConversation(ctx context.Context, listingID, buyerID string) (bool, error) {
	_, ok := r.byListingBuyer[listingID+"|"+buyerID]
	return ok, nil
}

func (r *fakeChatRepository) ListConversationsForUser(ctx context.Context, userID string) ([]models.Conversation, error) {
	var out []models.Conversation
	for _, c := range r.conversations {
		if c.BuyerID == userID || c.SellerID == userID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (r *fakeChatRepository) CreateMessage(ctx context.Context, msg *models.Message) error {
	r.nextID++
	msg.ID = fmt.Sprintf("msg-%d", r.nextID)
	msg.CreatedAt = time.Now()
	r.messages[msg.ConversationID] = append(r.messages[msg.ConversationID], *msg)
	if conv, ok := r.conversations[msg.ConversationID]; ok {
		conv.LastMessageAt = msg.CreatedAt
	}
	return nil
}

func (r *fakeChatRepository) ListMessages(ctx context.Context, conversationID string) ([]models.Message, error) {
	return r.messages[conversationID], nil
}

// ---------- fakeGoogleVerifier ----------

// fakeGoogleVerifier คืนค่า claims ที่กำหนดไว้ล่วงหน้าตรงๆ โดยไม่ verify signature จริง
// (การ verify signature จริงถูกเทสแยกไว้แล้วใน internal/auth/google_test.go)
// ใช้ตัวนี้ทดสอบแค่ business logic รอบข้าง (find-or-create user, link account)
type fakeGoogleVerifier struct {
	claims *auth.GoogleClaims
	err    error
}

func (v *fakeGoogleVerifier) Verify(ctx context.Context, idToken string) (*auth.GoogleClaims, error) {
	if v.err != nil {
		return nil, v.err
	}
	return v.claims, nil
}

// ---------- fakeReviewRepository ----------

type fakeReviewRepository struct {
	reviews map[string]*models.Review // key: id
	nextID  int
	// จำลอง users map ไว้เอง (ไม่ผูกกับ fakeUserRepository) เพื่อเทส RecomputeTrustScore ได้ตรงๆ
	trustScores map[string]int
}

func newFakeReviewRepository() *fakeReviewRepository {
	return &fakeReviewRepository{
		reviews:     make(map[string]*models.Review),
		trustScores: make(map[string]int),
	}
}

func (r *fakeReviewRepository) Create(ctx context.Context, review *models.Review) error {
	r.nextID++
	review.ID = fmt.Sprintf("review-%d", r.nextID)
	review.CreatedAt = time.Now()
	cp := *review
	r.reviews[review.ID] = &cp
	return nil
}

func (r *fakeReviewRepository) HasReviewed(ctx context.Context, listingID, reviewerID string) (bool, error) {
	for _, rv := range r.reviews {
		if rv.ListingID == listingID && rv.ReviewerID == reviewerID {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeReviewRepository) ListForUser(ctx context.Context, revieweeID string) ([]models.Review, error) {
	var out []models.Review
	for _, rv := range r.reviews {
		if rv.RevieweeID == revieweeID {
			out = append(out, *rv)
		}
	}
	return out, nil
}

func (r *fakeReviewRepository) RecomputeTrustScore(ctx context.Context, userID string) error {
	var sum, count int
	for _, rv := range r.reviews {
		if rv.RevieweeID == userID {
			sum += rv.Rating
			count++
		}
	}
	if count == 0 {
		return nil
	}
	avg := float64(sum) / float64(count)
	r.trustScores[userID] = int(avg/5*100 + 0.5) // round ครึ่งขึ้น เหมือน SQL ROUND()
	return nil
}

// ---------- fakeEmbedderClient ----------

// fakeEmbedderClient จำลอง ml-service — คืน embedding แบบ deterministic จาก byte แรกๆ
// ของรูป ใช้เทสว่า ListingService เรียก embedder ถูกต้อง โดยไม่ต้องมี ml-service รันจริง
type fakeEmbedderClient struct {
	err       error
	callCount int
}

func (c *fakeEmbedderClient) Embed(ctx context.Context, imageData []byte, filename string) ([]float32, error) {
	c.callCount++
	if c.err != nil {
		return nil, c.err
	}
	// สร้าง embedding ปลอมจาก byte ของรูป (รูปเดียวกัน = embedding เดียวกันเสมอ)
	seed := float32(0)
	for _, b := range imageData {
		seed += float32(b)
	}
	vec := make([]float32, 8) // สั้นพอสำหรับเทส ไม่ต้องเต็ม 256 มิติจริง
	for i := range vec {
		vec[i] = seed + float32(i)
	}
	return vec, nil
}

// ---------- fakeShipmentRepository ----------

type fakeShipmentRepository struct {
	byID           map[string]*models.Shipment
	byConversation map[string]*models.Shipment
	events         map[string][]models.ShipmentEvent
	nextID         int
}

func newFakeShipmentRepository() *fakeShipmentRepository {
	return &fakeShipmentRepository{
		byID:           make(map[string]*models.Shipment),
		byConversation: make(map[string]*models.Shipment),
		events:         make(map[string][]models.ShipmentEvent),
	}
}

func (r *fakeShipmentRepository) Create(ctx context.Context, s *models.Shipment) error {
	r.nextID++
	s.ID = fmt.Sprintf("shipment-%d", r.nextID)
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	r.byID[s.ID] = s
	r.byConversation[s.ConversationID] = s
	r.events[s.ID] = append(r.events[s.ID], models.ShipmentEvent{
		ID: fmt.Sprintf("event-%d", r.nextID), ShipmentID: s.ID, Status: s.Status,
		Note: "สร้างรายการติดตามสินค้า", CreatedAt: time.Now(),
	})
	return nil
}

func (r *fakeShipmentRepository) GetByConversationID(ctx context.Context, conversationID string) (*models.Shipment, error) {
	s, ok := r.byConversation[conversationID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *s
	cp.Events = r.events[s.ID]
	return &cp, nil
}

func (r *fakeShipmentRepository) UpdateStatus(ctx context.Context, shipmentID string, status models.ShipmentStatus, note string) error {
	s, ok := r.byID[shipmentID]
	if !ok {
		return repository.ErrNotFound
	}
	s.Status = status
	s.UpdatedAt = time.Now()
	r.nextID++
	r.events[shipmentID] = append(r.events[shipmentID], models.ShipmentEvent{
		ID: fmt.Sprintf("event-%d", r.nextID), ShipmentID: shipmentID, Status: status,
		Note: note, CreatedAt: time.Now(),
	})
	return nil
}

// helper เอาไว้จำลอง unexpected DB error (ไม่ใช่ ErrNotFound) ในเทส
var errSimulatedDBFailure = errors.New("simulated db failure")

// ---------- fakeReportRepository ----------

type fakeReportRepository struct {
	byID   map[string]*models.Report
	nextID int
}

func newFakeReportRepository() *fakeReportRepository {
	return &fakeReportRepository{byID: make(map[string]*models.Report)}
}

func (r *fakeReportRepository) Create(ctx context.Context, report *models.Report) error {
	r.nextID++
	report.ID = fmt.Sprintf("report-%d", r.nextID)
	report.Status = models.ReportPending
	report.CreatedAt = time.Now()
	cp := *report
	r.byID[report.ID] = &cp
	return nil
}

func (r *fakeReportRepository) GetByID(ctx context.Context, id string) (*models.Report, error) {
	rp, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *rp
	return &cp, nil
}

func (r *fakeReportRepository) List(ctx context.Context, status *models.ReportStatus) ([]models.Report, error) {
	var out []models.Report
	for _, rp := range r.byID {
		if status != nil && rp.Status != *status {
			continue
		}
		out = append(out, *rp)
	}
	return out, nil
}

func (r *fakeReportRepository) Resolve(ctx context.Context, id string, status models.ReportStatus, action models.ReportResolutionAction, note, adminID string) error {
	rp, ok := r.byID[id]
	if !ok || rp.Status != models.ReportPending {
		return repository.ErrNotFound
	}
	rp.Status = status
	rp.ResolutionAction = &action
	rp.ResolutionNote = note
	rp.ResolvedBy = &adminID
	now := time.Now()
	rp.ResolvedAt = &now
	return nil
}

// ---------- fakeTicketRepository ----------

type fakeTicketRepository struct {
	byID     map[string]*models.SupportTicket
	messages map[string][]models.TicketMessage
	nextID   int
}

func newFakeTicketRepository() *fakeTicketRepository {
	return &fakeTicketRepository{
		byID:     make(map[string]*models.SupportTicket),
		messages: make(map[string][]models.TicketMessage),
	}
}

func (r *fakeTicketRepository) CreateTicket(ctx context.Context, ticket *models.SupportTicket) error {
	r.nextID++
	ticket.ID = fmt.Sprintf("ticket-%d", r.nextID)
	ticket.Status = models.TicketOpen
	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()
	cp := *ticket
	r.byID[ticket.ID] = &cp
	return nil
}

func (r *fakeTicketRepository) GetTicket(ctx context.Context, id string) (*models.SupportTicket, error) {
	t, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *fakeTicketRepository) ListForUser(ctx context.Context, userID string) ([]models.SupportTicket, error) {
	var out []models.SupportTicket
	for _, t := range r.byID {
		if t.UserID == userID {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (r *fakeTicketRepository) ListAll(ctx context.Context, status *models.TicketStatus) ([]models.SupportTicket, error) {
	var out []models.SupportTicket
	for _, t := range r.byID {
		if status != nil && t.Status != *status {
			continue
		}
		out = append(out, *t)
	}
	return out, nil
}

func (r *fakeTicketRepository) UpdateStatus(ctx context.Context, id string, status models.TicketStatus) error {
	t, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	t.Status = status
	t.UpdatedAt = time.Now()
	return nil
}

func (r *fakeTicketRepository) AddMessage(ctx context.Context, msg *models.TicketMessage) error {
	if _, ok := r.byID[msg.TicketID]; !ok {
		return repository.ErrNotFound
	}
	r.nextID++
	msg.ID = fmt.Sprintf("ticket-msg-%d", r.nextID)
	msg.CreatedAt = time.Now()
	r.messages[msg.TicketID] = append(r.messages[msg.TicketID], *msg)
	return nil
}

func (r *fakeTicketRepository) ListMessages(ctx context.Context, ticketID string) ([]models.TicketMessage, error) {
	return r.messages[ticketID], nil
}
