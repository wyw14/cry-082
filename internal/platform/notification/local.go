package notification

import (
	"context"
	"sync"
	"time"
)

type Message struct {
	Recipient, Subject string
	Data               map[string]string
	SentAt             time.Time
}
type Local struct {
	mu       sync.Mutex
	messages []Message
}

func NewLocal() *Local { return &Local{} }
func (l *Local) Notify(ctx context.Context, recipient, subject string, data map[string]string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	copyData := make(map[string]string, len(data))
	for key, value := range data {
		copyData[key] = value
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, Message{Recipient: recipient, Subject: subject, Data: copyData, SentAt: time.Now().UTC()})
	return nil
}
func (l *Local) Messages() []Message {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Message(nil), l.messages...)
}
