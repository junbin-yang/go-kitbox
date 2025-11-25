package lifecycle

import "fmt"

var (
	// ErrWorkerExists 当协程已存在时返回
	ErrWorkerExists = fmt.Errorf("worker already exists")

	// ErrWorkerNotFound 当协程不存在时返回
	ErrWorkerNotFound = fmt.Errorf("worker not found")

	// ErrShutdownTimeout 当退出超时时返回
	ErrShutdownTimeout = fmt.Errorf("shutdown timeout")

	// ErrAlreadyRunning 当管理器已在运行时返回
	ErrAlreadyRunning = fmt.Errorf("manager already running")
)
