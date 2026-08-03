package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAccessLoggerRedactsAuthorizationCanary proves request diagnostics never
// carry credential material: a request bearing an Authorization canary must
// not appear in access log output.
func TestAccessLoggerRedactsAuthorizationCanary(t *testing.T) {
	const canary = "eyJhbGciOiJIUzI1NiJ9.canary-auth-token-do-not-log.signature"

	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := AccessLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/instances", nil)
	req.Header.Set("Authorization", "Bearer "+canary)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.NotEmpty(t, buf.String(), "access logger should emit a record")
	require.NotContains(t, buf.String(), canary, "access log leaked authorization token")
	require.NotContains(t, buf.String(), "authorization", "access log should not record headers at all")
}
