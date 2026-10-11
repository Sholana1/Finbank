// Package accesstoken signs short-lived access tokens (JWTs).
package accesstoken

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	issuerName = "fintechdemo-identity"
	audience   = "fintechdemo-api"
)

// claims is the token's payload. Sub is the user, sid is the session.
type claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

type Issuer struct {
	privateKey ed25519.PrivateKey
	keyID      string
	ttl        time.Duration
}

// NewIssuer takes the private key as a base64-encoded 32-byte Ed25519 seed.
func NewIssuer(seedBase64 string, ttl time.Duration) (*Issuer, error) {
	seed, err := base64.StdEncoding.DecodeString(seedBase64)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, errors.New("accesstoken: private key must be a base64-encoded 32-byte Ed25519 seed")
	}

	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	// The key id is derived from the public key, so a verifier holding
	// several keys during a rotation can tell which one signed a token.
	fingerprint := sha256.Sum256(publicKey)

	return &Issuer{
		privateKey: privateKey,
		keyID:      hex.EncodeToString(fingerprint[:8]),
		ttl:        ttl,
	}, nil
}

func (i *Issuer) Issue(userID, sessionID string, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(i.ttl)

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    issuerName,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})
	token.Header["kid"] = i.keyID

	signed, err := token.SignedString(i.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("accesstoken: sign: %w", err)
	}

	return signed, expiresAt, nil
}

// KeyID identifies the signing key. It is safe to log.
func (i *Issuer) KeyID() string {
	return i.keyID
}
