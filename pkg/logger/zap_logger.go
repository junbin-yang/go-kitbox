package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger 基于zap的日志实现
type ZapLogger struct {
	l  *zap.Logger
	al *zap.AtomicLevel
}

type Logger = ZapLogger

func New(out io.Writer, level Level, opts ...Option) *ZapLogger {
	if out == nil {
		out = os.Stderr
	}

	al := zap.NewAtomicLevelAt(toZapLevel(level))

	core := zapcore.NewCore(
		GetEncoder(),
		zapcore.AddSync(out),
		al,
	)
	return &ZapLogger{l: zap.New(core, opts...), al: &al}
}

// GetEncoder 自定义Encoder
func GetEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(
		zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller_line",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    cEncodeLevel,
			EncodeTime:     cEncodeTime,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   cEncodeCaller,
		})
}

// 自定义日志级别显示
func cEncodeLevel(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

const defaultTimeFormat = "2006-01-02 15:04:05"

// 自定义时间格式显示
func cEncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + t.Format(defaultTimeFormat) + "]")
}

// 自定义行号显示
func cEncodeCaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + caller.TrimmedPath() + "]")
}

func (l *ZapLogger) SetLevel(level Level) {
	if l.al != nil {
		l.al.SetLevel(toZapLevel(level))
	}
}

func (l *ZapLogger) Debug(msg string, fields ...Field) {
	l.l.Debug(msg, fields...)
}

func (l *ZapLogger) Info(msg string, fields ...Field) {
	l.l.Info(msg, fields...)
}

func (l *ZapLogger) Warn(msg string, fields ...Field) {
	l.l.Warn(msg, fields...)
}

func (l *ZapLogger) Error(msg string, fields ...Field) {
	l.l.Error(msg, fields...)
}

func (l *ZapLogger) Panic(msg string, fields ...Field) {
	l.l.Panic(msg, fields...)
}

func (l *ZapLogger) Fatal(msg string, fields ...Field) {
	l.l.Fatal(msg, fields...)
}

func (l *ZapLogger) Sync() error {
	return l.l.Sync()
}

func (l *ZapLogger) Debugf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.Debug(msg)
}

func (l *ZapLogger) Infof(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.Info(msg)
}

func (l *ZapLogger) Warnf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.Warn(msg)
}

func (l *ZapLogger) Errorf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.Error(msg)
}

func (l *ZapLogger) Panicf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.Panic(msg)
}

func (l *ZapLogger) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.Fatal(msg)
}
