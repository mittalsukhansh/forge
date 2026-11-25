package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mittalsukhansh/forge/internal/queue"
	"github.com/mittalsukhansh/forge/internal/store"
)

func testAPI(t *testing.T) http.Handler {
	t.Helper()

	s, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	q := queue.New(ctx, s, 0)

	t.Cleanup(func() {
		cancel()
		q.Wait()
		s.Close()
	})

	return New(s, q)
}

func TestHealth(t *testing.T) {
	handler := testAPI(t)

	req := httptest.NewRequest("GET", "/health", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != 200 {
		t.Fatal("Health check failed")
	}
}

func TestDashboard(t *testing.T) {
	handler := testAPI(t)

	req := httptest.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != 200 {
		t.Fatal("Dashboard unavailable")
	}

	if !strings.Contains(res.Body.String(), "FORGE") {
		t.Fatal("Dashboard content missing")
	}
}

func TestInvalidPipeline(t *testing.T) {
	handler := testAPI(t)

	req := httptest.NewRequest(
		"POST",
		"/api/jobs",
		strings.NewReader(`{"pipeline":"invalid"}`),
	)

	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != 400 {
		t.Fatalf("Expected 400, received %d", res.Code)
	}
}

func TestMissingJob(t *testing.T) {
	handler := testAPI(t)

	req := httptest.NewRequest("GET", "/api/jobs/999", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != 404 {
		t.Fatal("Expected 404")
	}
}

func TestJobSubmission(t *testing.T) {
	handler := testAPI(t)

	body := `{
"pipeline": "name: Test\nsteps:\n  - name: Build\n    image: alpine:3\n    commands: [echo hello]"
}`

	req := httptest.NewRequest(
		"POST",
		"/api/jobs",
		strings.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != 202 {
		t.Fatalf("Expected 202, received %d", res.Code)
	}
}
