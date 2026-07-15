package auth

import "testing"

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		name     string
		password string
		valid    bool
	}{
		{"valide", "Correct-Horse-42!", true},
		{"trop court", "Abc-123!", false},
		{"sans chiffre", "MotDePasse-Fort!", false},
		{"sans symbole", "MotDePasse1234", false},
		{"sans lettre", "1234-5678-9012!", false},
		{"vide", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidatePassword(c.password)
			if c.valid && err != nil {
				t.Errorf("attendu valide, reçu %v", err)
			}
			if !c.valid && err == nil {
				t.Error("attendu une erreur, reçu nil")
			}
		})
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	const password = "Correct-Horse-42!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("le bon mot de passe doit être accepté")
	}

	ok, err = VerifyPassword("mauvais-mot-de-passe-1!", hash)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("un mauvais mot de passe doit être refusé")
	}
}

func TestHashPasswordSaltsAreUnique(t *testing.T) {
	h1, err := HashPassword("Correct-Horse-42!")
	if err != nil {
		t.Fatal(err)
	}
	h2, err := HashPassword("Correct-Horse-42!")
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Error("deux hashs du même mot de passe doivent différer (sel aléatoire)")
	}
}
