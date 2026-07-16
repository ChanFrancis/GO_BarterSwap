package barterswap

import "testing"

func TestValidatePseudo(t *testing.T) {
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
			err := ValidatePseudo(c.pseudo)
			if c.valide && err != nil {
				t.Errorf("attendu valide, reçu %v", err)
			}
			if !c.valide && err == nil {
				t.Error("attendu une erreur, reçu nil")
			}
		})
	}
}

func TestValidateService(t *testing.T) {
	valide := ServiceInput{Titre: "Cours de piano", Categorie: "Musique", DureeMinutes: 60, Credits: 2}

	cases := []struct {
		name   string
		mutate func(*ServiceInput)
		valide bool
	}{
		{"valide", func(in *ServiceInput) {}, true},
		{"titre vide", func(in *ServiceInput) { in.Titre = "  " }, false},
		{"catégorie hors liste", func(in *ServiceInput) { in.Categorie = "Astrologie" }, false},
		{"durée nulle", func(in *ServiceInput) { in.DureeMinutes = 0 }, false},
		{"crédits négatifs", func(in *ServiceInput) { in.Credits = -1 }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := valide
			c.mutate(&in)
			err := ValidateService(in)
			if c.valide && err != nil {
				t.Errorf("attendu valide, reçu %v", err)
			}
			if !c.valide && err == nil {
				t.Error("attendu une erreur, reçu nil")
			}
		})
	}
}

func TestValidateNote(t *testing.T) {
	cases := []struct {
		note   int
		valide bool
	}{{0, false}, {1, true}, {3, true}, {5, true}, {6, false}, {-1, false}}
	for _, c := range cases {
		if err := ValidateNote(c.note); (err == nil) != c.valide {
			t.Errorf("note %d : validité attendue %v, err=%v", c.note, c.valide, err)
		}
	}
}

func TestValidStatus(t *testing.T) {
	for _, s := range []string{"pending", "accepted", "rejected", "cancelled", "completed"} {
		if !ValidStatus(s) {
			t.Errorf("%q devrait être un statut valide", s)
		}
	}
	if ValidStatus("zzz") {
		t.Error("zzz ne devrait pas être valide")
	}
}

func TestValidateSkills(t *testing.T) {
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
			err := ValidateSkills(c.skills)
			if c.valide && err != nil {
				t.Errorf("attendu valide, reçu %v", err)
			}
			if !c.valide && err == nil {
				t.Error("attendu une erreur, reçu nil")
			}
		})
	}
}
