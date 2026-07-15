package items

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Service porte l'accès aux données des objets.
type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, ownerID int64, in Input) (*Item, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	item := &Item{OwnerID: ownerID, Title: in.Title, Description: in.Description,
		Category: in.Category, Condition: in.Condition}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO items (owner_id, title, description, category, condition)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, status, created_at`,
		ownerID, in.Title, in.Description, in.Category, in.Condition).
		Scan(&item.ID, &item.Status, &item.CreatedAt)
	return item, err
}

func (s *Service) Get(ctx context.Context, id int64) (*Item, error) {
	item := &Item{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, owner_id, title, description, category, condition, status, created_at
		 FROM items WHERE id = $1`, id).
		Scan(&item.ID, &item.OwnerID, &item.Title, &item.Description,
			&item.Category, &item.Condition, &item.Status, &item.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (s *Service) Update(ctx context.Context, userID, id int64, in Input) (*Item, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	item, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.OwnerID != userID {
		return nil, ErrForbidden
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE items SET title = $1, description = $2, category = $3, condition = $4,
		 updated_at = now() WHERE id = $5`,
		in.Title, in.Description, in.Category, in.Condition, id)
	if err != nil {
		return nil, err
	}
	item.Title, item.Description, item.Category, item.Condition =
		in.Title, in.Description, in.Category, in.Condition
	return item, nil
}

func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	item, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if item.OwnerID != userID {
		return ErrForbidden
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM items WHERE id = $1`, id)
	return err
}

// Filter décrit une recherche dans le catalogue.
type Filter struct {
	Category string
	Query    string // recherche texte dans le titre et la description
	OwnerID  int64  // 0 = tous les propriétaires
	Page     int    // à partir de 1
	PerPage  int
}

// List retourne les objets disponibles correspondant au filtre, paginés.
func (s *Service) List(ctx context.Context, f Filter) ([]Item, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 50 {
		f.PerPage = 20
	}

	where := []string{"status = $1"}
	args := []any{StatusAvailable}
	if f.Category != "" {
		args = append(args, strings.ToLower(f.Category))
		where = append(where, fmt.Sprintf("category = $%d", len(args)))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", len(args), len(args)))
	}
	if f.OwnerID != 0 {
		args = append(args, f.OwnerID)
		where = append(where, fmt.Sprintf("owner_id = $%d", len(args)))
	}
	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT id, owner_id, title, description, category, condition, status, created_at
		 FROM items WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []Item{}
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.OwnerID, &item.Title, &item.Description,
			&item.Category, &item.Condition, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}
