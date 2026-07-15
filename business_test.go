package main

import "testing"

func TestValiderPseudo(t *testing.T) {
	cases := []struct {
		name   string
		pseudo string
		valide bool
	}{
		{"valide", "francis75", true},
		{"vide", "", false},
		{"espaces seulement", "   ", false},
		{"trop long", string(make([]rune, 51)), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validerPseudo(c.pseudo)
			if c.valide && err != nil {
				t.Errorf("attendu valide, reçu %v", err)
			}
			if !c.valide && err == nil {
				t.Error("attendu une erreur, reçu nil")
			}
		})
	}
}

func TestValiderSkills(t *testing.T) {
	cases := []struct {
		name   string
		skills []Skill
		valide bool
	}{
		{"liste vide", []Skill{}, true},
		{"valide", []Skill{{Nom: "Jardinage", Niveau: "expert"}}, true},
		{"nom vide", []Skill{{Nom: " ", Niveau: "expert"}}, false},
		{"niveau inconnu", []Skill{{Nom: "Cuisine", Niveau: "champion"}}, false},
		{"une invalide parmi deux", []Skill{
			{Nom: "Cuisine", Niveau: "débutant"},
			{Nom: "Piano", Niveau: "virtuose"},
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validerSkills(c.skills)
			if c.valide && err != nil {
				t.Errorf("attendu valide, reçu %v", err)
			}
			if !c.valide && err == nil {
				t.Error("attendu une erreur, reçu nil")
			}
		})
	}
}
