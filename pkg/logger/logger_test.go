package logger

import "testing"

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
