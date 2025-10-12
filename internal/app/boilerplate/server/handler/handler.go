// Package handler provides API handlers on http server.
package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/fx"

	"github.com/pocj8ur4in/boilerplate-go/internal/gen/api"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

// NewModule provides module for handler.
func NewModule() fx.Option {
	return fx.Module("handler",
		fx.Provide(New),
	)
}

// Handler implements api.ServerInterface.
type Handler struct {
	logger *logger.Logger
	db     *database.DB
	redis  *redis.Redis
	jwt    *jwt.JWT
}

// New creates a new handler instance.
func New(
	log *logger.Logger,
	dbConn *database.DB,
	redisConn *redis.Redis,
	jwt *jwt.JWT,
) api.ServerInterface {
	return &Handler{
		logger: log,
		db:     dbConn,
		redis:  redisConn,
		jwt:    jwt,
	}
}

// sendResponse sends response.
func (h *Handler) sendResponse(writer http.ResponseWriter, code int, data interface{}) {
	// set response header
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)

	// encode response
	if err := json.NewEncoder(writer).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode response")
	}
}

// sendError sends error response.
func (h *Handler) sendError(writer http.ResponseWriter, code int, message string) {
	// set response header
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)

	// encode error response
	if err := json.NewEncoder(writer).Encode(map[string]string{"error": message}); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode error response")
	}
}
