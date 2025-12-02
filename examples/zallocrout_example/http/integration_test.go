package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/netconn"
	"github.com/junbin-yang/go-kitbox/pkg/zallocrout"
)

// TestHTTP_Integration_WithNetConn 使用 netconn 包进行真实网络连接的集成测试
// 服务端和客户端都使用 netconn 包实现
func TestHTTP_Integration_WithNetConn(t *testing.T) {
	// 创建路由器
	router := zallocrout.NewRouter()
	router.AddRoute("GET", "/", homeHandler)
	router.AddRoute("GET", "/users/:id", getUserHandler)
	router.AddRoute("GET", "/users/:userId/posts/:postId", getPostHandler)

	// 启动 netconn 服务器
	port := 18080
	server, err := startNetConnHTTPServer(t, router, port)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.StopBaseListener()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 运行测试用例
	t.Run("GET /", func(t *testing.T) {
		testHTTPRequest(t, "localhost", port, "GET", "/", "", 200, "欢迎")
	})

	t.Run("GET /users/123", func(t *testing.T) {
		testHTTPRequest(t, "localhost", port, "GET", "/users/123", "", 200, "123")
	})

	t.Run("GET /users/123/posts/456", func(t *testing.T) {
		testHTTPRequest(t, "localhost", port, "GET", "/users/123/posts/456", "", 200, "456")
	})

	t.Run("GET /nonexistent", func(t *testing.T) {
		testHTTPRequest(t, "localhost", port, "GET", "/nonexistent", "", 404, "404")
	})
}

// responseCapture 实现 http.ResponseWriter 接口，用于捕获handler的响应
type responseCapture struct {
	statusCode int
	header     http.Header
	body       *bytes.Buffer
}

func newResponseCapture() *responseCapture {
	return &responseCapture{
		statusCode: 200,
		header:     make(http.Header),
		body:       &bytes.Buffer{},
	}
}

func (rc *responseCapture) Header() http.Header {
	return rc.header
}

func (rc *responseCapture) Write(data []byte) (int, error) {
	return rc.body.Write(data)
}

func (rc *responseCapture) WriteHeader(statusCode int) {
	rc.statusCode = statusCode
}

// buildHTTPResponseFromCapture 从responseCapture构建HTTP响应
func (rc *responseCapture) buildHTTPResponseFromCapture() []byte {
	var buf bytes.Buffer

	// 状态行
	statusText := http.StatusText(rc.statusCode)
	if statusText == "" {
		statusText = "Unknown"
	}
	buf.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", rc.statusCode, statusText))

	// Headers
	bodyLen := rc.body.Len()
	rc.header.Set("Content-Length", fmt.Sprintf("%d", bodyLen))
	rc.header.Set("Connection", "close")

	for key, values := range rc.header {
		for _, value := range values {
			buf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
		}
	}

	// 空行分隔header和body
	buf.WriteString("\r\n")

	// Body
	buf.Write(rc.body.Bytes())

	return buf.Bytes()
}

