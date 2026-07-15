package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/lib/pq" // driver PostgreSQL (seule dépendance externe autorisée)
)

//go:embed db/schema.sql
var schemaSQL string

// app porte les dépendances partagées par les handlers.
type app struct {
	db *sql.DB
}

// openDB se connecte à PostgreSQL et applique le schéma (idempotent).
func openDB(databaseURL string) (*sql.DB, error) {
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
	return db, nil
}

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

// creditBalance calcule le solde d'un utilisateur (hors transaction).
func (a *app) creditBalance(ctx context.Context, userID int) (int, error) {
	return balance(ctx, a.db, userID)
}

// userExists vérifie qu'un utilisateur existe.
func (a *app) userExists(ctx context.Context, id int) error {
	var ok bool
	err := a.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return ErrIntrouvable
	}
	return nil
}
