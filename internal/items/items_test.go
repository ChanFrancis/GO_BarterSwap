package items

import "testing"

func TestInputValidate(t *testing.T) {
	valid := Input{Title: "Vélo de ville", Category: "Sport", Condition: "Bon"}

	cases := []struct {
		name   string
		mutate func(*Input)
		valid  bool
	}{
		{"valide", func(in *Input) {}, true},
		{"titre trop court", func(in *Input) { in.Title = "ab" }, false},
		{"titre avec espaces seulement", func(in *Input) { in.Title = "   " }, false},
		{"catégorie vide", func(in *Input) { in.Category = " " }, false},
		{"état inconnu", func(in *Input) { in.Condition = "cassé" }, false},
		{"état en majuscules accepté", func(in *Input) { in.Condition = "NEUF" }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := valid
			c.mutate(&in)
			err := in.Validate()
			if c.valid && err != nil {
				t.Errorf("attendu valide, reçu %v", err)
			}
			if !c.valid && err == nil {
				t.Error("attendu une erreur, reçu nil")
			}
		})
	}
}

func TestValidateNormalizes(t *testing.T) {
	in := Input{Title: "  Vélo  ", Category: " Sport ", Condition: " Neuf "}
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	if in.Title != "Vélo" || in.Category != "sport" || in.Condition != "neuf" {
		t.Errorf("normalisation incorrecte : %+v", in)
	}
}
