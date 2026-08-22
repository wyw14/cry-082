package memory

import (
	"context"

	"github.com/wyw14/cry-082/internal/platform/transaction"
)

type TransactionManager struct{}
type memoryTransaction struct{ finished bool }

func (TransactionManager) Begin(ctx context.Context) (transaction.Unit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &memoryTransaction{}, nil
}
func (t *memoryTransaction) Bind(ctx context.Context) context.Context { return ctx }
func (t *memoryTransaction) Commit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	t.finished = true
	return nil
}
func (t *memoryTransaction) Rollback(ctx context.Context) error { t.finished = true; return nil }
