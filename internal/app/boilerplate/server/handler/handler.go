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

// client implements api.ServerInterface.
type client struct {
	// log provides logger client.
	log logger.Client

	// db provides database queries client.
	db database.Client

	// jwt provides JWT service client.
	jwt jwt.Client

	// redis provides redis connection client.
	redis redis.Client
}

// ConstructorParams represents parameters for constructor.
type ConstructorParams struct {
	// In extends fx.In.
	fx.In

	// Log provides logger client.
	Log logger.Client

	// DB provides database queries client.
	DB database.Client

	// JWT provides JWT service client.
	JWT jwt.Client

	// Redis provides redis connection client.
	Redis redis.Client
}

// NewModule provides module for handler.
func NewModule() fx.Option {
	return fx.Module("handler",
		// provide concrete type for constructor
		fx.Provide(func(params ConstructorParams) (api.ServerInterface, error) {
			return &client{
				log:   params.Log,
				db:    params.DB,
				jwt:   params.JWT,
				redis: params.Redis,
			}, nil
		}),
	)
}

// sendResponse sends response.
func (c *client) sendResponse(writer http.ResponseWriter, code int, data interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)

	// encode response
	if err := json.NewEncoder(writer).Encode(data); err != nil {
		c.log.Error("failed to encode response", "error", err)
	}
}

// sendError sends error response.
func (c *client) sendError(writer http.ResponseWriter, code int, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)

	// encode error response
	if err := json.NewEncoder(writer).Encode(map[string]string{"error": message}); err != nil {
		c.log.Error("failed to encode error response", "error", err)
	}
}