// startNetConnHTTPServer 使用 netconn 包启动 HTTP 服务器
func startNetConnHTTPServer(t *testing.T, router *zallocrout.Router, port int) (*netconn.BaseServer, error) {
	server := netconn.NewBaseServer(nil)

	// 服务端接收缓冲区（每个连接）
	clientBuffers := make(map[int]*bytes.Buffer)
	var bufferMu sync.RWMutex

	callback := &netconn.BaseListenerCallback{
		OnConnected: func(fd int, connType netconn.ConnectionType, connOpt *netconn.ConnectOption) {
			if connType == netconn.ConnectionTypeServer {
				t.Logf("[Server] Listening on %s:%d", connOpt.LocalSocket.Addr, connOpt.LocalSocket.Port)
			} else {
				t.Logf("[Server] Client connected, fd=%d", fd)
				// 为新连接创建缓冲区
				bufferMu.Lock()
				clientBuffers[fd] = &bytes.Buffer{}
				bufferMu.Unlock()
			}
		},
		OnDisconnected: func(fd int, connType netconn.ConnectionType) {
			if connType == netconn.ConnectionTypeClient {
				t.Logf("[Server] Client disconnected, fd=%d", fd)
				// 清理缓冲区
				bufferMu.Lock()
				delete(clientBuffers, fd)
				bufferMu.Unlock()
			}
		},
		OnDataReceived: func(fd int, connType netconn.ConnectionType, buf []byte, used int) int {
			// 获取该连接的缓冲区
			bufferMu.RLock()
			buffer, ok := clientBuffers[fd]
			bufferMu.RUnlock()

			if !ok {
				t.Logf("[Server] No buffer for fd=%d", fd)
				return used
			}

			// 将数据添加到缓冲区
			buffer.Write(buf[:used])

			// 尝试解析 HTTP 请求
			data := buffer.Bytes()
			if !strings.Contains(string(data), "\r\n\r\n") {
				// 还没收到完整的请求头，继续等待
				return used
			}

			// 解析 HTTP 请求
			reader := bufio.NewReader(bytes.NewReader(data))
			req, err := http.ReadRequest(reader)
			if err != nil {
				t.Logf("[Server] Failed to parse request: %v", err)
				// 清空缓冲区
				buffer.Reset()
				return used
			}

			t.Logf("[Server] Request: %s %s", req.Method, req.URL.Path)

			// 使用路由器处理请求
			ctx, handler, middlewares, ok := router.Match(req.Method, req.URL.Path, context.Background())
			var response []byte

			if !ok {
				// 404 Not Found
				response = []byte("HTTP/1.1 404 Not Found\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: 15\r\nConnection: close\r\n\r\n404 页面未找到")
			} else {
				// 创建响应捕获器
				respCapture := newResponseCapture()

				// 将请求和响应写入器注入到context
				zallocrout.SetValue(ctx, "http.Request", req)
				zallocrout.SetValue(ctx, "http.ResponseWriter", respCapture)

				// 执行处理器
				err := zallocrout.ExecuteHandler(ctx, handler, middlewares)
				if err != nil {
					// 500 Internal Server Error
					response = []byte("HTTP/1.1 500 Internal Server Error\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: 21\r\nConnection: close\r\n\r\n500 Internal Server Error")
				} else {
					// 从响应捕获器构建HTTP响应
					response = respCapture.buildHTTPResponseFromCapture()
				}
			}

			// 发送响应
			err = server.SendBytes(fd, response)
			if err != nil {
				t.Logf("[Server] Failed to send response: %v", err)
			}

			// 清空缓冲区
			buffer.Reset()

			return used
		},
	}

	opt := &netconn.ServerOption{
		Protocol: netconn.ProtocolTCP,
		Addr:     "0.0.0.0",
		Port:     port,
	}

	if err := server.StartBaseListener(opt, callback); err != nil {
		return nil, err
	}

	return server, nil
}

// testHTTPRequest 使用 netconn 包发送 HTTP 请求并验证响应
func testHTTPRequest(t *testing.T, host string, port int, method, path, body string, expectedStatus int, expectedBody string) {
	// 创建连接管理器
	connMgr := netconn.NewConnectionManager()

	// 接收到的数据
	var receivedData []byte
	var receivedMu sync.Mutex
	var responseChan = make(chan bool, 1)

	// 设置回调
	callback := &netconn.BaseListenerCallback{
		OnConnected: func(fd int, connType netconn.ConnectionType, connOpt *netconn.ConnectOption) {
			t.Logf("Connected to %s:%d", host, port)
		},
		OnDisconnected: func(fd int, connType netconn.ConnectionType) {
			t.Logf("Disconnected from %s:%d", host, port)
		},
		OnDataReceived: func(fd int, connType netconn.ConnectionType, buf []byte, used int) int {
			receivedMu.Lock()
			receivedData = append(receivedData, buf[:used]...)
			receivedMu.Unlock()

			// 检查是否接收到完整的 HTTP 响应
			receivedMu.Lock()
			data := string(receivedData)
			receivedMu.Unlock()

			// 简单判断：如果包含两个连续的 \r\n\r\n，说明至少接收到了 header
			if strings.Contains(data, "\r\n\r\n") {
				// 尝试解析 Content-Length
				if strings.Contains(data, "Content-Length:") {
					lines := strings.Split(data, "\r\n")
					var contentLength int
					for _, line := range lines {
						if strings.HasPrefix(line, "Content-Length:") {
							lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
							contentLength, _ = strconv.Atoi(lengthStr)
							break
						}
					}

					// 检查是否接收到完整的响应体
					parts := strings.SplitN(data, "\r\n\r\n", 2)
					if len(parts) == 2 && len(parts[1]) >= contentLength {
						responseChan <- true
					}
				} else if strings.Contains(data, "Transfer-Encoding: chunked") {
					// Chunked 编码：检查是否以 0\r\n\r\n 结束
					if strings.HasSuffix(data, "0\r\n\r\n") {
						responseChan <- true
					}
				} else {
					// 没有 Content-Length，假设连接关闭时响应完成
					// 或者已经接收到完整响应
					responseChan <- true
				}
			}

			return used
		},
	}

	// 创建客户端
	client := netconn.NewBaseClient(connMgr, callback)

	// 连接到服务器
	clientOpt := &netconn.ClientOption{
		Protocol:   netconn.ProtocolTCP,
		RemoteIP:   host,
		RemotePort: port,
		Timeout:    5 * time.Second,
	}

	_, err := client.Connect(clientOpt)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	defer func() {
		client.Close()
	}()

	// 构建 HTTP 请求
	var reqBuilder strings.Builder
	reqBuilder.WriteString(fmt.Sprintf("%s %s HTTP/1.1\r\n", method, path))
	reqBuilder.WriteString(fmt.Sprintf("Host: %s:%d\r\n", host, port))
	reqBuilder.WriteString("Connection: close\r\n")

	if body != "" {
		reqBuilder.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body)))
		reqBuilder.WriteString(body)
	} else {
		reqBuilder.WriteString("\r\n")
	}

	request := reqBuilder.String()

	// 发送请求
	err = client.SendBytes([]byte(request))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// 等待响应（最多5秒）
	select {
	case <-responseChan:
		// 响应接收完成
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for response")
	}

	// 解析响应
	receivedMu.Lock()
	responseData := receivedData
	receivedMu.Unlock()

	// 解析 HTTP 响应
	reader := bufio.NewReader(bytes.NewReader(responseData))
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	defer resp.Body.Close()

	// 验证状态码
	if resp.StatusCode != expectedStatus {
		t.Errorf("Status code = %d, want %d", resp.StatusCode, expectedStatus)
	}

	// 验证响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	bodyStr := string(bodyBytes)
	if !strings.Contains(bodyStr, expectedBody) {
		t.Errorf("Response body does not contain expected content.\nExpected substring: %s\nActual body: %s", expectedBody, bodyStr)
	}

	t.Logf("Request succeeded: %s %s -> %d", method, path, resp.StatusCode)
}

