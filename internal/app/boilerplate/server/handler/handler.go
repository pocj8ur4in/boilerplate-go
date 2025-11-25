// Package handler provides API handlers on http server.
package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/fx"

	genAPI "github.com/pocj8ur4in/boilerplate-go/internal/gen/api"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

// client implements api.ServerInterface.
type client struct {
	// log provides logger client.
	log logger.Client

	// database provides database queries client.
	database database.Client

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

	// DataBase provides database queries client.
	DataBase database.Client

	// JWT provides JWT service client.
	JWT jwt.Client

	// Redis provides redis connection client.
	Redis redis.Client
}

// NewModule provides module for handler.
func NewModule() fx.Option {
	return fx.Module("handler",
		// provide concrete type for constructor
		fx.Provide(func(params ConstructorParams) (genAPI.ServerInterface, error) {
			return &client{
				log:      params.Log,
				database: params.DataBase,
				jwt:      params.JWT,
				redis:    params.Redis,
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
func (c *client) sendError(writer http.ResponseWriter, httpStatus int, errorType, message string) {
	genAPI.SendError(writer, httpStatus, errorType, message)
}
