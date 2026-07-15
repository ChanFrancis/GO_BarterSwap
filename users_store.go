package main

import (
	"context"
	"database/sql"
	"errors"
)

// Accès base de données des utilisateurs et de leurs compétences.

// insertUser crée l'utilisateur et son crédit de bienvenue dans la même
// transaction.
func (a *app) insertUser(ctx context.Context, pseudo, bio, ville string) (User, error) {
	var u User
	tx, err := a.db.BeginTx(ctx, nil)
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
		u.ID, CreditsBienvenue)
	if err != nil {
		return u, err
	}

	u.CreditBalance = CreditsBienvenue
	return u, tx.Commit()
}

// fetchUser retourne le profil complet : infos, compétences et solde.
func (a *app) fetchUser(ctx context.Context, id int) (User, error) {
	var u User
	err := a.db.QueryRowContext(ctx,
		`SELECT id, pseudo, bio, ville, created_at FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Pseudo, &u.Bio, &u.Ville, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrIntrouvable
	}
	if err != nil {
		return u, err
	}
	if u.Skills, err = a.fetchSkills(ctx, id); err != nil {
		return u, err
	}
	if u.CreditBalance, err = a.creditBalance(ctx, id); err != nil {
		return u, err
	}
	return u, nil
}

// updateUser modifie le profil d'un utilisateur existant.
func (a *app) updateUser(ctx context.Context, id int, pseudo, bio, ville string) error {
	res, err := a.db.ExecContext(ctx,
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
		return ErrIntrouvable
	}
	return nil
}

// fetchSkills retourne les compétences d'un utilisateur.
func (a *app) fetchSkills(ctx context.Context, userID int) ([]Skill, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT nom, niveau FROM skills WHERE user_id = $1 ORDER BY nom`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	skills := []Skill{}
	for rows.Next() {
		var s Skill
		if err := rows.Scan(&s.Nom, &s.Niveau); err != nil {
			return nil, err
		}
		skills = append(skills, s)
	}
	return skills, rows.Err()
}

// replaceSkills écrase toutes les compétences de l'utilisateur (règle du
// sujet : pas d'ajout individuel).
func (a *app) replaceSkills(ctx context.Context, userID int, skills []Skill) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM skills WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, s := range skills {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO skills (user_id, nom, niveau) VALUES ($1, $2, $3)`,
			userID, s.Nom, s.Niveau); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// userHasSkill indique si l'utilisateur possède une compétence dont le nom
// correspond à la catégorie donnée (règle : on ne propose un service que
// dans une catégorie où l'on a une compétence).
func (a *app) userHasSkill(ctx context.Context, userID int, categorie string) (bool, error) {
	var ok bool
	err := a.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM skills WHERE user_id = $1 AND nom = $2)`,
		userID, categorie).Scan(&ok)
	return ok, err
}
