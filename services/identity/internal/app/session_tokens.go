package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sholana1/fintechdemo/identity/internal/domain"
)

const refreshTokenBytes = 32

// TokenPair is what a client receives after signing in or refreshing.
type TokenPair struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	SessionID             string
	UserID                string
}

// SessionTokens issues an access token and a fresh refresh token for a
// session. Login and RefreshSession both use it, so the rules for
// minting tokens live in one place.
type SessionTokens struct {
	refreshTokens   RefreshTokenRepository
	accessTokens    AccessTokenIssuer
	refreshTokenTTL time.Duration
}

func NewSessionTokens(refreshTokens RefreshTokenRepository, accessTokens AccessTokenIssuer, refreshTokenTTL time.Duration) *SessionTokens {
	return &SessionTokens{
		refreshTokens:   refreshTokens,
		accessTokens:    accessTokens,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (s *SessionTokens) Issue(ctx context.Context, session domain.Session, now time.Time) (TokenPair, error) {
	plaintext, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate refresh token id: %w", err)
	}

	// A refresh token never outlives its session.
	expiresAt := now.Add(s.refreshTokenTTL)
	if expiresAt.After(session.ExpiresAt) {
		expiresAt = session.ExpiresAt
	}

	record := domain.RefreshToken{
		ID:        id.String(),
		SessionID: session.ID,
		TokenHash: HashRefreshToken(plaintext),
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	if err := s.refreshTokens.Create(ctx, record); err != nil {
		return TokenPair{}, err
	}

	accessToken, accessExpiresAt, err := s.accessTokens.Issue(session.UserID, session.ID, now)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue access token: %w", err)
	}

	return TokenPair{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshToken:          plaintext,
		RefreshTokenExpiresAt: expiresAt,
		SessionID:             session.ID,
		UserID:                session.UserID,
	}, nil
}

func newRefreshToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashRefreshToken is how a refresh token is stored and looked up.
// Plain SHA-256 is right here, unlike for passwords: the token is 256
// random bits, so there is nothing to guess and no need to be slow.
func HashRefreshToken(plaintext string) []byte {
	sum := sha256.Sum256([]byte(plaintext))
	return sum[:]
}
