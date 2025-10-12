// Package app provides the application.
package app

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	configPkg "github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/config"
	serverPkg "github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server"
	databasePkg "github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	jwtPkg "github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	loggerPkg "github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

// New creates a new application.
func New() *fx.App {
	return fx.New(
		// modules
		configPkg.NewModule(),
		loggerPkg.NewModule(),
		databasePkg.NewModule(),
		redis.NewModule(),
		jwtPkg.NewModule(),
		serverPkg.NewModule(),

		// lifecycle hooks
		fx.Invoke(registerHooks),
	)
}

// registerHooks registers lifecycle hooks for the application.
func registerHooks(
	lifecycle fx.Lifecycle,
	dbConn *databasePkg.DB,
	log *loggerPkg.Logger,
	redisConn *redis.Redis,
	server *serverPkg.Server,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			log.Info().Msg("starting application...")

			// start server in a goroutine
			go func() {
				if err := server.Run(); err != nil {
					log.Error().Err(err).Msg("server failed to run")
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info().Msg("shutting down application...")

			// shutdown server
			if err := server.Shutdown(ctx); err != nil {
				log.Error().Err(err).Msg("failed to shutdown server")

				return fmt.Errorf("shutdown server: %w", err)
			}

			// close database
			if err := dbConn.Close(); err != nil {
				log.Error().Err(err).Msg("failed to close database")

				return fmt.Errorf("close database: %w", err)
			}

			// close redis
			if err := redisConn.Close(); err != nil {
				log.Error().Err(err).Msg("failed to close redis")

				return fmt.Errorf("close redis: %w", err)
			}

			log.Info().Msg("application stopped")

			return nil
		},
	})
}
