package mailer

import (
	"fmt"
	"net/smtp"
	"strings"
)

// SMTP envoie des emails via un serveur SMTP sans authentification :
// Mailpit en développement, un relais local en production.
type SMTP struct {
	Addr string // hôte:port, ex. "mailpit:1025"
	From string
}

func (m SMTP) Send(to, subject, body string) error {
	headers := []string{
		"From: " + m.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}
	msg := strings.Join(headers, "\r\n") + "\r\n\r\n" + body
	if err := smtp.SendMail(m.Addr, nil, m.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("envoi de l'email : %w", err)
	}
	return nil
}
