// Package passwordhash hashes and verifies passwords with Argon2id.
package passwordhash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var ErrMalformedHash = errors.New("passwordhash: malformed hash")

// Params are the Argon2id cost settings. Memory is in KiB.
type Params struct {
	Memory      uint32
	Time        uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams: 64 MiB, 3 passes, 2 threads.
var DefaultParams = Params{
	Memory:      64 * 1024,
	Time:        3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

type Argon2id struct {
	params Params
}

func NewArgon2id(params Params) *Argon2id {
	return &Argon2id{params: params}
}

// Hash returns a self-describing string:
// $argon2id$v=19$m=65536,t=3,p=2$<salt>$<key>
func (a *Argon2id) Hash(password string) (string, error) {
	salt := make([]byte, a.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("passwordhash: generate salt: %w", err)
	}

	p := a.params
	key := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Parallelism, p.KeyLength)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Time, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify reports whether password matches encoded. It reads the cost
// settings from encoded, so old hashes keep working after we raise them.
func (a *Argon2id) Verify(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrMalformedHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrMalformedHash
	}

	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &p.Parallelism); err != nil {
		return false, ErrMalformedHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrMalformedHash
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrMalformedHash
	}

	got := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Parallelism, uint32(len(want)))

	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
