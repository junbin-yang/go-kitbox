package logger

import (
	"testing"
	"time"
)

func Test_LOG(t *testing.T) {
	defer func() { _ = Sync() }()
	Info("Info msg")
	Warn("Warn msg")
	Error("Error msg")
	Debug("Debug msg", Int("age", 3))
}

// CustomLogger 自定义日志实现示例
type CustomLogger struct{}

func (c *CustomLogger) Debug(msg string, fields ...Field)      {}
func (c *CustomLogger) Info(msg string, fields ...Field)       {}
func (c *CustomLogger) Warn(msg string, fields ...Field)       {}
func (c *CustomLogger) Error(msg string, fields ...Field)      {}
func (c *CustomLogger) Panic(msg string, fields ...Field)      {}
func (c *CustomLogger) Fatal(msg string, fields ...Field)      {}
func (c *CustomLogger) Debugf(format string, v ...interface{}) {}
func (c *CustomLogger) Infof(format string, v ...interface{})  {}
func (c *CustomLogger) Warnf(format string, v ...interface{})  {}
func (c *CustomLogger) Errorf(format string, v ...interface{}) {}
func (c *CustomLogger) Panicf(format string, v ...interface{}) {}
func (c *CustomLogger) Fatalf(format string, v ...interface{}) {}
func (c *CustomLogger) SetLevel(level Level)                   {}
func (c *CustomLogger) Sync() error                            { return nil }

func Test_CustomLogger(t *testing.T) {
	// 替换为自定义日志实现
	custom := &CustomLogger{}
	ReplaceDefault(custom)

	// 验证可以正常调用
	Info("test custom logger")
	Debugf("test %s", "custom logger")

	// 恢复默认实现
	ReplaceDefault(New(nil, InfoLevel, AddCaller(), AddCallerSkip(2)))
}

func Test_LevelMapping(t *testing.T) {
	// 验证级别映射正确
	if toZapLevel(DebugLevel) != -1 {
		t.Errorf("DebugLevel mapping failed: got %d, want -1", toZapLevel(DebugLevel))
	}
	if toZapLevel(InfoLevel) != 0 {
		t.Errorf("InfoLevel mapping failed: got %d, want 0", toZapLevel(InfoLevel))
	}
	if toZapLevel(WarnLevel) != 1 {
		t.Errorf("WarnLevel mapping failed: got %d, want 1", toZapLevel(WarnLevel))
	}
	if toZapLevel(ErrorLevel) != 2 {
		t.Errorf("ErrorLevel mapping failed: got %d, want 2", toZapLevel(ErrorLevel))
	}
	if toZapLevel(PanicLevel) != 4 {
		t.Errorf("PanicLevel mapping failed: got %d, want 4 (skip DPanic=3)", toZapLevel(PanicLevel))
	}
	if toZapLevel(FatalLevel) != 5 {
		t.Errorf("FatalLevel mapping failed: got %d, want 5", toZapLevel(FatalLevel))
	}
}

func Test_FormattedLogging(t *testing.T) {
	defer func() { _ = Sync() }()
	Infof("Info msg: %s", "test")
	Warnf("Warn msg: %d", 123)
	Errorf("Error msg: %v", true)
}

func Test_SetLevel(t *testing.T) {
	logger := New(nil, InfoLevel)
	logger.SetLevel(DebugLevel)
	logger.Debug("debug message")
	SetLevel(WarnLevel)
}

func Test_Default(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Error("Default logger should not be nil")
	}
}

func Test_RotateByTime(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &RotateConfig{
		Filename:     tmpDir + "/test.log",
		MaxAge:       7,
		RotationTime: 24 * time.Hour,
		LocalTime:    true,
	}
	writer := NewRotateByTime(cfg)
	if writer == nil {
		t.Error("NewRotateByTime should not return nil")
	}
}

func Test_RotateBySize(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &RotateConfig{
		Filename:   tmpDir + "/test.log",
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 5,
		Compress:   true,
		LocalTime:  true,
	}
	writer := NewRotateBySize(cfg)
	if writer == nil {
		t.Error("NewRotateBySize should not return nil")
	}
}

func Test_ProductionRotate(t *testing.T) {
	tmpDir := t.TempDir()

	writer1 := NewProductionRotateByTime(tmpDir + "/time.log")
	if writer1 == nil {
		t.Error("NewProductionRotateByTime should not return nil")
	}

	writer2 := NewProductionRotateBySize(tmpDir + "/size.log")
	if writer2 == nil {
		t.Error("NewProductionRotateBySize should not return nil")
	}

	cfg := NewProductionRotateConfig(tmpDir + "/prod.log")
	if cfg == nil {
		t.Error("NewProductionRotateConfig should not return nil")
	}
	if cfg.MaxAge != 30 {
		t.Errorf("Expected MaxAge=30, got %d", cfg.MaxAge)
	}
}
