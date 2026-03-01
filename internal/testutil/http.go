package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gomantics/semantix/internal/api/auth"
	"github.com/gomantics/semantix/internal/api/gittokens"
	"github.com/gomantics/semantix/internal/api/health"
	"github.com/gomantics/semantix/internal/api/repositories"
	"github.com/gomantics/semantix/internal/api/search"
	"github.com/gomantics/semantix/internal/api/workspaces"
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
	cookie *http.Cookie
}

// NewState creates a State backed by a minimal Echo server with all routes registered.
func NewState(t *testing.T) *State {
	t.Helper()

	l := zap.NewNop()
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	health.Configure(e, l)
	auth.Configure(e, l)
	workspaces.Configure(e, l)
	gittokens.Configure(e, l)
	repositories.Configure(e, l)
	search.Configure(e, l)

	return &State{t: t, server: e}
}

// NewAuthState creates a State with a session cookie pre-set by logging in as
// the admin user created by WithAdminUser(). Tests using this helper must
// include WithAdminUser() in their TestMain options.
func NewAuthState(t *testing.T) *State {
	t.Helper()

	s := NewState(t)

	_, err := s.Post("/v1/auth/login", map[string]any{
		"email":    AdminCreds.Email,
		"password": AdminCreds.Password,
	})
	if err != nil {
		t.Fatalf("NewAuthState: login failed: %v", err)
	}

	if s.cookie == nil {
		t.Fatal("NewAuthState: no session cookie set after login")
	}

	return s
}

// Get performs a GET request against the test server.
func (s *State) Get(path string) (map[string]any, error) {
	return s.do(http.MethodGet, path, nil)
}

// GetStatus performs a GET request and returns only the error (useful for
// asserting non-2xx status codes without caring about the body).
func (s *State) GetStatus(path string) error {
	_, err := s.do(http.MethodGet, path, nil)
	return err
}

// Post performs a POST request with a JSON body.
func (s *State) Post(path string, body any) (map[string]any, error) {
	return s.do(http.MethodPost, path, body)
}

// PostStatus performs a POST request and returns only the error.
func (s *State) PostStatus(path string, body any) error {
	_, err := s.do(http.MethodPost, path, body)
	return err
}

// Put performs a PUT request with a JSON body.
func (s *State) Put(path string, body any) (map[string]any, error) {
	return s.do(http.MethodPut, path, body)
}

// DeleteStatus performs a DELETE request and returns only the error.
func (s *State) DeleteStatus(path string) error {
	_, err := s.do(http.MethodDelete, path, nil)
	return err
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
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	if s.cookie != nil {
		req.AddCookie(s.cookie)
	}

	rec := httptest.NewRecorder()
	s.server.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	// Capture session cookie from signup/login responses.
	for _, c := range resp.Cookies() {
		if c.Name == "session_token" {
			if c.MaxAge >= 0 && c.Value != "" {
				s.cookie = c
			} else {
				// MaxAge < 0 means cookie was cleared (logout).
				s.cookie = nil
			}
			break
		}
	}

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
