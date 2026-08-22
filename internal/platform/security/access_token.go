package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/wyw14/cry-082/internal/domain/auth"
)

var ErrInvalidAccessToken = errors.New("invalid access token")

type AccessClaims struct {
	Subject   string `json:"sub"`
	Username  string `json:"username"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type AccessTokens struct {
	key   []byte
	clock func() time.Time
}

func NewAccessTokens(key string, clock func() time.Time) (*AccessTokens, error) {
	if len(key) < 32 || clock == nil {
		return nil, ErrInvalidAccessToken
	}
	return &AccessTokens{key: []byte(key), clock: clock}, nil
}

func (a *AccessTokens) IssueAccessToken(ctx context.Context, user auth.User, ttl time.Duration) (string, time.Time, error) {
	if err := ctx.Err(); err != nil {
		return "", time.Time{}, err
	}
	if user.ID == "" || ttl <= 0 {
		return "", time.Time{}, ErrInvalidAccessToken
	}
	now := a.clock().UTC()
	expires := now.Add(ttl)
	payload, err := json.Marshal(AccessClaims{Subject: user.ID, Username: user.Username, IssuedAt: now.Unix(), ExpiresAt: expires.Unix()})
	if err != nil {
		return "", time.Time{}, err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + a.sign(encoded), expires, nil
}

func (a *AccessTokens) VerifyAccessToken(ctx context.Context, raw string) (AccessClaims, error) {
	if err := ctx.Err(); err != nil {
		return AccessClaims{}, err
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || !hmac.Equal([]byte(parts[1]), []byte(a.sign(parts[0]))) {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	var claims AccessClaims
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Subject == "" || claims.ExpiresAt <= claims.IssuedAt {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	if !a.clock().UTC().Before(time.Unix(claims.ExpiresAt, 0)) {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	return claims, nil
}

func (a *AccessTokens) sign(payload string) string {
	mac := hmac.New(sha256.New, a.key)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
