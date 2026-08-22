package authn

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wyw14/cry-082/internal/domain/auth"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
type Repository interface {
	FindUserByUsername(context.Context, string) (auth.User, error)
	FindUserByID(context.Context, string) (auth.User, error)
	SaveUser(context.Context, auth.User) error
	SaveRefreshToken(context.Context, auth.RefreshToken) error
	FindRefreshToken(context.Context, string) (auth.RefreshToken, error)
	RotateRefreshToken(context.Context, auth.RefreshToken, auth.RefreshToken) error
}
type TokenIssuer interface {
	IssueAccessToken(context.Context, auth.User, time.Duration) (string, time.Time, error)
}

type Service struct {
	repository            Repository
	issuer                TokenIssuer
	clock                 Clock
	ids                   IDGenerator
	accessTTL, refreshTTL time.Duration
}

func New(repository Repository, issuer TokenIssuer, clock Clock, ids IDGenerator, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{repository: repository, issuer: issuer, clock: clock, ids: ids, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

func (s *Service) Login(ctx context.Context, username, password string) (TokenPair, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	user, err := s.repository.FindUserByUsername(ctx, username)
	if err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}
	if err := user.CanLogin(s.clock.Now()); err != nil {
		return TokenPair{}, err
	}
	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		user.RecordFailure(s.clock.Now(), 5, 15*time.Minute)
		_ = s.repository.SaveUser(ctx, user)
		return TokenPair{}, ErrInvalidCredentials
	}
	user.RecordSuccess()
	if err := s.repository.SaveUser(ctx, user); err != nil {
		return TokenPair{}, err
	}
	return s.issuePair(ctx, user)
}

func (s *Service) Refresh(ctx context.Context, tokenID, raw string) (TokenPair, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stored, err := s.repository.FindRefreshToken(ctx, tokenID)
	if err != nil {
		return TokenPair{}, auth.ErrInvalidToken
	}
	if err := stored.Validate(raw, s.clock.Now()); err != nil {
		return TokenPair{}, err
	}
	user, err := s.repository.FindUserByID(ctx, stored.UserID)
	if err != nil {
		return TokenPair{}, err
	}
	pair, replacement, err := s.createPair(ctx, user)
	if err != nil {
		return TokenPair{}, err
	}
	stored.Revoke(s.clock.Now(), replacement.ID)
	if err := s.repository.RotateRefreshToken(ctx, stored, replacement); err != nil {
		return TokenPair{}, err
	}
	return pair, nil
}

func (s *Service) issuePair(ctx context.Context, user auth.User) (TokenPair, error) {
	pair, refresh, err := s.createPair(ctx, user)
	if err != nil {
		return TokenPair{}, err
	}
	if err := s.repository.SaveRefreshToken(ctx, refresh); err != nil {
		return TokenPair{}, err
	}
	return pair, nil
}

func (s *Service) createPair(ctx context.Context, user auth.User) (TokenPair, auth.RefreshToken, error) {
	access, accessExpires, err := s.issuer.IssueAccessToken(ctx, user, s.accessTTL)
	if err != nil {
		return TokenPair{}, auth.RefreshToken{}, err
	}
	rawBytes := make([]byte, 48)
	if _, err := rand.Read(rawBytes); err != nil {
		return TokenPair{}, auth.RefreshToken{}, err
	}
	raw := base64.RawURLEncoding.EncodeToString(rawBytes)
	refreshExpires := s.clock.Now().Add(s.refreshTTL)
	refresh, err := auth.NewRefreshToken(s.ids.NewID(), user.ID, raw, s.clock.Now(), refreshExpires)
	if err != nil {
		return TokenPair{}, auth.RefreshToken{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh.ID + "." + raw, AccessExpiresAt: accessExpires, RefreshExpiresAt: refreshExpires}, refresh, nil
}
