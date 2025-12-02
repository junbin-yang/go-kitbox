package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/junbin-yang/go-kitbox/pkg/zallocrout"
)

func setupTestRouter() *HTTPAdapter {
	router := zallocrout.NewRouter()
	router.AddRoute("GET", "/", homeHandler, loggingMiddleware)
	router.AddRoute("GET", "/health", healthHandler, loggingMiddleware)
	router.AddRoute("GET", "/users", listUsersHandler, loggingMiddleware)
	router.AddRoute("GET", "/users/:id", getUserHandler, loggingMiddleware)
	router.AddRoute("POST", "/users", createUserHandler, loggingMiddleware, authMiddleware)
	router.AddRoute("GET", "/users/:userId/posts/:postId", getPostHandler, loggingMiddleware)
	router.AddRoute("GET", "/files/*path", getFileHandler, loggingMiddleware)
	router.AddRoute("GET", "/admin/users", adminHandler, loggingMiddleware, authMiddleware)
	router.AddRoute("GET", "/metrics", metricsHandler(router), loggingMiddleware)
	return NewHTTPAdapter(router)
}

func TestHTTP_HomeHandler(t *testing.T) {
	adapter := setupTestRouter()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "欢迎") {
		t.Errorf("body should contain welcome message")
	}
}

func TestHTTP_GetUserHandler(t *testing.T) {
	adapter := setupTestRouter()
	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "123") {
		t.Errorf("body should contain user_id 123")
	}
}

func TestHTTP_GetPostHandler(t *testing.T) {
	adapter := setupTestRouter()
	req := httptest.NewRequest("GET", "/users/123/posts/456", nil)
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "123") || !strings.Contains(bodyStr, "456") {
		t.Errorf("body should contain both user_id and post_id")
	}
}

func TestHTTP_GetFileHandler(t *testing.T) {
	adapter := setupTestRouter()
	req := httptest.NewRequest("GET", "/files/docs/readme.md", nil)
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "docs/readme.md") {
		t.Errorf("body should contain file path")
	}
}

func TestHTTP_AdminHandler(t *testing.T) {
	adapter := setupTestRouter()
	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want 200", resp.StatusCode)
	}
}

func TestHTTP_AdminHandler_Unauthorized(t *testing.T) {
	adapter := setupTestRouter()
	req := httptest.NewRequest("GET", "/admin/users", nil)
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status code = %d, want 401", resp.StatusCode)
	}
}

func TestHTTP_NotFound(t *testing.T) {
	adapter := setupTestRouter()
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status code = %d, want 404", resp.StatusCode)
	}
}

func BenchmarkHTTP_StaticRoute(b *testing.B) {
	adapter := setupTestRouter()
	req := httptest.NewRequest("GET", "/users", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)
	}
}

func BenchmarkHTTP_ParamRoute(b *testing.B) {
	adapter := setupTestRouter()
	req := httptest.NewRequest("GET", "/users/123", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)
	}
}

func BenchmarkHTTP_MultiParamRoute(b *testing.B) {
	adapter := setupTestRouter()
	req := httptest.NewRequest("GET", "/users/123/posts/456", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)
	}
}
