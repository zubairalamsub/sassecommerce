package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func serveReady(t *testing.T, ping func(context.Context) error) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ready", readinessCheck(ping))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))
	return w
}

func TestReadinessCheck_DatabaseUp(t *testing.T) {
	w := serveReady(t, func(context.Context) error { return nil })

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ready"`)
	assert.Contains(t, w.Body.String(), `"database":"ok"`)
}

func TestReadinessCheck_DatabaseDown(t *testing.T) {
	w := serveReady(t, func(context.Context) error {
		return errors.New("server selection error: mongodb-internal.example:27017")
	})

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"not ready"`)
	assert.False(t, strings.Contains(w.Body.String(), "mongodb-internal"),
		"the driver error names the database host and must not reach the response")
}

// A ping that never returns on its own must be cut off by readinessTimeout,
// not left to hang the probe.
func TestReadinessCheck_HungPingIsBounded(t *testing.T) {
	start := time.Now()
	w := serveReady(t, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Less(t, time.Since(start), readinessTimeout+time.Second)
}
