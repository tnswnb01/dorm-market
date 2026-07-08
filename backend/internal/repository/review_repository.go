package repository

import (
	"context"
	"database/sql"

	"dormmarket/internal/models"
)

type ReviewRepository interface {
	Create(ctx context.Context, review *models.Review) error
	HasReviewed(ctx context.Context, listingID, reviewerID string) (bool, error)
	ListForUser(ctx context.Context, revieweeID string) ([]models.Review, error)
	// RecomputeTrustScore คำนวณ trust_score ใหม่จากค่าเฉลี่ยดาวทั้งหมดที่ user นี้เคยได้รับ
	// สูตร: trust_score = round(ค่าเฉลี่ยดาว / 5 * 100) เช่น เฉลี่ย 4.5 ดาว -> trust_score 90
	RecomputeTrustScore(ctx context.Context, userID string) error
}

type postgresReviewRepository struct {
	db *sql.DB
}

func NewReviewRepository(db *sql.DB) ReviewRepository {
	return &postgresReviewRepository{db: db}
}

func (r *postgresReviewRepository) Create(ctx context.Context, review *models.Review) error {
	query := `
		INSERT INTO reviews (listing_id, reviewer_id, reviewee_id, rating, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query,
		review.ListingID, review.ReviewerID, review.RevieweeID, review.Rating, review.Comment,
	).Scan(&review.ID, &review.CreatedAt)
}

func (r *postgresReviewRepository) HasReviewed(ctx context.Context, listingID, reviewerID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM reviews WHERE listing_id = $1 AND reviewer_id = $2)`,
		listingID, reviewerID,
	).Scan(&exists)
	return exists, err
}

func (r *postgresReviewRepository) ListForUser(ctx context.Context, revieweeID string) ([]models.Review, error) {
	query := `
		SELECT r.id, r.listing_id, r.reviewer_id, r.reviewee_id, r.rating, r.comment, r.created_at,
		       u.id, u.name, u.dorm_building, u.avatar_url, u.trust_score
		FROM reviews r
		JOIN users u ON u.id = r.reviewer_id
		WHERE r.reviewee_id = $1
		ORDER BY r.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, revieweeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []models.Review
	for rows.Next() {
		var rv models.Review
		var reviewer models.PublicUser
		if err := rows.Scan(
			&rv.ID, &rv.ListingID, &rv.ReviewerID, &rv.RevieweeID, &rv.Rating, &rv.Comment, &rv.CreatedAt,
			&reviewer.ID, &reviewer.Name, &reviewer.DormBuilding, &reviewer.AvatarURL, &reviewer.TrustScore,
		); err != nil {
			return nil, err
		}
		rv.Reviewer = &reviewer
		reviews = append(reviews, rv)
	}
	return reviews, rows.Err()
}

func (r *postgresReviewRepository) RecomputeTrustScore(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET trust_score = (
			SELECT ROUND(AVG(rating) / 5 * 100)
			FROM reviews
			WHERE reviewee_id = $1
		)
		WHERE id = $1`,
		userID,
	)
	return err
}
