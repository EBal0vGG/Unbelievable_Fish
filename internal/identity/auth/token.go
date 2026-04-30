package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type TokenProvider struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

type Claims struct {
	UserID    string        `json:"sub"`
	CompanyID string        `json:"company_id"`
	Role      identity.Role `json:"role"`
	ExpiresAt int64         `json:"exp"`
}

type tokenHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

func NewTokenProvider(secret string, ttl time.Duration) *TokenProvider {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &TokenProvider{
		secret: []byte(secret),
		ttl:    ttl,
		now:    time.Now,
	}
}

func (p *TokenProvider) Generate(user *identity.User) (string, error) {
	headerPart, err := p.encode(tokenHeader{
		Algorithm: "HS256",
		Type:      "JWT",
	})
	if err != nil {
		return "", err
	}

	claimsPart, err := p.encode(Claims{
		UserID:    user.ID(),
		CompanyID: user.CompanyID(),
		Role:      user.Role(),
		ExpiresAt: p.now().Add(p.ttl).Unix(),
	})
	if err != nil {
		return "", err
	}

	signingInput := headerPart + "." + claimsPart
	signature, err := p.sign(signingInput)
	if err != nil {
		return "", err
	}

	return signingInput + "." + signature, nil
}

func (p *TokenProvider) Validate(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	var header tokenHeader
	if err := p.decode(parts[0], &header); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if header.Algorithm != "HS256" || header.Type != "JWT" {
		return Claims{}, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature, err := p.sign(signingInput)
	if err != nil {
		return Claims{}, err
	}
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := p.decode(parts[1], &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.UserID == "" || !identity.IsValidRole(claims.Role) {
		return Claims{}, ErrInvalidToken
	}
	if claims.ExpiresAt <= p.now().Unix() {
		return Claims{}, ErrExpiredToken
	}

	return claims, nil
}

func (p *TokenProvider) encode(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func (p *TokenProvider) decode(part string, dst any) error {
	payload, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, dst)
}

func (p *TokenProvider) sign(value string) (string, error) {
	mac := hmac.New(sha256.New, p.secret)
	if _, err := mac.Write([]byte(value)); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func ParseBearerToken(header string) (string, error) {
	if header == "" {
		return "", ErrMissingAuthorizationHeader
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" || strings.TrimSpace(parts[1]) == "" {
		return "", ErrInvalidAuthorizationHeader
	}
	return strings.TrimSpace(parts[1]), nil
}

func IdentityFromClaims(claims Claims) (Identity, error) {
	if claims.UserID == "" || !identity.IsValidRole(claims.Role) {
		return Identity{}, ErrInvalidToken
	}
	return Identity{
		UserID:    claims.UserID,
		CompanyID: claims.CompanyID,
		Role:      claims.Role,
	}, nil
}

func ValidateBearerToken(provider *TokenProvider, header string) (Identity, error) {
	if provider == nil {
		return Identity{}, errors.New("token provider is nil")
	}
	token, err := ParseBearerToken(header)
	if err != nil {
		return Identity{}, err
	}
	claims, err := provider.Validate(token)
	if err != nil {
		return Identity{}, err
	}
	return IdentityFromClaims(claims)
}
