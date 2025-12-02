package main

import (
	"strings"
	"testing"

	"github.com/junbin-yang/go-kitbox/pkg/zallocrout"
)

func setupTestCLIAdapter() *CLIAdapter {
	router := zallocrout.NewRouter()
	router.AddRoute("CLI", "/user/list", userListCommand, loggingMiddleware, validationMiddleware)
	router.AddRoute("CLI", "/user/get/:id", userGetCommand, loggingMiddleware, validationMiddleware)
	router.AddRoute("CLI", "/user/create/:name", userCreateCommand, loggingMiddleware, validationMiddleware)
	router.AddRoute("CLI", "/config/get/:key", configGetCommand, loggingMiddleware, validationMiddleware)
	router.AddRoute("CLI", "/help", helpCommand, loggingMiddleware)
	return &CLIAdapter{router: router}
}

func TestCLI_UserList(t *testing.T) {
	adapter := setupTestCLIAdapter()
	err := adapter.Execute([]string{"user", "list"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCLI_UserGet(t *testing.T) {
	adapter := setupTestCLIAdapter()
	err := adapter.Execute([]string{"user", "get", "123"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCLI_UserCreate(t *testing.T) {
	adapter := setupTestCLIAdapter()
	err := adapter.Execute([]string{"user", "create", "Alice"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCLI_ConfigGet(t *testing.T) {
	adapter := setupTestCLIAdapter()
	err := adapter.Execute([]string{"config", "get", "database.host"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCLI_Help(t *testing.T) {
	adapter := setupTestCLIAdapter()
	err := adapter.Execute([]string{"help"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	adapter := setupTestCLIAdapter()
	err := adapter.Execute([]string{"unknown", "command"})
	if err == nil {
		t.Error("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("error should contain 'unknown command', got: %v", err)
	}
}

func BenchmarkCLI_UserList(b *testing.B) {
	adapter := setupTestCLIAdapter()
	args := []string{"user", "list"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = adapter.Execute(args)
	}
}

func BenchmarkCLI_UserGet(b *testing.B) {
	adapter := setupTestCLIAdapter()
	args := []string{"user", "get", "123"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = adapter.Execute(args)
	}
}

func BenchmarkCLI_UserCreate(b *testing.B) {
	adapter := setupTestCLIAdapter()
	args := []string{"user", "create", "Alice"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = adapter.Execute(args)
	}
}