// TestHTTP_Integration_ConcurrentRequests 测试并发请求
func TestHTTP_Integration_ConcurrentRequests(t *testing.T) {
	// 创建路由器
	router := zallocrout.NewRouter()
	router.AddRoute("GET", "/users/:id", getUserHandler)

	// 启动服务器
	port := 18081
	server, err := startNetConnHTTPServer(t, router, port)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.StopBaseListener()

	time.Sleep(100 * time.Millisecond)

	// 并发请求
	var wg sync.WaitGroup
	concurrency := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			userID := fmt.Sprintf("%d", id)
			testHTTPRequest(t, "localhost", port, "GET", "/users/"+userID, "", 200, userID)
		}(i)
	}

	wg.Wait()
	t.Logf("Successfully completed %d concurrent requests", concurrency)
}

// BenchmarkHTTP_Integration_WithNetConn 性能基准测试
func BenchmarkHTTP_Integration_WithNetConn(b *testing.B) {
	// 创建路由器
	router := zallocrout.NewRouter()
	router.AddRoute("GET", "/users/:id", getUserHandler)

	// 启动服务器
	port := 18082
	server, err := startNetConnHTTPServer(&testing.T{}, router, port)
	if err != nil {
		b.Fatalf("Failed to start server: %v", err)
	}
	defer server.StopBaseListener()

	time.Sleep(100 * time.Millisecond)

	// 创建连接管理器和客户端
	connMgr := netconn.NewConnectionManager()

	var receivedData []byte
	var receivedMu sync.Mutex
	var responseChan = make(chan bool, 1)

	callback := &netconn.BaseListenerCallback{
		OnDataReceived: func(fd int, connType netconn.ConnectionType, buf []byte, used int) int {
			receivedMu.Lock()
			receivedData = append(receivedData, buf[:used]...)
			data := string(receivedData)
			receivedMu.Unlock()

			if strings.Contains(data, "\r\n\r\n") {
				if strings.Contains(data, "Content-Length:") {
					lines := strings.Split(data, "\r\n")
					var contentLength int
					for _, line := range lines {
						if strings.HasPrefix(line, "Content-Length:") {
							lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
							contentLength, _ = strconv.Atoi(lengthStr)
							break
						}
					}
					parts := strings.SplitN(data, "\r\n\r\n", 2)
					if len(parts) == 2 && len(parts[1]) >= contentLength {
						responseChan <- true
					}
				} else {
					responseChan <- true
				}
			}
			return used
		},
	}

	client := netconn.NewBaseClient(connMgr, callback)
	clientOpt := &netconn.ClientOption{
		Protocol:   netconn.ProtocolTCP,
		RemoteIP:   "localhost",
		RemotePort: port,
		Timeout:    5 * time.Second,
	}

	_, err = client.Connect(clientOpt)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	request := "GET /users/123 HTTP/1.1\r\nHost: localhost\r\n\r\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		receivedMu.Lock()
		receivedData = receivedData[:0]
		receivedMu.Unlock()

		err := client.SendBytes([]byte(request))
		if err != nil {
			b.Fatalf("Failed to send request: %v", err)
		}

		select {
		case <-responseChan:
		case <-time.After(1 * time.Second):
			b.Fatal("Timeout")
		}
	}
}
