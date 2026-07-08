package repository

import (
	"context"
	"database/sql"
	"errors"

	"dormmarket/internal/models"
)

var ErrNotFound = errors.New("record not found")

// UserRepository คือ interface สำหรับเข้าถึงข้อมูล user ทั้งหมด
// การแยกเป็น interface ทำให้ swap implementation (เช่น เปลี่ยนไปใช้ DB อื่น หรือ mock ตอนเทส) ได้ง่าย
type UserRepository interface {
	Create(ctx context.Context, u *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*models.User, error)
	LinkGoogleID(ctx context.Context, userID string, googleID string) error
}

type postgresUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Create(ctx context.Context, u *models.User) error {
	query := `
		INSERT INTO users (email, password_hash, name, dorm_building, google_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, avatar_url, trust_score, created_at`
	return r.db.QueryRowContext(ctx, query, u.Email, u.PasswordHash, u.Name, u.DormBuilding, u.GoogleID).
		Scan(&u.ID, &u.AvatarURL, &u.TrustScore, &u.CreatedAt)
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, dorm_building, avatar_url, trust_score, created_at, google_id
		FROM users WHERE email = $1`
	return scanUser(r.db.QueryRowContext(ctx, query, email))
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, dorm_building, avatar_url, trust_score, created_at, google_id
		FROM users WHERE id = $1`
	return scanUser(r.db.QueryRowContext(ctx, query, id))
}

func (r *postgresUserRepository) GetByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, dorm_building, avatar_url, trust_score, created_at, google_id
		FROM users WHERE google_id = $1`
	return scanUser(r.db.QueryRowContext(ctx, query, googleID))
}

// LinkGoogleID ผูกบัญชี Google เข้ากับ user ที่สมัครด้วยอีเมล/รหัสผ่านไว้อยู่แล้ว
// (เคสที่ email ตรงกันระหว่างสมัครแบบเดิมกับ login ผ่าน Google ครั้งแรก)
func (r *postgresUserRepository) LinkGoogleID(ctx context.Context, userID string, googleID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET google_id = $1 WHERE id = $2`, googleID, userID)
	return err
}

func scanUser(row *sql.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.DormBuilding,
		&u.AvatarURL, &u.TrustScore, &u.CreatedAt, &u.GoogleID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
