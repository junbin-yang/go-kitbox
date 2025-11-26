package taskpool

import (
	"sync/atomic"
	"time"
)

// Metrics 指标统计
type Metrics struct {
	totalSubmitted atomic.Int64 // 总提交数
	totalCompleted atomic.Int64 // 总完成数
	totalFailed    atomic.Int64 // 总失败数
	totalTimeout   atomic.Int64 // 总超时数
	totalPanic     atomic.Int64 // 总panic数
	totalWaitTime  atomic.Int64 // 总等待时间（纳秒）
	totalExecTime  atomic.Int64 // 总执行时间（纳秒）
}

// newMetrics 创建指标统计
func newMetrics() *Metrics {
	return &Metrics{}
}

// recordSubmit 记录任务提交
func (m *Metrics) recordSubmit() {
	m.totalSubmitted.Add(1)
}

// recordTaskComplete 记录任务完成
func (m *Metrics) recordTaskComplete(result *TaskResult) {
	m.totalCompleted.Add(1)

	if result.Err != nil {
		m.totalFailed.Add(1)
		if result.Err == ErrTimeout {
			m.totalTimeout.Add(1)
		}
	}

	if result.Panic != nil {
		m.totalPanic.Add(1)
	}

	m.totalExecTime.Add(result.Duration.Nanoseconds())
}

// recordWaitTime 记录等待时间
func (m *Metrics) recordWaitTime(waitTime time.Duration) {
	m.totalWaitTime.Add(waitTime.Nanoseconds())
}

// snapshot 生成快照
func (m *Metrics) snapshot(queueLen, runningTasks, activeWorkers int) *MetricsSnapshot {
	submitted := m.totalSubmitted.Load()
	completed := m.totalCompleted.Load()
	failed := m.totalFailed.Load()

	var successRate float64
	if completed > 0 {
		successRate = float64(completed-failed) / float64(completed) * 100
	}

	var avgWaitTime time.Duration
	if completed > 0 {
		avgWaitTime = time.Duration(m.totalWaitTime.Load() / completed)
	}

	var avgExecTime time.Duration
	if completed > 0 {
		avgExecTime = time.Duration(m.totalExecTime.Load() / completed)
	}

	return &MetricsSnapshot{
		TotalSubmitted: submitted,
		TotalCompleted: completed,
		TotalFailed:    failed,
		TotalTimeout:   m.totalTimeout.Load(),
		TotalPanic:     m.totalPanic.Load(),
		SuccessRate:    successRate,
		AvgWaitTime:    avgWaitTime,
		AvgExecTime:    avgExecTime,
		CurrentQueue:   queueLen,
		RunningTasks:   runningTasks,
		ActiveWorkers:  activeWorkers,
	}
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	TotalSubmitted int64         // 总提交数
	TotalCompleted int64         // 总完成数
	TotalFailed    int64         // 总失败数
	TotalTimeout   int64         // 总超时数
	TotalPanic     int64         // 总panic数
	SuccessRate    float64       // 成功率（%）
	AvgWaitTime    time.Duration // 平均等待时间
	AvgExecTime    time.Duration // 平均执行时间
	CurrentQueue   int           // 当前队列长度
	RunningTasks   int           // 运行中任务数
	ActiveWorkers  int           // 活跃工作协程数
}
