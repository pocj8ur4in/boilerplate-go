package handler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

// InitForTest initializes for testing.
//
//revive:disable:unexported-return // returns unexported type for testing purposes
func InitForTest(t *testing.T) *client {
	t.Helper()

	testClient := &client{
		log:   logger.InitForTest(t),
		db:    database.InitForTest(t),
		jwt:   jwt.InitForTest(t),
		redis: redis.InitForTest(t),
	}

	require.NotNil(t, testClient.log)
	require.NotNil(t, testClient.db)
	require.NotNil(t, testClient.jwt)
	require.NotNil(t, testClient.redis)

	return testClient
}
