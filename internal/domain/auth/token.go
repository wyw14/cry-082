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

type RotationPlan struct {
	Current     RefreshToken
	Replacement RefreshToken
}

func NewRotationPlan(current, replacement RefreshToken) (RotationPlan, error) {
	if strings.TrimSpace(current.ID) == "" || strings.TrimSpace(replacement.ID) == "" {
		return RotationPlan{}, ErrInvalidToken
	}
	if current.ID == replacement.ID || current.UserID != replacement.UserID {
		return RotationPlan{}, ErrInvalidToken
	}
	if current.RevokedAt != nil || replacement.RevokedAt != nil {
		return RotationPlan{}, ErrInvalidToken
	}
	if replacement.IssuedAt.Before(current.IssuedAt) || !replacement.IssuedAt.Before(replacement.ExpiresAt) {
		return RotationPlan{}, ErrInvalidToken
	}
	return RotationPlan{Current: current, Replacement: replacement}, nil
}

func (p RotationPlan) Apply(at time.Time) (RefreshToken, RefreshToken, error) {
	if p.Current.RevokedAt != nil || p.Replacement.RevokedAt != nil {
		return RefreshToken{}, RefreshToken{}, ErrInvalidToken
	}
	current := p.Current
	replacement := p.Replacement
	current.Revoke(at, replacement.ID)
	return current, replacement, nil
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
