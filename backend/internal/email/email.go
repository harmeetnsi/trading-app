package email

import (
	"log"
	"gopkg.in/gomail.v2"
)

type EmailService struct {
	dialer *gomail.Dialer
	sender string
}

func NewEmailService(host string, port int, username, password, sender string) *EmailService {
	dialer := gomail.NewDialer(host, port, username, password)
	return &EmailService{
		dialer: dialer,
		sender: sender,
	}
}

func (s *EmailService) SendEmail(recipient, subject, body string) error {
	if s.dialer == nil {
		log.Printf("Email service not configured. Skipping email to %s with subject: %s", recipient, subject)
		return nil // Don't treat as a hard error
	}

	log.Printf("Attempting to send email to %s with subject: %s", recipient, subject)

	m := gomail.NewMessage()
	m.SetHeader("From", s.sender)
	m.SetHeader("To", recipient)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		log.Printf("ERROR: Failed to send email to %s. Subject: '%s'. Error: %v", recipient, subject, err)
		return err
	}

	log.Printf("Email sent successfully to %s", recipient)
	return nil
}
