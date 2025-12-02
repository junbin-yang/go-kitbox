package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/zallocrout"
)

// HTTPAdapter HTTP 适配器（零分配实现）
type HTTPAdapter struct {
	router *zallocrout.Router
}

// NewHTTPAdapter 创建 HTTP 适配器
func NewHTTPAdapter(router *zallocrout.Router) *HTTPAdapter {
	return &HTTPAdapter{router: router}
}

// ServeHTTP 实现 http.Handler 接口
func (h *HTTPAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, handler, middlewares, ok := h.router.Match(r.Method, r.URL.Path, r.Context())
	if !ok {
		http.NotFound(w, r)
		return
	}

	// 设置 HTTP 相关值到 context（零分配）
	zallocrout.SetValue(ctx, "http.ResponseWriter", w)
	zallocrout.SetValue(ctx, "http.Request", r)

	// 执行处理器（自动释放 context）
	if err := zallocrout.ExecuteHandler(ctx, handler, middlewares); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// 业务处理器示例

// homeHandler 首页
func homeHandler(ctx context.Context) error {
	w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "欢迎使用 zallocrout 通用路由器\n\n")
	fmt.Fprintf(w, "特性:\n")
	fmt.Fprintf(w, "  - 零内存分配 (0 allocs/op)\n")
	fmt.Fprintf(w, "  - 亚微秒延迟 (P99 < 30ns)\n")
	fmt.Fprintf(w, "  - 高吞吐量 (870万+ QPS)\n")
	fmt.Fprintf(w, "  - 通用设计 (支持 HTTP/RPC/CLI)\n")
	return nil
}

// healthHandler 健康检查
func healthHandler(ctx context.Context) error {
	w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","timestamp":%d}`, time.Now().Unix())
	return nil
}

// getUserHandler 获取用户信息
func getUserHandler(ctx context.Context) error {
	w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)
	userID, _ := zallocrout.GetParam(ctx, "id")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"user_id":"%s","name":"用户 %s"}`, userID, userID)
	return nil
}

// listUsersHandler 列出所有用户
func listUsersHandler(ctx context.Context) error {
	w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"users":[{"id":"1","name":"Alice"},{"id":"2","name":"Bob"}]}`)
	return nil
}

// createUserHandler 创建用户
func createUserHandler(ctx context.Context) error {
	w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)
	r := ctx.Value("http.Request").(*http.Request)

	// 这里可以解析请求体
	_ = r.Body

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id":"123","name":"新用户"}`)
	return nil
}

// getPostHandler 获取用户的文章
func getPostHandler(ctx context.Context) error {
	w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)
	userID, _ := zallocrout.GetParam(ctx, "userId")
	postID, _ := zallocrout.GetParam(ctx, "postId")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"user_id":"%s","post_id":"%s","title":"文章 %s"}`, userID, postID, postID)
	return nil
}

// getFileHandler 获取文件（通配符路由）
func getFileHandler(ctx context.Context) error {
	w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)
	path, _ := zallocrout.GetParam(ctx, "*")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "文件路径: %s", path)
	return nil
}

// adminHandler 管理员接口
func adminHandler(ctx context.Context) error {
	w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message":"管理员列表","users":[]}`)
	return nil
}

// metricsHandler 性能指标
func metricsHandler(router *zallocrout.Router) zallocrout.HandlerFunc {
	return func(ctx context.Context) error {
		w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)
		metrics := router.Metrics()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "# zallocrout 指标\n\n")
		fmt.Fprintf(w, "总匹配: %d\n", metrics.TotalMatches)
		fmt.Fprintf(w, "缓存命中: %d\n", metrics.CacheHits)
		fmt.Fprintf(w, "缓存未命中: %d\n", metrics.CacheMisses)
		fmt.Fprintf(w, "命中率: %.2f%%\n\n", router.CacheHitRate()*100)
		fmt.Fprintf(w, "静态路由: %d\n", metrics.StaticRoutes)
		fmt.Fprintf(w, "参数路由: %d\n", metrics.ParamRoutes)
		fmt.Fprintf(w, "通配符路由: %d\n", metrics.WildcardRoutes)

		total, dist := router.CacheStats()
		fmt.Fprintf(w, "\n缓存条目: %d\n", total)
		fmt.Fprintf(w, "分片分布: %v\n", dist)
		return nil
	}
}

// 中间件示例

// loggingMiddleware 日志中间件
func loggingMiddleware(next zallocrout.HandlerFunc) zallocrout.HandlerFunc {
	return func(ctx context.Context) error {
		r := ctx.Value("http.Request").(*http.Request)
		start := time.Now()
		log.Printf("[%s] %s %s", time.Now().Format("15:04:05"), r.Method, r.URL.Path)
		err := next(ctx)
		log.Printf("  耗时: %v", time.Since(start))
		return err
	}
}

// authMiddleware 认证中间件
func authMiddleware(next zallocrout.HandlerFunc) zallocrout.HandlerFunc {
	return func(ctx context.Context) error {
		r := ctx.Value("http.Request").(*http.Request)
		w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)

		// 检查 Authorization header
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"error":"未授权"}`)
			return nil
		}

		return next(ctx)
	}
}

// corsMiddleware CORS 中间件
func corsMiddleware(next zallocrout.HandlerFunc) zallocrout.HandlerFunc {
	return func(ctx context.Context) error {
		w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		return next(ctx)
	}
}

func main() {
	// 创建路由器
	router := zallocrout.NewRouter()

	// 注册路由
	router.AddRoute("GET", "/", homeHandler, loggingMiddleware)
	router.AddRoute("GET", "/health", healthHandler, loggingMiddleware)
	router.AddRoute("GET", "/users", listUsersHandler, loggingMiddleware)
	router.AddRoute("GET", "/users/:id", getUserHandler, loggingMiddleware)
	router.AddRoute("POST", "/users", createUserHandler, loggingMiddleware, authMiddleware)
	router.AddRoute("GET", "/users/:userId/posts/:postId", getPostHandler, loggingMiddleware)
	router.AddRoute("GET", "/files/*path", getFileHandler, loggingMiddleware)
	router.AddRoute("GET", "/admin/users", adminHandler, loggingMiddleware, authMiddleware)
	router.AddRoute("GET", "/metrics", metricsHandler(router), loggingMiddleware)

	// 创建 HTTP 适配器
	adapter := NewHTTPAdapter(router)

	// 启动服务器
	log.Println("服务器启动在 http://localhost:8080")
	log.Println("\n测试命令:")
	log.Println("  curl http://localhost:8080/")
	log.Println("  curl http://localhost:8080/health")
	log.Println("  curl http://localhost:8080/users")
	log.Println("  curl http://localhost:8080/users/123")
	log.Println("  curl -X POST http://localhost:8080/users -H 'Authorization: Bearer token'")
	log.Println("  curl http://localhost:8080/users/123/posts/456")
	log.Println("  curl http://localhost:8080/files/docs/readme.md")
	log.Println("  curl -H 'Authorization: Bearer token' http://localhost:8080/admin/users")
	log.Println("  curl http://localhost:8080/metrics")

	if err := http.ListenAndServe(":8080", adapter); err != nil {
		log.Fatal(err)
	}
}
