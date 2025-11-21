package main

import (
	"time"

	log "github.com/junbin-yang/go-kitbox/pkg/logger"
)

func main() {
	defer log.Sync()

	// 示例 1: 基础使用
	log.Info("=== 基础日志示例 ===")
	log.Debug("这是 Debug 日志")
	log.Info("这是 Info 日志")
	log.Warn("这是 Warn 日志", log.Int("code", 400))
	log.Error("这是 Error 日志", log.String("error", "something wrong"))

	// 示例 2: 结构化日志
	log.Info("用户登录",
		log.String("username", "alice"),
		log.Int("user_id", 123),
		log.Duration("latency", time.Millisecond*50),
		log.Bool("success", true),
	)

	// 示例 3: 格式化日志
	log.Infof("用户 %s 登录成功，ID: %d", "bob", 456)

	// 示例 4: 动态调整日志级别
	log.Info("修改日志级别为 ErrorLevel")
	log.SetLevel(log.ErrorLevel)
	log.Info("这条 Info 不会输出")
	log.Error("这条 Error 会输出")

	// 示例 5: 使用日志轮转
	log.Info("=== 日志轮转示例 ===")
	out := log.NewProductionRotateBySize("app.log")
	logger := log.New(out, log.InfoLevel, log.AddCaller())
	logger.Info("这条日志会写入 app.log 并按大小轮转")

	// 示例 6: 自定义轮转配置
	cfg := &log.RotateConfig{
		Filename:     "custom.log",
		MaxAge:       7,
		RotationTime: time.Hour * 1,
		LocalTime:    true,
	}
	out2 := log.NewRotateByTime(cfg)
	logger2 := log.New(out2, log.InfoLevel)
	logger2.Info("这条日志会按小时轮转")
}
