package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ChanFrancis/GO_BarterSwap/internal/barterswap"
)

// InsertUser crée l'utilisateur et son crédit de bienvenue dans la même
// transaction.
func (s *Store) InsertUser(ctx context.Context, pseudo, bio, ville string) (barterswap.User, error) {
	var u barterswap.User
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return u, err
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO users (pseudo, bio, ville) VALUES ($1, $2, $3)
		 RETURNING id, pseudo, bio, ville, created_at`,
		pseudo, bio, ville).
		Scan(&u.ID, &u.Pseudo, &u.Bio, &u.Ville, &u.CreatedAt)
	if err != nil {
		return u, err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO credit_transactions (user_id, montant, type) VALUES ($1, $2, 'earn')`,
		u.ID, barterswap.CreditsBienvenue)
	if err != nil {
		return u, err
	}

	u.CreditBalance = barterswap.CreditsBienvenue
	return u, tx.Commit()
}

// FetchUser retourne le profil complet : infos, compétences et solde.
func (s *Store) FetchUser(ctx context.Context, id int) (barterswap.User, error) {
	var u barterswap.User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, pseudo, bio, ville, created_at FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Pseudo, &u.Bio, &u.Ville, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return u, barterswap.ErrIntrouvable
	}
	if err != nil {
		return u, err
	}
	if u.Skills, err = s.FetchSkills(ctx, id); err != nil {
		return u, err
	}
	if u.CreditBalance, err = s.creditBalance(ctx, id); err != nil {
		return u, err
	}
	return u, nil
}

// UpdateUser modifie le profil d'un utilisateur existant.
func (s *Store) UpdateUser(ctx context.Context, id int, pseudo, bio, ville string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET pseudo = $1, bio = $2, ville = $3 WHERE id = $4`,
		pseudo, bio, ville, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return barterswap.ErrIntrouvable
	}
	return nil
}

// FetchSkills retourne les compétences d'un utilisateur.
func (s *Store) FetchSkills(ctx context.Context, userID int) ([]barterswap.Skill, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT nom, niveau FROM skills WHERE user_id = $1 ORDER BY nom`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	skills := []barterswap.Skill{}
	for rows.Next() {
		var sk barterswap.Skill
		if err := rows.Scan(&sk.Nom, &sk.Niveau); err != nil {
			return nil, err
		}
		skills = append(skills, sk)
	}
	return skills, rows.Err()
}

// ReplaceSkills écrase toutes les compétences de l'utilisateur (règle du
// sujet : pas d'ajout individuel).
func (s *Store) ReplaceSkills(ctx context.Context, userID int, skills []barterswap.Skill) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM skills WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, sk := range skills {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO skills (user_id, nom, niveau) VALUES ($1, $2, $3)`,
			userID, sk.Nom, sk.Niveau); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UserHasSkill indique si l'utilisateur possède une compétence dont le nom
// correspond à la catégorie donnée.
func (s *Store) UserHasSkill(ctx context.Context, userID int, categorie string) (bool, error) {
	var ok bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM skills WHERE user_id = $1 AND nom = $2)`,
		userID, categorie).Scan(&ok)
	return ok, err
}

// UserExists vérifie qu'un utilisateur existe.
func (s *Store) UserExists(ctx context.Context, id int) error {
	var ok bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return barterswap.ErrIntrouvable
	}
	return nil
}

// creditBalance calcule le solde d'un utilisateur (hors transaction).
func (s *Store) creditBalance(ctx context.Context, userID int) (int, error) {
	return balance(ctx, s.db, userID)
}
