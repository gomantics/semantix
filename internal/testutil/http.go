package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gomantics/semantix/internal/api/health"
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// StatusError is returned when a response has a non-2xx status code.
type StatusError struct {
	Code int
	Body []byte
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("unexpected status %d: %s", e.Code, e.Body)
}

// State holds the test Echo server and provides HTTP helper methods.
type State struct {
	t      *testing.T
	server *echo.Echo
}

// NewState creates a State backed by a minimal Echo server with all routes registered.
func NewState(t *testing.T) *State {
	t.Helper()

	l := zap.NewNop()
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	health.Configure(e, l)

	return &State{t: t, server: e}
}

// Get performs an authenticated GET request against the test server.
func (s *State) Get(path string) (map[string]any, error) {
	return s.do(http.MethodGet, path, nil)
}

// Post performs a POST request with a JSON body.
func (s *State) Post(path string, body any) (map[string]any, error) {
	return s.do(http.MethodPost, path, body)
}

// Put performs a PUT request with a JSON body.
func (s *State) Put(path string, body any) (map[string]any, error) {
	return s.do(http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (s *State) Delete(path string) (map[string]any, error) {
	return s.do(http.MethodDelete, path, nil)
}

func (s *State) do(method, path string, body any) (map[string]any, error) {
	s.t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set(echo.MIMEApplicationJSON, "application/json")
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	s.server.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &StatusError{Code: resp.StatusCode, Body: rawBody}
	}

	var result map[string]any
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &result); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return result, nil
}

// RequireStatus asserts that err is a *StatusError with the expected code,
// or that err is nil when expectedCode is 2xx. Fails the test immediately.
func RequireStatus(t *testing.T, err error, expectedCode int) {
	t.Helper()

	if err == nil {
		if expectedCode >= 200 && expectedCode < 300 {
			return
		}
		t.Fatalf("expected status %d but request succeeded", expectedCode)
	}

	se, ok := err.(*StatusError)
	if !ok {
		t.Fatalf("unexpected error (not a StatusError): %v", err)
	}

	if se.Code != expectedCode {
		t.Fatalf("expected status %d, got %d: %s", expectedCode, se.Code, se.Body)
	}
}

// Wrap adapts a web.HandlerFunc into the echo handler format for direct testing.
func Wrap(h web.HandlerFunc, l *zap.Logger) echo.HandlerFunc {
	return web.Wrap(h, l)
}
