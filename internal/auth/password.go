package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

// ErrWeakPassword est renvoyée quand le mot de passe ne respecte pas la
// politique CNIL : 12 caractères minimum, avec lettres, chiffres et symboles.
var ErrWeakPassword = errors.New(
	"le mot de passe doit contenir au moins 12 caractères, dont des lettres, des chiffres et des symboles")

// ValidatePassword vérifie la politique de mot de passe fort de la CNIL.
func ValidatePassword(password string) error {
	var hasLetter, hasDigit, hasSymbol bool
	length := 0
	for _, r := range password {
		length++
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSymbol = true
		}
	}
	if length < 12 || !hasLetter || !hasDigit || !hasSymbol {
		return ErrWeakPassword
	}
	return nil
}

// Paramètres argon2id recommandés (OWASP) : 64 Mo de mémoire, 1 itération,
// 4 threads.
const (
	argonMemory  = 64 * 1024
	argonTime    = 1
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// HashPassword dérive le mot de passe avec argon2id et retourne une chaîne
// autoportante au format standard $argon2id$...
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword compare un mot de passe avec son hash en temps constant.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("format de hash invalide")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, err
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	key := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expected)))
	return subtle.ConstantTimeCompare(key, expected) == 1, nil
}
