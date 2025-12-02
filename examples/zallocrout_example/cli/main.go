package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/junbin-yang/go-kitbox/pkg/zallocrout"
)

// CLIAdapter CLI 适配器
type CLIAdapter struct {
	router *zallocrout.Router
}

// NewCLIAdapter 创建 CLI 适配器
func NewCLIAdapter(router *zallocrout.Router) *CLIAdapter {
	return &CLIAdapter{router: router}
}

// Execute 执行 CLI 命令
func (a *CLIAdapter) Execute(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	// 将 CLI 命令转换为路由路径
	// 例如: ["user", "get", "123"] -> "/user/get/123"
	path := "/" + strings.Join(args, "/")

	// 创建 context 并匹配路由
	ctx, handler, middlewares, ok := a.router.Match("CLI", path, context.Background())
	if !ok {
		return fmt.Errorf("unknown command: %s", strings.Join(args, " "))
	}

	// 设置 CLI 参数到 context
	zallocrout.SetValue(ctx, "cli.args", args)
	zallocrout.SetValue(ctx, "cli.stdout", os.Stdout)
	zallocrout.SetValue(ctx, "cli.stderr", os.Stderr)

	// 执行处理器（自动释放 context）
	return zallocrout.ExecuteHandler(ctx, handler, middlewares)
}

// CLI 命令处理器示例

// userGetCommand 获取用户信息
func userGetCommand(ctx context.Context) error {
	userID, ok := zallocrout.GetParam(ctx, "id")
	if !ok {
		return fmt.Errorf("user ID is required")
	}

	stdout := ctx.Value("cli.stdout").(*os.File)

	fmt.Fprintf(stdout, "User Information:\n")
	fmt.Fprintf(stdout, "  ID: %s\n", userID)
	fmt.Fprintf(stdout, "  Name: User %s\n", userID)
	fmt.Fprintf(stdout, "  Status: Active\n")

	return nil
}

// userListCommand 列出所有用户
func userListCommand(ctx context.Context) error {
	stdout := ctx.Value("cli.stdout").(*os.File)

	fmt.Fprintf(stdout, "User List:\n")
	fmt.Fprintf(stdout, "  1. Alice (alice@example.com)\n")
	fmt.Fprintf(stdout, "  2. Bob (bob@example.com)\n")
	fmt.Fprintf(stdout, "  3. Charlie (charlie@example.com)\n")

	return nil
}

// userCreateCommand 创建用户
func userCreateCommand(ctx context.Context) error {
	name, ok := zallocrout.GetParam(ctx, "name")
	if !ok {
		return fmt.Errorf("user name is required")
	}

	stdout := ctx.Value("cli.stdout").(*os.File)

	fmt.Fprintf(stdout, "Creating user: %s\n", name)
	fmt.Fprintf(stdout, "User created successfully!\n")
	fmt.Fprintf(stdout, "  ID: 123\n")
	fmt.Fprintf(stdout, "  Name: %s\n", name)

	return nil
}

// configGetCommand 获取配置
func configGetCommand(ctx context.Context) error {
	key, ok := zallocrout.GetParam(ctx, "key")
	if !ok {
		return fmt.Errorf("config key is required")
	}

	stdout := ctx.Value("cli.stdout").(*os.File)

	// 模拟配置值
	configs := map[string]string{
		"host": "localhost",
		"port": "8080",
		"env":  "production",
	}

	if value, exists := configs[key]; exists {
		fmt.Fprintf(stdout, "%s = %s\n", key, value)
	} else {
		fmt.Fprintf(stdout, "Config key '%s' not found\n", key)
	}

	return nil
}

// helpCommand 显示帮助信息
func helpCommand(ctx context.Context) error {
	stdout := ctx.Value("cli.stdout").(*os.File)

	fmt.Fprintf(stdout, "Available Commands:\n")
	fmt.Fprintf(stdout, "  user list              - List all users\n")
	fmt.Fprintf(stdout, "  user get <id>          - Get user by ID\n")
	fmt.Fprintf(stdout, "  user create <name>     - Create a new user\n")
	fmt.Fprintf(stdout, "  config get <key>       - Get configuration value\n")
	fmt.Fprintf(stdout, "  help                   - Show this help message\n")

	return nil
}

// CLI 中间件示例

// loggingMiddleware 日志中间件
func loggingMiddleware(next zallocrout.HandlerFunc) zallocrout.HandlerFunc {
	return func(ctx context.Context) error {
		args := ctx.Value("cli.args").([]string)
		log.Printf("[CLI] Command: %s", strings.Join(args, " "))
		return next(ctx)
	}
}

// validationMiddleware 验证中间件
func validationMiddleware(next zallocrout.HandlerFunc) zallocrout.HandlerFunc {
	return func(ctx context.Context) error {
		args := ctx.Value("cli.args").([]string)

		if len(args) == 0 {
			return fmt.Errorf("no command specified")
		}

		return next(ctx)
	}
}

func main() {
	// 解析命令行参数
	flag.Parse()
	args := flag.Args()

	// 创建路由器
	router := zallocrout.NewRouter()

	// 注册 CLI 命令（映射到路由）
	router.AddRoute("CLI", "/user/list", userListCommand, loggingMiddleware, validationMiddleware)
	router.AddRoute("CLI", "/user/get/:id", userGetCommand, loggingMiddleware, validationMiddleware)
	router.AddRoute("CLI", "/user/create/:name", userCreateCommand, loggingMiddleware, validationMiddleware)
	router.AddRoute("CLI", "/config/get/:key", configGetCommand, loggingMiddleware, validationMiddleware)
	router.AddRoute("CLI", "/help", helpCommand, loggingMiddleware)

	// 创建 CLI 适配器
	adapter := NewCLIAdapter(router)

	// 如果没有参数，显示帮助
	if len(args) == 0 {
		args = []string{"help"}
	}

	// 执行命令
	if err := adapter.Execute(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
