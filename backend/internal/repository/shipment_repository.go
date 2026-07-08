package repository

import (
	"context"
	"database/sql"
	"errors"

	"dormmarket/internal/models"
)

type ShipmentRepository interface {
	Create(ctx context.Context, s *models.Shipment) error
	GetByConversationID(ctx context.Context, conversationID string) (*models.Shipment, error)
	// UpdateStatus เปลี่ยนสถานะ + บันทึก event ลง timeline ในธุรกรรมเดียวกัน (atomic)
	UpdateStatus(ctx context.Context, shipmentID string, status models.ShipmentStatus, note string) error
}

type postgresShipmentRepository struct {
	db *sql.DB
}

func NewShipmentRepository(db *sql.DB) ShipmentRepository {
	return &postgresShipmentRepository{db: db}
}

func (r *postgresShipmentRepository) Create(ctx context.Context, s *models.Shipment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO shipments (conversation_id, method, courier_name, tracking_number, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	if err := tx.QueryRowContext(ctx, query,
		s.ConversationID, s.Method, s.CourierName, s.TrackingNumber, s.Status,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return err
	}

	if err := insertShipmentEvent(ctx, tx, s.ID, s.Status, "สร้างรายการติดตามสินค้า"); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *postgresShipmentRepository) GetByConversationID(ctx context.Context, conversationID string) (*models.Shipment, error) {
	query := `
		SELECT id, conversation_id, method, courier_name, tracking_number, status, created_at, updated_at
		FROM shipments WHERE conversation_id = $1`

	var s models.Shipment
	err := r.db.QueryRowContext(ctx, query, conversationID).Scan(
		&s.ID, &s.ConversationID, &s.Method, &s.CourierName, &s.TrackingNumber,
		&s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	events, err := r.listEvents(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.Events = events

	return &s, nil
}

func (r *postgresShipmentRepository) listEvents(ctx context.Context, shipmentID string) ([]models.ShipmentEvent, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, shipment_id, status, note, created_at
		 FROM shipment_events WHERE shipment_id = $1 ORDER BY created_at ASC`,
		shipmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.ShipmentEvent
	for rows.Next() {
		var e models.ShipmentEvent
		if err := rows.Scan(&e.ID, &e.ShipmentID, &e.Status, &e.Note, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *postgresShipmentRepository) UpdateStatus(ctx context.Context, shipmentID string, status models.ShipmentStatus, note string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`UPDATE shipments SET status = $1, updated_at = now() WHERE id = $2`,
		status, shipmentID,
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

	if err := insertShipmentEvent(ctx, tx, shipmentID, status, note); err != nil {
		return err
	}

	return tx.Commit()
}

func insertShipmentEvent(ctx context.Context, tx *sql.Tx, shipmentID string, status models.ShipmentStatus, note string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO shipment_events (shipment_id, status, note) VALUES ($1, $2, $3)`,
		shipmentID, status, note,
	)
	return err
}
