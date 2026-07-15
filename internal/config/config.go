package config

import "os"

// Config regroupe la configuration de l'application, chargée
// depuis les variables d'environnement (voir .env.example).
type Config struct {
	Port         string
	DatabaseURL  string
	SMTPAddr     string // hôte:port du serveur SMTP (Mailpit en dev)
	EmailFrom    string
	AppURL       string // URL publique, utilisée dans les liens d'emails
	SecureCookie bool   // true en production (HTTPS obligatoire)
}

func Load() Config {
	return Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", ""),
		SMTPAddr:     getEnv("SMTP_ADDR", "localhost:1025"),
		EmailFrom:    getEnv("EMAIL_FROM", "no-reply@barterswap.local"),
		AppURL:       getEnv("APP_URL", "http://localhost:8080"),
		SecureCookie: getEnv("SECURE_COOKIE", "false") == "true",
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
