package lifecycle

import (
	"context"
	"os"
	"time"
)

// Option 管理器配置选项
type Option func(*Manager)

// WithSignals 设置监听的信号
func WithSignals(signals ...os.Signal) Option {
	return func(m *Manager) {
		m.signals = signals
	}
}

// WithShutdownTimeout 设置退出超时时间
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(m *Manager) {
		m.shutdownTimeout = timeout
	}
}

// WithContext 设置根上下文
func WithContext(ctx context.Context) Option {
	return func(m *Manager) {
		m.rootCtx = ctx
	}
}
