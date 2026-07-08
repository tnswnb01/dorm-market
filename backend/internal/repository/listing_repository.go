package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"dormmarket/internal/models"
)

type ListingRepository interface {
	Create(ctx context.Context, l *models.Listing) error
	GetByID(ctx context.Context, id string) (*models.Listing, error)
	List(ctx context.Context, filter models.ListingFilter) ([]models.Listing, error)
	AddImage(ctx context.Context, img *models.ListingImage) error
	ListImages(ctx context.Context, listingID string) ([]models.ListingImage, error)
	UpdateStatus(ctx context.Context, id string, sellerID string, status models.ListingStatus) error
	SuggestPrice(ctx context.Context, categoryID string, condition models.ListingCondition) (*models.PriceSuggestion, error)
	Update(ctx context.Context, l *models.Listing) error
	SoftDelete(ctx context.Context, id string, sellerID string) error
	SetImageEmbedding(ctx context.Context, imageID string, embedding []float32) error
	SearchBySimilarListings(ctx context.Context, embedding []float32, limit int) ([]models.Listing, error)
}

type postgresListingRepository struct {
	db *sql.DB
}

func NewListingRepository(db *sql.DB) ListingRepository {
	return &postgresListingRepository{db: db}
}

func (r *postgresListingRepository) Create(ctx context.Context, l *models.Listing) error {
	query := `
		INSERT INTO listings (seller_id, category_id, title, description, condition, price, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query,
		l.SellerID, l.CategoryID, l.Title, l.Description, l.Condition, l.Price, l.Status,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
}

func (r *postgresListingRepository) GetByID(ctx context.Context, id string) (*models.Listing, error) {
	query := `
		SELECT l.id, l.seller_id, l.category_id, l.title, l.description, l.condition,
		       l.price, l.suggested_price, l.status, l.created_at, l.updated_at, l.deleted_at,
		       u.id, u.name, u.dorm_building, u.avatar_url, u.trust_score
		FROM listings l
		JOIN users u ON u.id = l.seller_id
		WHERE l.id = $1`

	var l models.Listing
	var seller models.PublicUser
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&l.ID, &l.SellerID, &l.CategoryID, &l.Title, &l.Description, &l.Condition,
		&l.Price, &l.SuggestedPrice, &l.Status, &l.CreatedAt, &l.UpdatedAt, &l.DeletedAt,
		&seller.ID, &seller.Name, &seller.DormBuilding, &seller.AvatarURL, &seller.TrustScore,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	l.Seller = &seller

	images, err := r.ListImages(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	l.Images = images

	return &l, nil
}

// List คืนรายการประกาศตามเงื่อนไข filter โดย build WHERE clause แบบ dynamic
// ปลอดภัยจาก SQL injection เพราะใช้ parameterized query ($1, $2, ...) เสมอ ไม่มีการต่อ string ค่าดิบ
func (r *postgresListingRepository) List(ctx context.Context, filter models.ListingFilter) ([]models.Listing, error) {
	var conditions []string
	var args []any
	argN := 1

	// ประกาศที่ถูกลบ (soft delete) ต้องไม่โผล่ในผลค้นหา/รายการเด็ดขาด ไม่ว่า filter อื่นจะเป็นอะไร
	conditions = append(conditions, "l.deleted_at IS NULL")

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("l.status = $%d", argN))
		args = append(args, filter.Status)
		argN++
	}
	if filter.CategoryID != "" {
		conditions = append(conditions, fmt.Sprintf("l.category_id = $%d", argN))
		args = append(args, filter.CategoryID)
		argN++
	}
	if filter.SellerID != "" {
		conditions = append(conditions, fmt.Sprintf("l.seller_id = $%d", argN))
		args = append(args, filter.SellerID)
		argN++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(l.title ILIKE $%d OR l.description ILIKE $%d)", argN, argN))
		args = append(args, "%"+filter.Search+"%")
		argN++
	}
	if filter.MinPrice != nil {
		conditions = append(conditions, fmt.Sprintf("l.price >= $%d", argN))
		args = append(args, *filter.MinPrice)
		argN++
	}
	if filter.MaxPrice != nil {
		conditions = append(conditions, fmt.Sprintf("l.price <= $%d", argN))
		args = append(args, *filter.MaxPrice)
		argN++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 24
	}

	query := fmt.Sprintf(`
		SELECT l.id, l.seller_id, l.category_id, l.title, l.description, l.condition,
		       l.price, l.suggested_price, l.status, l.created_at, l.updated_at,
		       u.id, u.name, u.dorm_building, u.avatar_url, u.trust_score,
		       COALESCE(
		         (SELECT url FROM listing_images WHERE listing_id = l.id ORDER BY sort_order LIMIT 1),
		         ''
		       ) AS cover_image
		FROM listings l
		JOIN users u ON u.id = l.seller_id
		%s
		ORDER BY l.created_at DESC
		LIMIT $%d OFFSET $%d`, where, argN, argN+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []models.Listing
	for rows.Next() {
		var l models.Listing
		var seller models.PublicUser
		var coverImage string
		if err := rows.Scan(
			&l.ID, &l.SellerID, &l.CategoryID, &l.Title, &l.Description, &l.Condition,
			&l.Price, &l.SuggestedPrice, &l.Status, &l.CreatedAt, &l.UpdatedAt,
			&seller.ID, &seller.Name, &seller.DormBuilding, &seller.AvatarURL, &seller.TrustScore,
			&coverImage,
		); err != nil {
			return nil, err
		}
		l.Seller = &seller
		if coverImage != "" {
			l.Images = []models.ListingImage{{URL: coverImage}}
		}
		listings = append(listings, l)
	}
	return listings, rows.Err()
}

func (r *postgresListingRepository) AddImage(ctx context.Context, img *models.ListingImage) error {
	query := `
		INSERT INTO listing_images (listing_id, url, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id`
	return r.db.QueryRowContext(ctx, query, img.ListingID, img.URL, img.SortOrder).Scan(&img.ID)
}

func (r *postgresListingRepository) ListImages(ctx context.Context, listingID string) ([]models.ListingImage, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, listing_id, url, sort_order FROM listing_images WHERE listing_id = $1 ORDER BY sort_order`,
		listingID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []models.ListingImage
	for rows.Next() {
		var img models.ListingImage
		if err := rows.Scan(&img.ID, &img.ListingID, &img.URL, &img.SortOrder); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

func (r *postgresListingRepository) UpdateStatus(ctx context.Context, id string, sellerID string, status models.ListingStatus) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE listings SET status = $1, updated_at = now() WHERE id = $2 AND seller_id = $3`,
		status, id, sellerID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// Update แก้ไขข้อมูลประกาศ (ชื่อ, รายละเอียด, หมวดหมู่, สภาพ, ราคา) — เฉพาะเจ้าของเท่านั้น
// (เช็คจาก seller_id ใน WHERE clause ตรงๆ แบบเดียวกับ UpdateStatus — ถ้าไม่ใช่เจ้าของหรือ
// ไม่พบประกาศ จะได้ rows affected = 0 เหมือนกัน ไม่บอกความต่างเพื่อความปลอดภัย)
func (r *postgresListingRepository) Update(ctx context.Context, l *models.Listing) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE listings
		SET title = $1, description = $2, category_id = $3, condition = $4, price = $5, updated_at = now()
		WHERE id = $6 AND seller_id = $7 AND deleted_at IS NULL`,
		l.Title, l.Description, l.CategoryID, l.Condition, l.Price, l.ID, l.SellerID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// SoftDelete ซ่อนประกาศจากการค้นหา/รายการ โดยไม่ลบข้อมูลจริง (เก็บประวัติแชท/รีวิว/shipment ไว้)
func (r *postgresListingRepository) SoftDelete(ctx context.Context, id string, sellerID string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE listings SET deleted_at = now() WHERE id = $1 AND seller_id = $2 AND deleted_at IS NULL`,
		id, sellerID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// SuggestPrice คำนวณราคาแนะนำแบบ rule-based จากค่าเฉลี่ยของประกาศอื่นในหมวดหมู่+สภาพเดียวกัน
// ที่มีอยู่แล้วในระบบ (ไม่ว่าสถานะไหนก็ตาม เพราะแม้แต่ของที่ขายไปแล้วก็เป็นสัญญาณราคาที่ดี)
func (r *postgresListingRepository) SuggestPrice(ctx context.Context, categoryID string, condition models.ListingCondition) (*models.PriceSuggestion, error) {
	query := `
		SELECT COUNT(*), COALESCE(AVG(price), 0), COALESCE(MIN(price), 0), COALESCE(MAX(price), 0)
		FROM listings
		WHERE category_id = $1 AND condition = $2`

	var suggestion models.PriceSuggestion
	var avg float64
	err := r.db.QueryRowContext(ctx, query, categoryID, condition).Scan(
		&suggestion.SampleSize, &avg, &suggestion.MinPrice, &suggestion.MaxPrice,
	)
	if err != nil {
		return nil, err
	}
	suggestion.SuggestedPrice = int(avg)
	return &suggestion, nil
}

// formatVector แปลง []float32 เป็น text format ที่ pgvector เข้าใจ เช่น "[0.12,-0.34,0.56]"
// จำเป็นเพราะใช้ lib/pq (database/sql) ธรรมดา ไม่มี native vector type support แบบ pgx
// ต้อง cast เป็น string แล้วให้ Postgres แปลงเองด้วย ::vector ใน query
func formatVector(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// SetImageEmbedding บันทึก embedding vector ของรูปสินค้าหนึ่งรูป (เรียกจาก service หลัง
// ได้ embedding กลับมาจาก ml-service ตอนอัปโหลดรูป)
func (r *postgresListingRepository) SetImageEmbedding(ctx context.Context, imageID string, embedding []float32) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE listing_images SET embedding = $1::vector WHERE id = $2`,
		formatVector(embedding), imageID,
	)
	return err
}

// SearchBySimilarListings หาประกาศที่มีรูปคล้ายกับ embedding ที่ส่งมามากที่สุด
// ใช้ cosine distance (<=>) ของ pgvector — ยิ่งค่าน้อยยิ่งคล้าย เทียบแค่รูปแรกที่ใกล้ที่สุด
// ของแต่ละประกาศ (กันประกาศเดียวถูกนับซ้ำหลายครั้งเพราะมีหลายรูป)
func (r *postgresListingRepository) SearchBySimilarListings(ctx context.Context, embedding []float32, limit int) ([]models.Listing, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	query := `
		SELECT l.id, l.seller_id, l.category_id, l.title, l.description, l.condition,
		       l.price, l.suggested_price, l.status, l.created_at, l.updated_at, l.deleted_at,
		       u.id, u.name, u.dorm_building, u.avatar_url, u.trust_score,
		       MIN(li.embedding <=> $1::vector) AS distance
		FROM listing_images li
		JOIN listings l ON l.id = li.listing_id
		JOIN users u ON u.id = l.seller_id
		WHERE li.embedding IS NOT NULL AND l.deleted_at IS NULL AND l.status = 'available'
		GROUP BY l.id, u.id
		ORDER BY distance ASC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, formatVector(embedding), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []models.Listing
	for rows.Next() {
		var l models.Listing
		var seller models.PublicUser
		var distance float64
		if err := rows.Scan(
			&l.ID, &l.SellerID, &l.CategoryID, &l.Title, &l.Description, &l.Condition,
			&l.Price, &l.SuggestedPrice, &l.Status, &l.CreatedAt, &l.UpdatedAt, &l.DeletedAt,
			&seller.ID, &seller.Name, &seller.DormBuilding, &seller.AvatarURL, &seller.TrustScore,
			&distance,
		); err != nil {
			return nil, err
		}
		l.Seller = &seller

		images, err := r.ListImages(ctx, l.ID)
		if err != nil {
			return nil, err
		}
		l.Images = images

		listings = append(listings, l)
	}
	return listings, rows.Err()
}
