package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/zallocrout"
)

func main() {
	router := zallocrout.NewRouter()

	// 注册路由
	router.AddRoute("GET", "/", homeHandler)
	router.AddRoute("GET", "/health", healthHandler)
	router.AddRoute("GET", "/users/:id", getUserHandler)
	router.AddRoute("GET", "/users/:userId/posts/:postId", getPostHandler)
	router.AddRoute("GET", "/files/*path", fileHandler)
	router.AddRoute("GET", "/admin/users", adminHandler, authMiddleware, loggingMiddleware)
	router.AddRoute("GET", "/metrics", metricsHandler(router))

	fmt.Println("服务器启动在 http://localhost:8080")
	fmt.Println("\n测试命令:")
	fmt.Println("  curl http://localhost:8080/")
	fmt.Println("  curl http://localhost:8080/health")
	fmt.Println("  curl http://localhost:8080/users/123")
	fmt.Println("  curl http://localhost:8080/users/123/posts/456")
	fmt.Println("  curl http://localhost:8080/files/docs/readme.md")
	fmt.Println("  curl -H \"Authorization: token\" http://localhost:8080/admin/users")
	fmt.Println("  curl http://localhost:8080/metrics\n")

	http.ListenAndServe(":8080", &routerHandler{router: router})
}

type routerHandler struct {
	router *zallocrout.Router
}

func (h *routerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	result, ok := h.router.Match(r.Method, r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	defer result.Release()

	// 将 result 存入 context
	ctx := context.WithValue(r.Context(), resultKey, &result)
	r = r.WithContext(ctx)

	// 执行中间件链
	handler := result.Handler
	for i := len(result.Middlewares) - 1; i >= 0; i-- {
		handler = result.Middlewares[i](handler)
	}
	handler(w, r, nil)
}

type ctxKey string

const resultKey ctxKey = "result"

func getResult(r *http.Request) *zallocrout.MatchResult {
	if v := r.Context().Value(resultKey); v != nil {
		return v.(*zallocrout.MatchResult)
	}
	return nil
}

func homeHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "欢迎使用 zallocrout 路由器\n\n")
	fmt.Fprintf(w, "特性:\n")
	fmt.Fprintf(w, "  - 零内存分配 (0 allocs/op)\n")
	fmt.Fprintf(w, "  - 亚微秒延迟 (P99 < 30ns)\n")
	fmt.Fprintf(w, "  - 高吞吐量 (300万+ QPS)\n")
}

func healthHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","timestamp":%d}`, time.Now().Unix())
}

func getUserHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	result := getResult(r)
	userID, _ := result.GetParam("id")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"user_id":"%s","name":"用户%s"}`, userID, userID)
}

func getPostHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	result := getResult(r)
	userID, _ := result.GetParam("userId")
	postID, _ := result.GetParam("postId")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"user_id":"%s","post_id":"%s","title":"文章%s"}`, userID, postID, postID)
}

func fileHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	result := getResult(r)
	path, _ := result.GetParam("*")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "文件路径: %s\n", path)
}

func adminHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message":"管理员列表","users":[]}`)
}

func authMiddleware(next zallocrout.HandlerFunc) zallocrout.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"error":"未授权"}`)
			return
		}
		next(w, r, params)
	}
}

func loggingMiddleware(next zallocrout.HandlerFunc) zallocrout.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		start := time.Now()
		fmt.Printf("[%s] %s %s\n", time.Now().Format("15:04:05"), r.Method, r.URL.Path)
		next(w, r, params)
		fmt.Printf("  耗时: %v\n", time.Since(start))
	}
}

func metricsHandler(router *zallocrout.Router) zallocrout.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
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
	}
}
