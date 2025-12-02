package main

import (
	"encoding/json"
	"testing"

	"github.com/junbin-yang/go-kitbox/pkg/zallocrout"
)

func setupTestAdapter() *RPCAdapter {
	router := zallocrout.NewRouter()
	router.AddRoute("RPC", "/user.get", getUserRPC, loggingMiddleware, validationMiddleware)
	router.AddRoute("RPC", "/user.list", listUsersRPC, loggingMiddleware, validationMiddleware)
	router.AddRoute("RPC", "/user.create", createUserRPC, loggingMiddleware, validationMiddleware)
	return NewRPCAdapter(router)
}

func TestRPC_UserGet(t *testing.T) {
	adapter := setupTestAdapter()
	req := &RPCRequest{
		Method: "user.get",
		Params: json.RawMessage(`"123"`),
		ID:     1,
	}
	resp := adapter.HandleRequest(req)
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error.Message)
	}
	if resp.Result == nil {
		t.Error("result should not be nil")
	}
}

func TestRPC_UserList(t *testing.T) {
	adapter := setupTestAdapter()
	req := &RPCRequest{
		Method: "user.list",
		Params: json.RawMessage(`{}`),
		ID:     2,
	}
	resp := adapter.HandleRequest(req)
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error.Message)
	}
	if resp.Result == nil {
		t.Error("result should not be nil")
	}
}

func TestRPC_UserCreate(t *testing.T) {
	adapter := setupTestAdapter()
	req := &RPCRequest{
		Method: "user.create",
		Params: json.RawMessage(`{"name":"Alice","age":25}`),
		ID:     3,
	}
	resp := adapter.HandleRequest(req)
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error.Message)
	}
	if resp.Result == nil {
		t.Error("result should not be nil")
	}
}

func TestRPC_MethodNotFound(t *testing.T) {
	adapter := setupTestAdapter()
	req := &RPCRequest{
		Method: "user.nonexistent",
		Params: json.RawMessage(`{}`),
		ID:     4,
	}
	resp := adapter.HandleRequest(req)
	if resp.Error == nil {
		t.Error("expected error for nonexistent method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
}

func BenchmarkRPC_UserGet(b *testing.B) {
	adapter := setupTestAdapter()
	req := &RPCRequest{
		Method: "user.get",
		Params: json.RawMessage(`"123"`),
		ID:     1,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.HandleRequest(req)
	}
}

func BenchmarkRPC_UserList(b *testing.B) {
	adapter := setupTestAdapter()
	req := &RPCRequest{
		Method: "user.list",
		Params: json.RawMessage(`{}`),
		ID:     2,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.HandleRequest(req)
	}
}

func BenchmarkRPC_UserCreate(b *testing.B) {
	adapter := setupTestAdapter()
	req := &RPCRequest{
		Method: "user.create",
		Params: json.RawMessage(`{"name":"Alice","age":25}`),
		ID:     3,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.HandleRequest(req)
	}
}
