package taskpool

// ScaleStrategy 扩缩容策略接口
type ScaleStrategy interface {
	// ShouldScaleUp 是否应该扩容
	ShouldScaleUp(queueLen, runningTasks, currentWorkers, maxWorkers int) bool

	// ShouldScaleDown 是否应该缩容
	ShouldScaleDown(queueLen, runningTasks, currentWorkers, minWorkers int) bool

	// ScaleUpCount 扩容数量
	ScaleUpCount(current, max int) int

	// ScaleDownCount 缩容数量
	ScaleDownCount(current, min int) int
}

// DefaultScaleStrategy 默认扩缩容策略
type DefaultScaleStrategy struct {
	ScaleUpThreshold   float64 // 队列使用率阈值（默认0.8）
	ScaleDownThreshold float64 // 队列使用率阈值（默认0.2）
	ScaleUpStep        int     // 每次扩容数量（默认2）
	ScaleDownStep      int     // 每次缩容数量（默认1）
	QueueCapacity      int     // 队列容量
}

// NewDefaultScaleStrategy 创建默认扩缩容策略
func NewDefaultScaleStrategy(queueCapacity int) *DefaultScaleStrategy {
	return &DefaultScaleStrategy{
		ScaleUpThreshold:   0.8,
		ScaleDownThreshold: 0.2,
		ScaleUpStep:        2,
		ScaleDownStep:      1,
		QueueCapacity:      queueCapacity,
	}
}

// ShouldScaleUp 是否应该扩容
func (s *DefaultScaleStrategy) ShouldScaleUp(queueLen, runningTasks, currentWorkers, maxWorkers int) bool {
	if currentWorkers >= maxWorkers {
		return false
	}

	queueUsage := float64(queueLen) / float64(s.QueueCapacity)
	return queueUsage > s.ScaleUpThreshold
}

// ShouldScaleDown 是否应该缩容
func (s *DefaultScaleStrategy) ShouldScaleDown(queueLen, runningTasks, currentWorkers, minWorkers int) bool {
	if currentWorkers <= minWorkers {
		return false
	}

	queueUsage := float64(queueLen) / float64(s.QueueCapacity)
	taskUsage := float64(runningTasks) / float64(currentWorkers)

	return queueUsage < s.ScaleDownThreshold && taskUsage < 0.5
}

// ScaleUpCount 扩容数量
func (s *DefaultScaleStrategy) ScaleUpCount(current, max int) int {
	count := s.ScaleUpStep
	if current+count > max {
		count = max - current
	}
	return count
}

// ScaleDownCount 缩容数量
func (s *DefaultScaleStrategy) ScaleDownCount(current, min int) int {
	count := s.ScaleDownStep
	if current-count < min {
		count = current - min
	}
	return count
}
