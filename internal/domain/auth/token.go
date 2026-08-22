package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid refresh token")

type RefreshToken struct {
	ID         string
	UserID     string
	Digest     string
	IssuedAt   time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ReplacedBy string
}

func DigestToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func NewRefreshToken(id, userID, raw string, issuedAt, expiresAt time.Time) (RefreshToken, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(userID) == "" || len(raw) < 32 || !issuedAt.Before(expiresAt) {
		return RefreshToken{}, ErrInvalidToken
	}
	return RefreshToken{ID: id, UserID: userID, Digest: DigestToken(raw), IssuedAt: issuedAt.UTC(), ExpiresAt: expiresAt.UTC()}, nil
}

func (t RefreshToken) Validate(raw string, now time.Time) error {
	if t.RevokedAt != nil || !now.UTC().Before(t.ExpiresAt) || DigestToken(raw) != t.Digest {
		return ErrInvalidToken
	}
	return nil
}

func (t *RefreshToken) Revoke(at time.Time, replacementID string) {
	when := at.UTC()
	t.RevokedAt = &when
	t.ReplacedBy = replacementID
}
