// Package app provides the application.
package app

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/config"
	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server"
	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server/handler"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

// New creates a new application.
func New() *fx.App {
	return fx.New(
		// modules
		config.NewModule(),
		logger.NewModule(),
		database.NewModule(),
		redis.NewModule(),
		jwt.NewModule(),
		handler.NewModule(),
		server.NewModule(),

		// register lifecycle hooks
		fx.Invoke(registerHooks),
	)
}

// registerHooks registers lifecycle hooks for the application.
func registerHooks(lifecycle fx.Lifecycle, log logger.Client, server server.Client) {
	lifecycle.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			log.Info("starting application...")

			// start server in a goroutine
			go func() {
				if err := server.Run(); err != nil {
					log.Error("server failed to run", "error", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("shutting down application...")

			// shutdown server
			if err := server.Shutdown(ctx); err != nil {
				log.Error("failed to shutdown server", "error", err)

				return fmt.Errorf("shutdown server: %w", err)
			}

			log.Info("application stopped")

			return nil
		},
	})
}
