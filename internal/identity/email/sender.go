package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
)

type DevVerificationSender struct {
	logger *slog.Logger
}

func NewDevVerificationSender(logger *slog.Logger) DevVerificationSender {
	if logger == nil {
		logger = slog.Default()
	}
	return DevVerificationSender{logger: logger}
}

func (s DevVerificationSender) SendVerificationEmail(ctx context.Context, email identityapp.VerificationEmail) error {
	s.logger.InfoContext(
		ctx,
		"DEV EMAIL VERIFICATION LINK",
		"component", "identity.email.sender",
		"to", email.To,
		"verification_link", email.VerificationLink,
		"expires_at", email.ExpiresAt,
	)
	return nil
}

type SMTPVerificationSender struct {
	host         string
	port         string
	username     string
	password     string
	envelopeFrom string
	headerFrom   string
	useTLS       bool
	secure       bool
	timeout      time.Duration
	logger       *slog.Logger
}

func NewSenderFromEnv(logger *slog.Logger) identityapp.VerificationEmailSender {
	if logger == nil {
		logger = slog.Default()
	}
	host := strings.TrimSpace(dbconfig.EnvOrDefault("SMTP_HOST", ""))
	from := strings.TrimSpace(dbconfig.EnvOrDefault("SMTP_FROM", ""))
	if host == "" || from == "" {
		if isProductionEnv() {
			logger.Error("email_verification_sender_not_configured", "component", "identity.email.sender", "required", "SMTP_HOST,SMTP_FROM")
			return disabledVerificationSender{logger: logger}
		}
		logger.Info("email_verification_dev_sender_enabled", "component", "identity.email.sender", "reason", "SMTP_HOST or SMTP_FROM is empty")
		return NewDevVerificationSender(logger)
	}
	fromAddress, err := mail.ParseAddress(from)
	if err != nil {
		logger.Error("email_verification_sender_invalid_from", "component", "identity.email.sender", "error", err)
		return disabledVerificationSender{logger: logger}
	}
	return SMTPVerificationSender{
		host:         host,
		port:         dbconfig.EnvOrDefault("SMTP_PORT", "587"),
		username:     dbconfig.EnvOrDefault("SMTP_USER", ""),
		password:     dbconfig.EnvOrDefault("SMTP_PASSWORD", ""),
		envelopeFrom: fromAddress.Address,
		headerFrom:   fromAddress.String(),
		useTLS:       dbconfig.EnvBool("SMTP_USE_TLS", true),
		secure:       dbconfig.EnvBool("SMTP_SECURE", false),
		timeout:      envDurationSeconds("SMTP_TIMEOUT_SECONDS", 10),
		logger:       logger,
	}
}

type disabledVerificationSender struct {
	logger *slog.Logger
}

func (s disabledVerificationSender) SendVerificationEmail(ctx context.Context, email identityapp.VerificationEmail) error {
	s.logger.ErrorContext(ctx, "email_verification_sender_disabled", "component", "identity.email.sender", "to", email.To)
	return errors.New("verification email sender is not configured")
}

func isProductionEnv() bool {
	for _, key := range []string{"APP_ENV", "ENV", "GO_ENV"} {
		if strings.EqualFold(strings.TrimSpace(os.Getenv(key)), "production") {
			return true
		}
	}
	return false
}

