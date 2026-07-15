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
	ErrIntrouvable = errors.New("ressource introuvable")
	ErrInterdit    = errors.New("action réservée au propriétaire de la ressource")
)

// ValidationError signale une entrée utilisateur invalide (HTTP 400).
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

// Niveaux de compétence acceptés.
var niveauxValides = []string{"débutant", "intermédiaire", "expert"}

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
