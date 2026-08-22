package auth

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidUser = errors.New("invalid user")
	ErrUserLocked  = errors.New("user locked")
)

type User struct {
	ID             string
	Username       string
	PasswordHash   []byte
	DisplayName    string
	MaskedPhone    string
	Active         bool
	FailedAttempts int
	LockedUntil    time.Time
	Version        int64
}

func NewUser(id, username, displayName string, passwordHash []byte) (User, error) {
	if strings.TrimSpace(id) == "" || len(strings.TrimSpace(username)) < 3 || strings.TrimSpace(displayName) == "" || len(passwordHash) == 0 {
		return User{}, ErrInvalidUser
	}
	return User{ID: id, Username: strings.ToLower(strings.TrimSpace(username)), DisplayName: displayName, PasswordHash: append([]byte(nil), passwordHash...), Active: true, Version: 1}, nil
}

func (u *User) RecordFailure(now time.Time, maximum int, lockDuration time.Duration) {
	u.FailedAttempts++
	if u.FailedAttempts >= maximum {
		u.LockedUntil = now.UTC().Add(lockDuration)
		u.FailedAttempts = 0
	}
	u.Version++
}

func (u *User) RecordSuccess() {
	u.FailedAttempts = 0
	u.LockedUntil = time.Time{}
	u.Version++
}

func (u User) CanLogin(now time.Time) error {
	if !u.Active || now.UTC().Before(u.LockedUntil) {
		return ErrUserLocked
	}
	return nil
}
