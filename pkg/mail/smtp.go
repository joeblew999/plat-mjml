// Package mail provides email sending and validation utilities.
package mail

import (
	"fmt"
	"net/smtp"
)

// Config holds configuration for sending emails via SMTP.
type Config struct {
	SMTPHost  string
	SMTPPort  string
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

// Send sends an HTML email.
func Send(config Config, toEmail, subject, htmlBody string) error {
	message := fmt.Sprintf(
		"From: %s <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		config.FromName, config.FromEmail,
		toEmail,
		subject,
		htmlBody,
	)

	auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)

	return smtp.SendMail(
		config.SMTPHost+":"+config.SMTPPort,
		auth,
		config.FromEmail,
		[]string{toEmail},
		[]byte(message),
	)
}

