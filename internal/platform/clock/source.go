package clock

import (
	"errors"
	"sync"
	"time"
)

var ErrUnknownTimezone = errors.New("unknown timezone")

type System struct {
	now func() time.Time
}

func NewSystem() System {
	return System{now: time.Now}
}

func (s System) Now() time.Time {
	return s.now().UTC()
}

type Manual struct {
	mu      sync.RWMutex
	current time.Time
}

func NewManual(current time.Time) *Manual {
	return &Manual{current: current.UTC()}
}

func (m *Manual) Now() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Manual) Advance(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = m.current.Add(duration)
}

func (m *Manual) Set(current time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = current.UTC()
}

func LocalDate(at time.Time, timezone string) (string, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return "", ErrUnknownTimezone
	}
	return at.In(location).Format("2006-01-02"), nil
}
