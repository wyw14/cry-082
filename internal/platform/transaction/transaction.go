package transaction

import "context"

type Unit interface {
	Bind(context.Context) context.Context
	Commit(context.Context) error
	Rollback(context.Context) error
}

type Manager interface {
	Begin(context.Context) (Unit, error)
}

func Execute(ctx context.Context, manager Manager, work func(context.Context) error) (err error) {
	unit, err := manager.Begin(ctx)
	if err != nil {
		return err
	}
	bound := unit.Bind(ctx)
	defer func() {
		if err != nil {
			_ = unit.Rollback(context.WithoutCancel(ctx))
		}
	}()
	if err = work(bound); err != nil {
		return err
	}
	if err = unit.Commit(bound); err != nil {
		return err
	}
	return nil
}
