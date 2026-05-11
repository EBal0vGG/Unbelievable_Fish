package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

const (
	defaultVerificationTTL      = 24 * time.Hour
	defaultVerificationCooldown = 5 * time.Minute
)

type VerifyEmailCommand struct {
	Token string
}

type ResendVerificationCommand struct {
	Login string
}

type VerifyEmailResult struct {
	AlreadyVerified bool
}

type ResendVerificationResult struct {
	AlreadyVerified bool
}

type VerificationTokenGenerator interface {
	NewToken() (string, error)
	HashToken(token string) string
	NewTokenID() string
}

type secureVerificationTokenGenerator struct{}

func NewSecureVerificationTokenGenerator() VerificationTokenGenerator {
	return secureVerificationTokenGenerator{}
}

func (secureVerificationTokenGenerator) NewToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func (secureVerificationTokenGenerator) HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (secureVerificationTokenGenerator) NewTokenID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return "email-token-" + hex.EncodeToString(b[:])
}

type EmailVerificationService struct {
	tokens    EmailVerificationTokenRepository
	sender    VerificationEmailSender
	generator VerificationTokenGenerator
	publicURL string
	ttl       time.Duration
	cooldown  time.Duration
	clock     Clock
}

func NewEmailVerificationService(
	tokens EmailVerificationTokenRepository,
	sender VerificationEmailSender,
	generator VerificationTokenGenerator,
	publicURL string,
	ttl time.Duration,
	cooldown time.Duration,
	clock Clock,
) (*EmailVerificationService, error) {
	if tokens == nil {
		return nil, ErrNilVerificationTokens
	}
	if sender == nil {
		return nil, ErrNilVerificationEmailSender
	}
	if generator == nil {
		generator = NewSecureVerificationTokenGenerator()
	}
	if strings.TrimSpace(publicURL) == "" {
		publicURL = "http://localhost:3000"
	}
	if ttl <= 0 {
		ttl = defaultVerificationTTL
	}
	if cooldown <= 0 {
		cooldown = defaultVerificationCooldown
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &EmailVerificationService{
		tokens:    tokens,
		sender:    sender,
		generator: generator,
		publicURL: strings.TrimRight(strings.TrimSpace(publicURL), "/"),
		ttl:       ttl,
		cooldown:  cooldown,
		clock:     clock,
	}, nil
}

func (s *EmailVerificationService) Send(ctx context.Context, user *identity.User) error {
	email, err := s.CreateToken(ctx, user)
	if err != nil {
		return err
	}
	return s.SendEmail(ctx, user.ID(), email)
}

func (s *EmailVerificationService) CreateToken(ctx context.Context, user *identity.User) (VerificationEmail, error) {
	if user == nil {
		return VerificationEmail{}, ErrUserNotFound
	}
	if user.EmailVerified() {
		return VerificationEmail{}, nil
	}
	rawToken, err := s.generator.NewToken()
	if err != nil {
		return VerificationEmail{}, err
	}
	now := s.clock.Now().UTC()
	token := EmailVerificationToken{
		ID:        s.generator.NewTokenID(),
		UserID:    user.ID(),
		TokenHash: s.generator.HashToken(rawToken),
		ExpiresAt: now.Add(s.ttl),
		CreatedAt: now,
		SentAt:    now,
	}
	if err := s.tokens.RevokeActiveForUser(ctx, user.ID(), now); err != nil {
		return VerificationEmail{}, err
	}
	if err := s.tokens.Save(ctx, token); err != nil {
		return VerificationEmail{}, err
	}
	return VerificationEmail{
		To:               user.Login(),
		VerificationLink: s.verificationLink(rawToken),
		ExpiresAt:        token.ExpiresAt,
	}, nil
}

func (s *EmailVerificationService) SendEmail(ctx context.Context, userID string, email VerificationEmail) error {
	if email.To == "" {
		return nil
	}
	if err := s.sender.SendVerificationEmail(ctx, email); err != nil {
		if revokeErr := s.tokens.RevokeActiveForUser(ctx, userID, s.clock.Now().UTC()); revokeErr != nil {
			slog.ErrorContext(ctx, "email_verification_revoke_after_send_failed", "component", "identity.email_verification", "user_id", userID, "error", revokeErr)
		}
		slog.ErrorContext(ctx, "email_verification_send_failed", "component", "identity.email_verification", "user_id", userID, "error", err)
		return ErrVerificationEmailSend
	}
	slog.InfoContext(ctx, "email_verification_sent", "component", "identity.email_verification", "user_id", userID, "expires_at", email.ExpiresAt)
	return nil
}

func (s *EmailVerificationService) CheckCooldown(ctx context.Context, userID string) error {
	lastSentAt, ok, err := s.tokens.LastSentAtForUser(ctx, userID)
	if err != nil {
		return err
	}
	if ok && s.clock.Now().UTC().Sub(lastSentAt) < s.cooldown {
		return ErrVerificationCooldown
	}
	return nil
}

func (s *EmailVerificationService) verificationLink(rawToken string) string {
	link, err := url.Parse(s.publicURL + "/verify-email")
	if err != nil {
		return s.publicURL + "/verify-email?token=" + url.QueryEscape(rawToken)
	}
	query := link.Query()
	query.Set("token", rawToken)
	link.RawQuery = query.Encode()
	return link.String()
}

type VerifyEmail struct {
	users     UserRepository
	tokens    EmailVerificationTokenRepository
	generator VerificationTokenGenerator
	clock     Clock
	uow       UnitOfWork
}

func NewVerifyEmail(
	users UserRepository,
	tokens EmailVerificationTokenRepository,
	generator VerificationTokenGenerator,
	clock Clock,
	uow UnitOfWork,
) (*VerifyEmail, error) {
	if users == nil {
		return nil, ErrNilUserRepository
	}
	if tokens == nil {
		return nil, ErrNilVerificationTokens
	}
	if generator == nil {
		generator = NewSecureVerificationTokenGenerator()
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &VerifyEmail{users: users, tokens: tokens, generator: generator, clock: clock, uow: uow}, nil
}

func (uc *VerifyEmail) Execute(ctx context.Context, cmd VerifyEmailCommand) (VerifyEmailResult, error) {
	rawToken := strings.TrimSpace(cmd.Token)
	if rawToken == "" {
		return VerifyEmailResult{}, ErrVerificationTokenRequired
	}
	token, err := uc.tokens.GetByHash(ctx, uc.generator.HashToken(rawToken))
	if err != nil {
		if errors.Is(err, ErrVerificationTokenInvalid) {
			return VerifyEmailResult{}, ErrVerificationTokenInvalid
		}
		return VerifyEmailResult{}, err
	}
	now := uc.clock.Now().UTC()
	if token.UsedAt != nil {
		return VerifyEmailResult{}, ErrVerificationTokenUsed
	}
	if token.RevokedAt != nil {
		return VerifyEmailResult{}, ErrVerificationTokenInvalid
	}
	if !token.ExpiresAt.After(now) {
		return VerifyEmailResult{}, ErrVerificationTokenExpired
	}

	var result VerifyEmailResult
	run := func(txCtx context.Context) error {
		user, err := uc.users.GetByID(txCtx, token.UserID)
		if err != nil {
			return err
		}
		if user.EmailVerified() {
			result.AlreadyVerified = true
		} else {
			user.VerifyEmail()
			if err := uc.users.Save(txCtx, user); err != nil {
				return err
			}
		}
		return uc.tokens.MarkUsed(txCtx, token.ID, now)
	}
	if uc.uow != nil {
		if err := uc.uow.WithinTx(ctx, run); err != nil {
			return VerifyEmailResult{}, err
		}
	} else if err := run(ctx); err != nil {
		return VerifyEmailResult{}, err
	}
	slog.InfoContext(ctx, "email_verified", "component", "identity.email_verification", "user_id", token.UserID)
	return result, nil
}

type ResendVerification struct {
	users   UserRepository
	service *EmailVerificationService
}

func NewResendVerification(users UserRepository, service *EmailVerificationService) (*ResendVerification, error) {
	if users == nil {
		return nil, ErrNilUserRepository
	}
	if service == nil {
		return nil, ErrNilVerificationTokens
	}
	return &ResendVerification{users: users, service: service}, nil
}

func (uc *ResendVerification) Execute(ctx context.Context, cmd ResendVerificationCommand) (ResendVerificationResult, error) {
	login := strings.ToLower(strings.TrimSpace(cmd.Login))
	if login == "" {
		return ResendVerificationResult{}, nil
	}
	user, err := uc.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ResendVerificationResult{}, nil
		}
		return ResendVerificationResult{}, err
	}
	if user.EmailVerified() {
		return ResendVerificationResult{AlreadyVerified: true}, nil
	}
	if err := uc.service.CheckCooldown(ctx, user.ID()); err != nil {
		return ResendVerificationResult{}, err
	}
	if err := uc.service.Send(ctx, user); err != nil {
		return ResendVerificationResult{}, err
	}
	return ResendVerificationResult{}, nil
}
