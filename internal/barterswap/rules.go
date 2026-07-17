package barterswap

import (
	"fmt"
	"strings"
)

// CreditsBienvenue est attribué à la création d'un compte.
const CreditsBienvenue = 10

// Statuts d'un échange et cycle de vie :
//
//	pending → accepted → completed
//	   ↓          ↓
//	rejected   cancelled
const (
	StatusPending   = "pending"
	StatusAccepted  = "accepted"
	StatusRejected  = "rejected"
	StatusCancelled = "cancelled"
	StatusCompleted = "completed"
)

var validStatuses = []string{
	StatusPending, StatusAccepted, StatusRejected, StatusCancelled, StatusCompleted,
}

// Niveaux de compétence acceptés.
var validLevels = []string{"débutant", "intermédiaire", "expert"}

// Catégories de service acceptées (liste fermée du sujet).
var validCategories = []string{
	"Informatique", "Jardinage", "Bricolage", "Cuisine", "Musique",
	"Langues", "Sport", "Tutorat", "Déménagement", "Photographie",
	"Animalier", "Couture", "Autre",
}

// ServiceInput porte les champs modifiables d'une annonce de service.
type ServiceInput struct {
	Titre        string `json:"titre"`
	Description  string `json:"description"`
	Categorie    string `json:"categorie"`
	DureeMinutes int    `json:"duree_minutes"`
	Credits      int    `json:"credits"`
	Ville        string `json:"ville"`
}

// ValidatePseudo vérifie le pseudo d'un utilisateur.
func ValidatePseudo(pseudo string) error {
	if strings.TrimSpace(pseudo) == "" {
		return ValidationError{"le pseudo est obligatoire"}
	}
	if len([]rune(pseudo)) > 50 {
		return ValidationError{"le pseudo ne doit pas dépasser 50 caractères"}
	}
	return nil
}

// ValidateSkills vérifie une liste de compétences (nom non vide, niveau dans
// la liste fermée).
func ValidateSkills(skills []Skill) error {
	for _, s := range skills {
		if strings.TrimSpace(s.Nom) == "" {
			return ValidationError{"le nom d'une compétence est obligatoire"}
		}
		if !contains(validLevels, s.Niveau) {
			return ValidationError{fmt.Sprintf(
				"niveau %q invalide (attendu : %s)", s.Niveau, strings.Join(validLevels, ", "))}
		}
	}
	return nil
}

// ValidateService vérifie les champs d'une annonce (hors contrôle de
// compétence, qui nécessite la base).
func ValidateService(in ServiceInput) error {
	if strings.TrimSpace(in.Titre) == "" {
		return ValidationError{"le titre est obligatoire"}
	}
	if len([]rune(in.Titre)) > 120 {
		return ValidationError{"le titre ne doit pas dépasser 120 caractères"}
	}
	if !contains(validCategories, in.Categorie) {
		return ValidationError{fmt.Sprintf(
			"catégorie %q invalide (attendu : %s)", in.Categorie, strings.Join(validCategories, ", "))}
	}
	if in.DureeMinutes <= 0 {
		return ValidationError{"la durée doit être supérieure à zéro"}
	}
	if in.Credits <= 0 {
		return ValidationError{"le coût en crédits doit être supérieur à zéro"}
	}
	return nil
}

// ValidateNote vérifie qu'une note d'évaluation est comprise entre 1 et 5.
func ValidateNote(note int) error {
	if note < 1 || note > 5 {
		return ValidationError{"la note doit être comprise entre 1 et 5"}
	}
	return nil
}

// ValidStatus indique si une valeur de filtre ?status= est reconnue.
func ValidStatus(s string) bool {
	return contains(validStatuses, s)
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
