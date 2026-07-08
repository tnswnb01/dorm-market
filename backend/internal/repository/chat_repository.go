package repository

import (
	"context"
	"database/sql"
	"errors"

	"dormmarket/internal/models"
)

type ChatRepository interface {
	// GetOrCreateConversation คืน conversation ที่มีอยู่แล้วถ้าเจอ (listing, buyer) คู่นี้แล้ว
	// ไม่งั้นสร้างใหม่ — ใช้ ON CONFLICT กัน race condition ตอนมีคน click "ติดต่อผู้ขาย" พร้อมกันซ้ำๆ
	GetOrCreateConversation(ctx context.Context, listingID, buyerID, sellerID string) (*models.Conversation, error)
	GetConversation(ctx context.Context, id string) (*models.Conversation, error)
	ListConversationsForUser(ctx context.Context, userID string) ([]models.Conversation, error)
	// HasConversation เช็คว่าเคยมี conversation ระหว่าง buyer กับ listing นี้ไหม
	// ใช้เป็นเงื่อนไขว่าใครมีสิทธิ์รีวิว (ต้องเคยคุยกับผู้ขายจริงก่อน กันรีวิวปลอม)
	HasConversation(ctx context.Context, listingID, buyerID string) (bool, error)

	CreateMessage(ctx context.Context, msg *models.Message) error
	ListMessages(ctx context.Context, conversationID string) ([]models.Message, error)
}

type postgresChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) ChatRepository {
	return &postgresChatRepository{db: db}
}

func (r *postgresChatRepository) GetOrCreateConversation(ctx context.Context, listingID, buyerID, sellerID string) (*models.Conversation, error) {
	query := `
		INSERT INTO conversations (listing_id, buyer_id, seller_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (listing_id, buyer_id) DO UPDATE SET listing_id = EXCLUDED.listing_id
		RETURNING id, listing_id, buyer_id, seller_id, created_at, last_message_at`

	var c models.Conversation
	err := r.db.QueryRowContext(ctx, query, listingID, buyerID, sellerID).Scan(
		&c.ID, &c.ListingID, &c.BuyerID, &c.SellerID, &c.CreatedAt, &c.LastMessageAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *postgresChatRepository) GetConversation(ctx context.Context, id string) (*models.Conversation, error) {
	query := `
		SELECT id, listing_id, buyer_id, seller_id, created_at, last_message_at
		FROM conversations WHERE id = $1`

	var c models.Conversation
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.ListingID, &c.BuyerID, &c.SellerID, &c.CreatedAt, &c.LastMessageAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListConversationsForUser คืนทุกห้องแชทที่ user เป็นผู้ซื้อหรือผู้ขาย พร้อมข้อมูล
// อีกฝ่าย + สินค้า + ข้อความล่าสุด (ใช้แสดงเป็นหน้า inbox) เรียงจากคุยล่าสุดก่อน
func (r *postgresChatRepository) ListConversationsForUser(ctx context.Context, userID string) ([]models.Conversation, error) {
	query := `
		SELECT
			c.id, c.listing_id, c.buyer_id, c.seller_id, c.created_at, c.last_message_at,
			l.title, l.price, l.status,
			COALESCE(
			  (SELECT url FROM listing_images WHERE listing_id = l.id ORDER BY sort_order LIMIT 1),
			  ''
			) AS cover_image,
			other.id, other.name, other.dorm_building, other.avatar_url, other.trust_score,
			lm.id, lm.sender_id, lm.content, lm.created_at
		FROM conversations c
		JOIN listings l ON l.id = c.listing_id
		JOIN users other ON other.id = CASE WHEN c.buyer_id = $1 THEN c.seller_id ELSE c.buyer_id END
		LEFT JOIN LATERAL (
			SELECT id, sender_id, content, created_at
			FROM messages
			WHERE conversation_id = c.id
			ORDER BY created_at DESC
			LIMIT 1
		) lm ON true
		WHERE c.buyer_id = $1 OR c.seller_id = $1
		ORDER BY c.last_message_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []models.Conversation
	for rows.Next() {
		var c models.Conversation
		var listing models.Listing
		var coverImage string
		var other models.PublicUser
		var msgID, msgSenderID, msgContent sql.NullString
		var msgCreatedAt sql.NullTime

		if err := rows.Scan(
			&c.ID, &c.ListingID, &c.BuyerID, &c.SellerID, &c.CreatedAt, &c.LastMessageAt,
			&listing.Title, &listing.Price, &listing.Status, &coverImage,
			&other.ID, &other.Name, &other.DormBuilding, &other.AvatarURL, &other.TrustScore,
			&msgID, &msgSenderID, &msgContent, &msgCreatedAt,
		); err != nil {
			return nil, err
		}

		listing.ID = c.ListingID
		if coverImage != "" {
			listing.Images = []models.ListingImage{{URL: coverImage}}
		}
		c.Listing = &listing
		c.OtherParty = &other

		if msgID.Valid {
			c.LastMessage = &models.Message{
				ID:             msgID.String,
				ConversationID: c.ID,
				SenderID:       msgSenderID.String,
				Content:        msgContent.String,
				CreatedAt:      msgCreatedAt.Time,
			}
		}

		conversations = append(conversations, c)
	}
	return conversations, rows.Err()
}

func (r *postgresChatRepository) HasConversation(ctx context.Context, listingID, buyerID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM conversations WHERE listing_id = $1 AND buyer_id = $2)`,
		listingID, buyerID,
	).Scan(&exists)
	return exists, err
}

func (r *postgresChatRepository) CreateMessage(ctx context.Context, msg *models.Message) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO messages (conversation_id, sender_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	if err := tx.QueryRowContext(ctx, query, msg.ConversationID, msg.SenderID, msg.Content).
		Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE conversations SET last_message_at = $1 WHERE id = $2`,
		msg.CreatedAt, msg.ConversationID,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *postgresChatRepository) ListMessages(ctx context.Context, conversationID string) ([]models.Message, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, conversation_id, sender_id, content, created_at
		 FROM messages WHERE conversation_id = $1 ORDER BY created_at ASC`,
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
