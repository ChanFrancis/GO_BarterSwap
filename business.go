package main

import (
	"errors"
	"fmt"
	"strings"
)

// Règles métier pures : aucune dépendance à HTTP ni à la base de données,
// pour être testables unitairement (tests table-driven).

// CreditsBienvenue est attribué à la création d'un compte.
const CreditsBienvenue = 10

// Erreurs sentinelles, traduites en codes HTTP par respondError.
var (
	ErrIntrouvable         = errors.New("ressource introuvable")
	ErrInterdit            = errors.New("action réservée au propriétaire de la ressource")
	ErrCompetenceManquante = errors.New("vous ne possédez pas de compétence correspondant à cette catégorie")
)

// ValidationError signale une entrée utilisateur invalide (HTTP 400).
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

// Niveaux de compétence acceptés.
var niveauxValides = []string{"débutant", "intermédiaire", "expert"}

// Catégories de service acceptées (liste fermée du sujet).
var categoriesValides = []string{
	"Informatique", "Jardinage", "Bricolage", "Cuisine", "Musique",
	"Langues", "Sport", "Tutorat", "Déménagement", "Photographie",
	"Animalier", "Couture", "Autre",
}

// serviceInput porte les champs modifiables d'une annonce de service.
type serviceInput struct {
	Titre        string `json:"titre"`
	Description  string `json:"description"`
	Categorie    string `json:"categorie"`
	DureeMinutes int    `json:"duree_minutes"`
	Credits      int    `json:"credits"`
	Ville        string `json:"ville"`
}

// validerService vérifie les champs d'une annonce (hors contrôle de
// compétence, qui nécessite la base).
func validerService(in serviceInput) error {
	if strings.TrimSpace(in.Titre) == "" {
		return ValidationError{"le titre est obligatoire"}
	}
	if len([]rune(in.Titre)) > 120 {
		return ValidationError{"le titre ne doit pas dépasser 120 caractères"}
	}
	if !contient(categoriesValides, in.Categorie) {
		return ValidationError{fmt.Sprintf(
			"catégorie %q invalide (attendu : %s)", in.Categorie, strings.Join(categoriesValides, ", "))}
	}
	if in.DureeMinutes <= 0 {
		return ValidationError{"la durée doit être supérieure à zéro"}
	}
	if in.Credits <= 0 {
		return ValidationError{"le coût en crédits doit être supérieur à zéro"}
	}
	return nil
}

// validerPseudo vérifie le pseudo d'un utilisateur.
func validerPseudo(pseudo string) error {
	if strings.TrimSpace(pseudo) == "" {
		return ValidationError{"le pseudo est obligatoire"}
	}
	if len([]rune(pseudo)) > 50 {
		return ValidationError{"le pseudo ne doit pas dépasser 50 caractères"}
	}
	return nil
}

// validerSkills vérifie une liste de compétences (nom non vide, niveau dans
// la liste fermée).
func validerSkills(skills []Skill) error {
	for _, s := range skills {
		if strings.TrimSpace(s.Nom) == "" {
			return ValidationError{"le nom d'une compétence est obligatoire"}
		}
		if !contient(niveauxValides, s.Niveau) {
			return ValidationError{fmt.Sprintf(
				"niveau %q invalide (attendu : %s)", s.Niveau, strings.Join(niveauxValides, ", "))}
		}
	}
	return nil
}

func contient(liste []string, valeur string) bool {
	for _, v := range liste {
		if v == valeur {
			return true
		}
	}
	return false
}
