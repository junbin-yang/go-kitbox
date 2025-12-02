package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/junbin-yang/go-kitbox/pkg/zallocrout"
)

// RPC 请求和响应结构
type RPCRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	ID     int             `json:"id"`
}

type RPCResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *RPCError   `json:"error,omitempty"`
	ID     int         `json:"id"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RPCAdapter RPC 适配器
type RPCAdapter struct {
	router *zallocrout.Router
}

// NewRPCAdapter 创建 RPC 适配器
func NewRPCAdapter(router *zallocrout.Router) *RPCAdapter {
	return &RPCAdapter{router: router}
}

// HandleRequest 处理 RPC 请求
func (a *RPCAdapter) HandleRequest(req *RPCRequest) *RPCResponse {
	// 将 RPC 方法映射到路由路径
	// 例如: "user.get" -> "/user/get"
	path := "/" + req.Method

	// 创建 context 并匹配路由
	ctx, handler, middlewares, ok := a.router.Match("RPC", path, context.Background())
	if !ok {
		return &RPCResponse{
			Error: &RPCError{
				Code:    -32601,
				Message: "Method not found",
			},
			ID: req.ID,
		}
	}
	defer zallocrout.ReleaseContext(ctx) // 确保 context 被释放

	// 设置 RPC 请求数据到 context
	zallocrout.SetValue(ctx, "rpc.request", req)
	zallocrout.SetValue(ctx, "rpc.params", req.Params)

	// 应用中间件链
	finalHandler := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		finalHandler = middlewares[i](finalHandler)
	}

	// 执行处理器（不使用 ExecuteHandler，因为需要在释放 context 前读取结果）
	if err := finalHandler(ctx); err != nil {
		return &RPCResponse{
			Error: &RPCError{
				Code:    -32603,
				Message: err.Error(),
			},
			ID: req.ID,
		}
	}

	// 获取结果（在释放 context 之前）
	result, _ := ctx.Value("result").(interface{})

	return &RPCResponse{
		Result: result,
		ID:     req.ID,
	}
}

// RPC 处理器示例

// getUserRPC 获取用户信息
func getUserRPC(ctx context.Context) error {
	params := ctx.Value("rpc.params").(json.RawMessage)

	var userID string
	if err := json.Unmarshal(params, &userID); err != nil {
		return fmt.Errorf("invalid params: %v", err)
	}

	// 模拟业务逻辑
	result := map[string]interface{}{
		"id":   userID,
		"name": fmt.Sprintf("User %s", userID),
		"age":  25,
	}

	// 将结果存储到 context（通过 SetValue）
	zallocrout.SetValue(ctx, "result", result)

	return nil
}

// listUsersRPC 列出用户
func listUsersRPC(ctx context.Context) error {
	// 模拟业务逻辑
	result := []map[string]interface{}{
		{"id": "1", "name": "Alice"},
		{"id": "2", "name": "Bob"},
		{"id": "3", "name": "Charlie"},
	}

	zallocrout.SetValue(ctx, "result", result)
	return nil
}

// createUserRPC 创建用户
func createUserRPC(ctx context.Context) error {
	params := ctx.Value("rpc.params").(json.RawMessage)

	var userData map[string]interface{}
	if err := json.Unmarshal(params, &userData); err != nil {
		return fmt.Errorf("invalid params: %v", err)
	}

	// 模拟业务逻辑
	result := map[string]interface{}{
		"id":      "123",
		"name":    userData["name"],
		"created": true,
	}

	zallocrout.SetValue(ctx, "result", result)
	return nil
}

// RPC 中间件示例

// loggingMiddleware 日志中间件
func loggingMiddleware(next zallocrout.HandlerFunc) zallocrout.HandlerFunc {
	return func(ctx context.Context) error {
		req := ctx.Value("rpc.request").(*RPCRequest)
		log.Printf("[RPC] Method: %s, ID: %d", req.Method, req.ID)
		return next(ctx)
	}
}

// validationMiddleware 验证中间件
func validationMiddleware(next zallocrout.HandlerFunc) zallocrout.HandlerFunc {
	return func(ctx context.Context) error {
		req := ctx.Value("rpc.request").(*RPCRequest)

		// 验证请求
		if req.Method == "" {
			return fmt.Errorf("method is required")
		}

		return next(ctx)
	}
}

func main() {
	// 创建路由器
	router := zallocrout.NewRouter()

	// 注册 RPC 方法（映射到路由）
	router.AddRoute("RPC", "/user.get", getUserRPC, loggingMiddleware, validationMiddleware)
	router.AddRoute("RPC", "/user.list", listUsersRPC, loggingMiddleware, validationMiddleware)
	router.AddRoute("RPC", "/user.create", createUserRPC, loggingMiddleware, validationMiddleware)

	// 创建 RPC 适配器
	adapter := NewRPCAdapter(router)

	// 示例：处理 RPC 请求
	log.Println("RPC Adapter Example")
	log.Println("===================")

	// 示例 1: 获取用户
	req1 := &RPCRequest{
		Method: "user.get",
		Params: json.RawMessage(`"123"`),
		ID:     1,
	}
	resp1 := adapter.HandleRequest(req1)
	printResponse("user.get", resp1)

	// 示例 2: 列出用户
	req2 := &RPCRequest{
		Method: "user.list",
		Params: json.RawMessage(`{}`),
		ID:     2,
	}
	resp2 := adapter.HandleRequest(req2)
	printResponse("user.list", resp2)

	// 示例 3: 创建用户
	req3 := &RPCRequest{
		Method: "user.create",
		Params: json.RawMessage(`{"name":"David","age":30}`),
		ID:     3,
	}
	resp3 := adapter.HandleRequest(req3)
	printResponse("user.create", resp3)

	// 示例 4: 调用不存在的方法
	req4 := &RPCRequest{
		Method: "user.delete",
		Params: json.RawMessage(`"123"`),
		ID:     4,
	}
	resp4 := adapter.HandleRequest(req4)
	printResponse("user.delete (not found)", resp4)
}

func printResponse(label string, resp *RPCResponse) {
	data, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("\n%s:\n%s\n", label, string(data))
}
