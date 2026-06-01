//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealth_OK(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)

	resp, err := http.Get(gw.Server.URL + "/healthz")
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHealth_RequestIDHeaderPresent(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)

	resp, err := http.Get(gw.Server.URL + "/healthz")
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.NotEmpty(t, resp.Header.Get("X-Request-Id"),
		"chi RequestID middleware must set X-Request-Id on every response")
}
