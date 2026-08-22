package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/wyw14/cry-082/internal/config"
)

func Run(cfg config.Config) error {
	lifetime, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app, err := Build(lifetime, cfg)
	if err != nil {
		return fmt.Errorf("build application: %w", err)
	}
	serveResult := make(chan error, 1)
	go func() {
		app.Logger.Info("monitoring api started", zap.String("listen_address", cfg.HTTPAddress))
		serveResult <- app.Server.ListenAndServe()
	}()
	select {
	case <-lifetime.Done():
		app.Logger.Info("process shutdown signal received")
	case serveErr := <-serveResult:
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return fmt.Errorf("serve monitoring api: %w", serveErr)
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := app.Close(shutdown); err != nil {
		return fmt.Errorf("close application: %w", err)
	}
	return nil
}
