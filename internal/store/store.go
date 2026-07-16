// Package store gère l'accès à la base PostgreSQL. Il traduit les lignes en
// entités du domaine (package barterswap) et applique les transactions et
// verrous nécessaires — c'est la base, et non un mutex, qui sérialise la
// concurrence.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/lib/pq" // driver PostgreSQL (seule dépendance externe autorisée)
)

//go:embed schema.sql
var schemaSQL string

// Store encapsule la connexion à la base.
type Store struct {
	db *sql.DB
}

// New se connecte à PostgreSQL et applique le schéma (idempotent).
func New(databaseURL string) (*Store, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("ouverture de la base : %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connexion à la base : %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("application du schéma : %w", err)
	}
	return &Store{db: db}, nil
}

// Close ferme la connexion à la base.
func (s *Store) Close() error { return s.db.Close() }

// DB expose la connexion sous-jacente (utile pour l'outillage et les tests).
func (s *Store) DB() *sql.DB { return s.db }

// rowQuerier couvre *sql.DB et *sql.Tx pour partager les calculs de solde.
type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// balance calcule le solde comme la somme du journal de transactions.
func balance(ctx context.Context, q rowQuerier, userID int) (int, error) {
	var solde int
	err := q.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(montant), 0) FROM credit_transactions WHERE user_id = $1`,
		userID).Scan(&solde)
	return solde, err
}
