package repository

import (
	"context"
	"database/sql"

	"dormmarket/internal/models"
)

type TicketRepository interface {
	CreateTicket(ctx context.Context, ticket *models.SupportTicket) error
	GetTicket(ctx context.Context, id string) (*models.SupportTicket, error)
	ListForUser(ctx context.Context, userID string) ([]models.SupportTicket, error)
	// ListAll คืนทุก ticket เรียงใหม่สุดก่อน — status เป็น nil แปลว่าเอาทุกสถานะ (สำหรับแอดมิน)
	ListAll(ctx context.Context, status *models.TicketStatus) ([]models.SupportTicket, error)
	UpdateStatus(ctx context.Context, id string, status models.TicketStatus) error
	AddMessage(ctx context.Context, msg *models.TicketMessage) error
	ListMessages(ctx context.Context, ticketID string) ([]models.TicketMessage, error)
}

type postgresTicketRepository struct {
	db *sql.DB
}

func NewTicketRepository(db *sql.DB) TicketRepository {
	return &postgresTicketRepository{db: db}
}

func (r *postgresTicketRepository) CreateTicket(ctx context.Context, ticket *models.SupportTicket) error {
	query := `
		INSERT INTO support_tickets (user_id, subject)
		VALUES ($1, $2)
		RETURNING id, status, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, ticket.UserID, ticket.Subject).
		Scan(&ticket.ID, &ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt)
}

func scanTicket(row *sql.Row) (*models.SupportTicket, error) {
	var t models.SupportTicket
	var user models.PublicUser
	err := row.Scan(
		&t.ID, &t.UserID, &t.Subject, &t.Status, &t.CreatedAt, &t.UpdatedAt,
		&user.ID, &user.Name, &user.DormBuilding, &user.AvatarURL, &user.TrustScore,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	t.User = &user
	return &t, nil
}

const ticketSelectQuery = `
	SELECT t.id, t.user_id, t.subject, t.status, t.created_at, t.updated_at,
	       u.id, u.name, u.dorm_building, u.avatar_url, u.trust_score
	FROM support_tickets t
	JOIN users u ON u.id = t.user_id`

func (r *postgresTicketRepository) GetTicket(ctx context.Context, id string) (*models.SupportTicket, error) {
	return scanTicket(r.db.QueryRowContext(ctx, ticketSelectQuery+` WHERE t.id = $1`, id))
}

func (r *postgresTicketRepository) ListForUser(ctx context.Context, userID string) ([]models.SupportTicket, error) {
	rows, err := r.db.QueryContext(ctx, ticketSelectQuery+` WHERE t.user_id = $1 ORDER BY t.updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	return scanTicketRows(rows)
}

func (r *postgresTicketRepository) ListAll(ctx context.Context, status *models.TicketStatus) ([]models.SupportTicket, error) {
	query := ticketSelectQuery
	args := []any{}
	if status != nil {
		query += ` WHERE t.status = $1`
		args = append(args, *status)
	}
	query += ` ORDER BY t.updated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return scanTicketRows(rows)
}

func scanTicketRows(rows *sql.Rows) ([]models.SupportTicket, error) {
	defer rows.Close()
	var tickets []models.SupportTicket
	for rows.Next() {
		var t models.SupportTicket
		var user models.PublicUser
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Subject, &t.Status, &t.CreatedAt, &t.UpdatedAt,
			&user.ID, &user.Name, &user.DormBuilding, &user.AvatarURL, &user.TrustScore,
		); err != nil {
			return nil, err
		}
		t.User = &user
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (r *postgresTicketRepository) UpdateStatus(ctx context.Context, id string, status models.TicketStatus) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE support_tickets SET status = $1, updated_at = now() WHERE id = $2`,
		status, id,
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

func (r *postgresTicketRepository) AddMessage(ctx context.Context, msg *models.TicketMessage) error {
	query := `
		INSERT INTO ticket_messages (ticket_id, sender_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	if err := r.db.QueryRowContext(ctx, query, msg.TicketID, msg.SenderID, msg.Body).
		Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `UPDATE support_tickets SET updated_at = now() WHERE id = $1`, msg.TicketID)
	return err
}

func (r *postgresTicketRepository) ListMessages(ctx context.Context, ticketID string) ([]models.TicketMessage, error) {
	query := `
		SELECT m.id, m.ticket_id, m.sender_id, m.body, m.created_at,
		       u.id, u.name, u.dorm_building, u.avatar_url, u.trust_score
		FROM ticket_messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.ticket_id = $1
		ORDER BY m.created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.TicketMessage
	for rows.Next() {
		var m models.TicketMessage
		var sender models.PublicUser
		if err := rows.Scan(
			&m.ID, &m.TicketID, &m.SenderID, &m.Body, &m.CreatedAt,
			&sender.ID, &sender.Name, &sender.DormBuilding, &sender.AvatarURL, &sender.TrustScore,
		); err != nil {
			return nil, err
		}
		m.Sender = &sender
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
