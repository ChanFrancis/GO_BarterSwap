package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// Règles CNIL et durées de vie des tokens.
const (
	maxFailedLogins   = 5                // tentatives avant blocage du compte
	lockDuration      = 15 * time.Minute // durée du blocage
	passwordMaxAge    = 60 * 24 * time.Hour
	sessionDuration   = 24 * time.Hour
	resetTokenTimeout = 30 * time.Minute
)

var (
	ErrInvalidCredentials = errors.New("email ou mot de passe incorrect")
	ErrAccountLocked      = errors.New("compte temporairement bloqué suite à trop de tentatives, réessayez plus tard")
	ErrPasswordExpired    = errors.New("mot de passe expiré (plus de 60 jours), veuillez le réinitialiser")
	ErrEmailTaken         = errors.New("un compte existe déjà avec cet email")
	ErrInvalidEmail       = errors.New("adresse email invalide")
	ErrInvalidToken       = errors.New("lien invalide ou expiré")
)

// Mailer est l'interface d'envoi d'emails (SMTP en production, Mailpit en
// dev, doublure dans les tests).
type Mailer interface {
	Send(to, subject, body string) error
}

// Service porte toute la logique d'authentification.
type Service struct {
	db     *sql.DB
	mailer Mailer
	appURL string
}

func NewService(db *sql.DB, mailer Mailer, appURL string) *Service {
	return &Service{db: db, mailer: mailer, appURL: appURL}
}

// Register crée un compte après validation de l'email et de la politique de
// mot de passe fort.
func (s *Service) Register(ctx context.Context, email, password string) error {
	email, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	if err := ValidatePassword(password); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)`, email, hash)
	if err != nil {
		if strings.Contains(err.Error(), "users_email_key") {
			return ErrEmailTaken
		}
		return err
	}
	return nil
}

// Login vérifie les identifiants en appliquant le blocage après échecs et
// l'expiration du mot de passe à 60 jours, puis ouvre une session.
func (s *Service) Login(ctx context.Context, email, password string) (sessionToken string, err error) {
	email, err = normalizeEmail(email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	var (
		userID            int64
		passwordHash      string
		passwordChangedAt time.Time
		failedAttempts    int
		lockedUntil       sql.NullTime
	)
	err = s.db.QueryRowContext(ctx,
		`SELECT id, password_hash, password_changed_at, failed_login_attempts, locked_until
		 FROM users WHERE email = $1`, email).
		Scan(&userID, &passwordHash, &passwordChangedAt, &failedAttempts, &lockedUntil)
	if errors.Is(err, sql.ErrNoRows) {
		// Vérification factice pour ne pas révéler l'existence du compte
		// par une différence de temps de réponse.
		VerifyPassword(password, fakeHash)
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}

	if lockedUntil.Valid && time.Now().Before(lockedUntil.Time) {
		return "", ErrAccountLocked
	}

	ok, err := VerifyPassword(password, passwordHash)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", s.recordFailedLogin(ctx, userID, failedAttempts)
	}

	if time.Since(passwordChangedAt) > passwordMaxAge {
		return "", ErrPasswordExpired
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE users SET failed_login_attempts = 0, locked_until = NULL WHERE id = $1`, userID)
	if err != nil {
		return "", err
	}
	return s.createSession(ctx, userID)
}

// recordFailedLogin incrémente le compteur d'échecs et bloque le compte
// temporairement une fois le seuil atteint (exigence CNIL).
func (s *Service) recordFailedLogin(ctx context.Context, userID int64, previousFailures int) error {
	failures := previousFailures + 1
	var lockedUntil sql.NullTime
	if failures >= maxFailedLogins {
		lockedUntil = sql.NullTime{Time: time.Now().Add(lockDuration), Valid: true}
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET failed_login_attempts = $1, locked_until = $2 WHERE id = $3`,
		failures, lockedUntil, userID)
	if err != nil {
		return err
	}
	if lockedUntil.Valid {
		return ErrAccountLocked
	}
	return ErrInvalidCredentials
}

func (s *Service) createSession(ctx context.Context, userID int64) (string, error) {
	token, tokenHash, err := newToken()
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, time.Now().Add(sessionDuration))
	if err != nil {
		return "", err
	}
	return token, nil
}

// UserIDFromSession retourne l'utilisateur associé à une session valide.
func (s *Service) UserIDFromSession(ctx context.Context, token string) (int64, error) {
	var userID int64
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > now()`,
		hashToken(token)).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidToken
	}
	return userID, err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE token_hash = $1`, hashToken(token))
	return err
}

// RequestPasswordReset envoie un lien de réinitialisation si le compte
// existe. La réponse est identique dans tous les cas pour ne pas révéler
// quels emails sont inscrits.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil
	}

	var userID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	token, tokenHash, err := newToken()
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, time.Now().Add(resetTokenTimeout))
	if err != nil {
		return err
	}

	link := fmt.Sprintf("%s/reset-password?token=%s", s.appURL, token)
	body := fmt.Sprintf(
		"Bonjour,\r\n\r\nPour réinitialiser votre mot de passe BarterSwap, ouvrez ce lien (valable 30 minutes) :\r\n%s\r\n\r\nSi vous n'êtes pas à l'origine de cette demande, ignorez cet email.",
		link)
	return s.mailer.Send(email, "Réinitialisation de votre mot de passe BarterSwap", body)
}

// ResetPassword consomme un token de réinitialisation à usage unique, change
// le mot de passe et révoque toutes les sessions du compte.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID int64
	err = tx.QueryRowContext(ctx,
		`UPDATE password_reset_tokens SET used_at = now()
		 WHERE token_hash = $1 AND expires_at > now() AND used_at IS NULL
		 RETURNING user_id`, hashToken(token)).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidToken
	}
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, password_changed_at = now(),
		 failed_login_attempts = 0, locked_until = NULL WHERE id = $2`, hash, userID)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", ErrInvalidEmail
	}
	return email, nil
}

// fakeHash sert uniquement à égaliser le temps de réponse quand l'email est
// inconnu (hash de "dummy-password").
var fakeHash = func() string {
	h, _ := HashPassword("dummy-password")
	return h
}()
