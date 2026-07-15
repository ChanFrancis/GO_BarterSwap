package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// newToken génère un secret aléatoire à remettre au client, et son empreinte
// SHA-256 à stocker en base : un vol de la base ne permet pas de rejouer les
// sessions ni les liens de réinitialisation.
func newToken() (token, tokenHash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
