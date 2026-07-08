package repository

import (
	"context"
	"database/sql"

	"dormmarket/internal/models"
)

type CategoryRepository interface {
	List(ctx context.Context) ([]models.Category, error)
}

type postgresCategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &postgresCategoryRepository{db: db}
}

func (r *postgresCategoryRepository) List(ctx context.Context) ([]models.Category, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, slug FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}