func (s SMTPVerificationSender) SendVerificationEmail(ctx context.Context, email identityapp.VerificationEmail) error {
	addr := net.JoinHostPort(s.host, s.port)
	sendCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	session, err := s.smtpClient(sendCtx, addr)
	if err != nil {
		return err
	}
	defer session.close()

	if s.username != "" || s.password != "" {
		if err := session.withDeadline(sendCtx); err != nil {
			return err
		}
		if err := session.client.Auth(smtp.PlainAuth("", s.username, s.password, s.host)); err != nil {
			return err
		}
	}
	if err := session.withDeadline(sendCtx); err != nil {
		return err
	}
	if err := session.client.Mail(s.envelopeFrom); err != nil {
		return err
	}
	if err := session.withDeadline(sendCtx); err != nil {
		return err
	}
	if err := session.client.Rcpt(email.To); err != nil {
		return err
	}
	if err := session.withDeadline(sendCtx); err != nil {
		return err
	}
	writer, err := session.client.Data()
	if err != nil {
		return err
	}
	if err := session.withDeadline(sendCtx); err != nil {
		_ = writer.Close()
		return err
	}
	if _, err := writer.Write([]byte(buildPlainTextMessage(s.headerFrom, email))); err != nil {
		_ = writer.Close()
		return err
	}
	if err := session.withDeadline(sendCtx); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if err := session.withDeadline(sendCtx); err != nil {
		return err
	}
	if err := session.client.Quit(); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "email_verification_smtp_sent", "component", "identity.email.sender", "to", email.To, "host", s.host, "port", s.port, "secure", s.secure, "starttls", s.useTLS && !s.secure)
	return nil
}

type smtpSession struct {
	client *smtp.Client
	conn   net.Conn
}

func (s smtpSession) close() {
	if s.client != nil {
		_ = s.client.Close()
		return
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
}

func (s smtpSession) withDeadline(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(10 * time.Second)
	}
	return s.conn.SetDeadline(deadline)
}

func (s SMTPVerificationSender) smtpClient(ctx context.Context, addr string) (smtpSession, error) {
	dialer := &net.Dialer{Timeout: s.timeout}
	if deadline, ok := ctx.Deadline(); ok {
		dialer.Deadline = deadline
	}
	if s.secure {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return smtpSession{}, err
		}
		session := smtpSession{conn: conn}
		if err := session.withDeadline(ctx); err != nil {
			_ = conn.Close()
			return smtpSession{}, err
		}
		client, err := smtp.NewClient(conn, s.host)
		if err != nil {
			_ = conn.Close()
			return smtpSession{}, err
		}
		session.client = client
		return session, nil
	}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return smtpSession{}, err
	}
	session := smtpSession{conn: conn}
	if err := session.withDeadline(ctx); err != nil {
		_ = conn.Close()
		return smtpSession{}, err
	}
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		_ = conn.Close()
		return smtpSession{}, err
	}
	session.client = client
	if s.useTLS {
		if err := session.withDeadline(ctx); err != nil {
			session.close()
			return smtpSession{}, err
		}
		ok, _ := client.Extension("STARTTLS")
		if ok {
			if err := session.withDeadline(ctx); err != nil {
				session.close()
				return smtpSession{}, err
			}
			if err := client.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
				session.close()
				return smtpSession{}, err
			}
		} else {
			session.close()
			return smtpSession{}, errors.New("smtp server does not advertise STARTTLS")
		}
	}
	return session, nil
}

func buildPlainTextMessage(from string, email identityapp.VerificationEmail) string {
	subject := "Подтверждение регистрации — Рыбная биржа"
	body := fmt.Sprintf(
		"Здравствуйте!\r\n\r\nДля завершения регистрации подтвердите email, перейдя по ссылке:\r\n%s\r\n\r\nЕсли вы не регистрировались на Рыбной бирже, просто проигнорируйте это письмо.\r\nСсылка действует 24 часа.\r\n",
		email.VerificationLink,
	)
	return strings.Join([]string{
		"From: " + from,
		"To: " + email.To,
		"Subject: " + mimeHeader(subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")
}

func mimeHeader(value string) string {
	encoded := mime.QEncoding.Encode("UTF-8", value)
	if encoded != "" {
		return encoded
	}
	return value
}

func envDurationSeconds(key string, def int) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return time.Duration(def) * time.Second
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return time.Duration(def) * time.Second
	}
	return time.Duration(seconds) * time.Second
}
