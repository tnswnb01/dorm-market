package repository

import (
	"context"
	"database/sql"

	"dormmarket/internal/models"
)

type ReportRepository interface {
	Create(ctx context.Context, report *models.Report) error
	GetByID(ctx context.Context, id string) (*models.Report, error)
	// List คืนรายการ report เรียงใหม่สุดก่อน — status เป็น nil แปลว่าเอาทุกสถานะ
	List(ctx context.Context, status *models.ReportStatus) ([]models.Report, error)
	Resolve(ctx context.Context, id string, status models.ReportStatus, action models.ReportResolutionAction, note, adminID string) error
}

type postgresReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) ReportRepository {
	return &postgresReportRepository{db: db}
}

func (r *postgresReportRepository) Create(ctx context.Context, report *models.Report) error {
	query := `
		INSERT INTO reports (reporter_id, target_type, target_listing_id, target_user_id, reason, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, status, created_at`
	return r.db.QueryRowContext(ctx, query,
		report.ReporterID, report.TargetType, report.TargetListingID, report.TargetUserID,
		report.Reason, report.Description,
	).Scan(&report.ID, &report.Status, &report.CreatedAt)
}

const reportSelectColumns = `
	r.id, r.reporter_id, r.target_type, r.target_listing_id, r.target_user_id, r.reason,
	r.description, r.status, r.resolution_action, r.resolution_note, r.resolved_by, r.resolved_at, r.created_at,
	reporter.id, reporter.name, reporter.dorm_building, reporter.avatar_url, reporter.trust_score,
	target_user.id, target_user.name, target_user.dorm_building, target_user.avatar_url, target_user.trust_score,
	target_listing.id, target_listing.title, target_listing.price, target_listing.status`

func scanReport(row interface {
	Scan(dest ...any) error
}) (*models.Report, error) {
	var rp models.Report
	var reporter models.PublicUser
	var targetUser models.PublicUser
	var targetUserID, targetUserName, targetUserDorm, targetUserAvatar sql.NullString
	var targetUserTrust sql.NullInt64
	var targetListingID, targetListingTitle, targetListingStatus sql.NullString
	var targetListingPrice sql.NullInt64

	err := row.Scan(
		&rp.ID, &rp.ReporterID, &rp.TargetType, &rp.TargetListingID, &rp.TargetUserID, &rp.Reason,
		&rp.Description, &rp.Status, &rp.ResolutionAction, &rp.ResolutionNote, &rp.ResolvedBy, &rp.ResolvedAt, &rp.CreatedAt,
		&reporter.ID, &reporter.Name, &reporter.DormBuilding, &reporter.AvatarURL, &reporter.TrustScore,
		&targetUserID, &targetUserName, &targetUserDorm, &targetUserAvatar, &targetUserTrust,
		&targetListingID, &targetListingTitle, &targetListingPrice, &targetListingStatus,
	)
	if err != nil {
		return nil, err
	}

	rp.Reporter = &reporter
	if targetUserID.Valid {
		targetUser = models.PublicUser{
			ID: targetUserID.String, Name: targetUserName.String,
			DormBuilding: targetUserDorm.String, AvatarURL: targetUserAvatar.String,
			TrustScore: int(targetUserTrust.Int64),
		}
		rp.TargetUser = &targetUser
	}
	if targetListingID.Valid {
		rp.TargetListing = &models.Listing{
			ID: targetListingID.String, Title: targetListingTitle.String,
			Price: int(targetListingPrice.Int64), Status: models.ListingStatus(targetListingStatus.String),
		}
	}
	return &rp, nil
}

func (r *postgresReportRepository) GetByID(ctx context.Context, id string) (*models.Report, error) {
	query := `
		SELECT ` + reportSelectColumns + `
		FROM reports r
		JOIN users reporter ON reporter.id = r.reporter_id
		LEFT JOIN users target_user ON target_user.id = r.target_user_id
		LEFT JOIN listings target_listing ON target_listing.id = r.target_listing_id
		WHERE r.id = $1`
	rp, err := scanReport(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return rp, err
}

func (r *postgresReportRepository) List(ctx context.Context, status *models.ReportStatus) ([]models.Report, error) {
	query := `
		SELECT ` + reportSelectColumns + `
		FROM reports r
		JOIN users reporter ON reporter.id = r.reporter_id
		LEFT JOIN users target_user ON target_user.id = r.target_user_id
		LEFT JOIN listings target_listing ON target_listing.id = r.target_listing_id`
	args := []any{}
	if status != nil {
		query += ` WHERE r.status = $1`
		args = append(args, *status)
	}
	query += ` ORDER BY r.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []models.Report
	for rows.Next() {
		rp, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, *rp)
	}
	return reports, rows.Err()
}

func (r *postgresReportRepository) Resolve(ctx context.Context, id string, status models.ReportStatus, action models.ReportResolutionAction, note, adminID string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE reports
		SET status = $1, resolution_action = $2, resolution_note = $3, resolved_by = $4, resolved_at = now()
		WHERE id = $5 AND status = 'pending'`,
		status, action, note, adminID, id,
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
