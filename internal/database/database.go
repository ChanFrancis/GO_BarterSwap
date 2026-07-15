package database

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Open se connecte à PostgreSQL et applique les migrations manquantes.
func Open(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("ouverture de la base : %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connexion à la base : %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrations : %w", err)
	}
	return db, nil
}

// migrate applique les fichiers de migrations/ dans l'ordre alphabétique,
// en mémorisant ceux déjà appliqués dans la table schema_migrations.
func migrate(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`)
	if err != nil {
		return err
	}

	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		var applied bool
		err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)`,
			entry.Name()).Scan(&applied)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("%s : %w", entry.Name(), err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (name) VALUES ($1)`, entry.Name()); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
