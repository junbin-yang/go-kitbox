package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/junbin-yang/go-kitbox/examples/app_framework/internal/config"
	"github.com/junbin-yang/go-kitbox/examples/app_framework/internal/handler"
	"github.com/junbin-yang/go-kitbox/examples/app_framework/internal/service"
	pkgConfig "github.com/junbin-yang/go-kitbox/pkg/config"
	"github.com/junbin-yang/go-kitbox/pkg/lifecycle"
	"github.com/junbin-yang/go-kitbox/pkg/logger"
)

func main() {
	// 1. 加载配置
	cfg := &config.Config{}
	configPath := getConfigPath()

	mgr := pkgConfig.NewConfigManager(cfg)
	if err := mgr.LoadConfig(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	log := logger.New(os.Stdout, logger.InfoLevel, zap.AddCaller())

	log.Info("应用启动",
		logger.String("name", cfg.Server.Name),
		logger.String("version", cfg.Server.Version),
	)

	// 3. 创建生命周期管理器
	lm := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(cfg.Server.ShutdownTimeout),
	)

	// 4. 初始化服务
	demoService := service.NewDemoService(log)

	// 5. 初始化处理器
	healthHandler := handler.NewHealthHandler()
	apiHandler := handler.NewAPIHandler(demoService, log)

	// 6. 创建HTTP路由
	mux := http.NewServeMux()
	mux.Handle("/health", healthHandler)
	mux.Handle("/api/demo", apiHandler)

	// 7. 创建HTTP服务器
	server := &http.Server{
		Addr:         cfg.Server.HTTP.Addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.HTTP.ReadTimeout,
		WriteTimeout: cfg.Server.HTTP.WriteTimeout,
		IdleTimeout:  cfg.Server.HTTP.IdleTimeout,
	}

	// 8. 注册HTTP服务器到生命周期管理器
	lm.AddWorker("http-server",
		func(ctx context.Context) error {
			log.Info("HTTP服务器启动", logger.String("addr", cfg.Server.HTTP.Addr))
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				return err
			}
			return nil
		},
		lifecycle.WithStopFunc(func(ctx context.Context) error {
			log.Info("正在关闭HTTP服务器...")
			return server.Shutdown(ctx)
		}),
	)

	// 9. 注册生命周期钩子
	lm.OnStartup(func(ctx context.Context) error {
		log.Info("应用启动完成")
		return nil
	})

	lm.OnShutdown(func(ctx context.Context) error {
		log.Info("正在清理资源...")
		log.Sync()
		return nil
	})

	lm.OnWorkerExit(func(name string, err error) {
		if err != nil {
			log.Error("协程异常退出",
				logger.String("worker", name),
				logger.String("error", err.Error()),
			)
		} else {
			log.Info("协程正常退出", logger.String("worker", name))
		}
	})

	// 10. 启动应用
	if err := lm.Run(); err != nil {
		log.Error("应用运行错误", logger.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("应用已退出")
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		return configPath
	}
	// 默认使用当前目录的configs/config.yaml
	return filepath.Join("configs", "config.yaml")
}
