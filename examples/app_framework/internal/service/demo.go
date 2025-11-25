package service

import (
	"context"
	"fmt"
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/logger"
)

// DemoService 示例服务
type DemoService struct {
	logger *logger.Logger
}

// NewDemoService 创建示例服务
func NewDemoService(log *logger.Logger) *DemoService {
	return &DemoService{
		logger: log,
	}
}

// Process 处理业务逻辑
func (s *DemoService) Process(ctx context.Context) string {
	s.logger.Debug("处理业务逻辑")
	return fmt.Sprintf("处理时间: %s", time.Now().Format(time.RFC3339))
}
