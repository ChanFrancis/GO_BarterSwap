package items

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Statuts d'un objet.
const (
	StatusAvailable = "disponible"
	StatusTraded    = "troqué" // réservé par un troc en cours d'acceptation
)

// Conditions acceptées pour un objet.
var validConditions = []string{"neuf", "très bon", "bon", "usé"}

var (
	ErrNotFound  = errors.New("objet introuvable")
	ErrForbidden = errors.New("vous n'êtes pas le propriétaire de cet objet")
)

// ValidationError signale une entrée utilisateur invalide.
type ValidationError struct{ msg string }

func (e ValidationError) Error() string { return e.msg }

type Item struct {
	ID          int64     `json:"id"`
	OwnerID     int64     `json:"owner_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Condition   string    `json:"condition"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Input porte les champs modifiables par l'utilisateur.
type Input struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Condition   string `json:"condition"`
}

// Validate normalise et vérifie les champs d'un objet.
func (in *Input) Validate() error {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Category = strings.ToLower(strings.TrimSpace(in.Category))
	in.Condition = strings.ToLower(strings.TrimSpace(in.Condition))

	if len([]rune(in.Title)) < 3 || len([]rune(in.Title)) > 120 {
		return ValidationError{"le titre doit faire entre 3 et 120 caractères"}
	}
	if len([]rune(in.Description)) > 2000 {
		return ValidationError{"la description ne doit pas dépasser 2000 caractères"}
	}
	if in.Category == "" {
		return ValidationError{"la catégorie est obligatoire"}
	}
	for _, c := range validConditions {
		if in.Condition == c {
			return nil
		}
	}
	return ValidationError{fmt.Sprintf("l'état doit être l'un de : %s", strings.Join(validConditions, ", "))}
}
