// Package app provides the application.
package app

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	configPkg "github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/config"
	databasePkg "github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	loggerPkg "github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

// New creates a new application.
func New() *fx.App {
	return fx.New(
		// modules
		configPkg.NewModule(),
		loggerPkg.NewModule(),
		databasePkg.NewModule(),

		// lifecycle hooks
		fx.Invoke(registerHooks),
	)
}

// registerHooks registers lifecycle hooks for the application.
func registerHooks(
	lifecycle fx.Lifecycle,
	dbConn *databasePkg.DB,
	log *loggerPkg.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			log.Info().Msg("starting application...")

			return nil
		},
		OnStop: func(_ context.Context) error {
			log.Info().Msg("shutting down application...")

			// close database
			if err := dbConn.Close(); err != nil {
				log.Error().Err(err).Msg("failed to close database")

				return fmt.Errorf("close database: %w", err)
			}

			log.Info().Msg("application stopped")

			return nil
		},
	})
}
