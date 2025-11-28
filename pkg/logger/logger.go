package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Level int8

const (
	DebugLevel Level = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	PanicLevel
	FatalLevel
)

type Field = zap.Field

// toZapLevel 转换为zap级别
func toZapLevel(level Level) zapcore.Level {
	// 映射到zap级别，跳过DPanicLevel
	if level >= PanicLevel {
		return zapcore.Level(level + 1)
	}
	return zapcore.Level(level)
}

var std Interface = New(os.Stderr, InfoLevel, AddCaller(), AddCallerSkip(2))

func Default() Interface         { return std }
func ReplaceDefault(l Interface) { std = l }

func SetLevel(level Level) { std.SetLevel(level) }

func Debug(msg string, fields ...Field) { std.Debug(msg, fields...) }
func Info(msg string, fields ...Field)  { std.Info(msg, fields...) }
func Warn(msg string, fields ...Field)  { std.Warn(msg, fields...) }
func Error(msg string, fields ...Field) { std.Error(msg, fields...) }
func Panic(msg string, fields ...Field) { std.Panic(msg, fields...) }
func Fatal(msg string, fields ...Field) { std.Fatal(msg, fields...) }

func Debugf(format string, v ...interface{}) { std.Debugf(format, v...) }
func Infof(format string, v ...interface{})  { std.Infof(format, v...) }
func Warnf(format string, v ...interface{})  { std.Warnf(format, v...) }
func Errorf(format string, v ...interface{}) { std.Errorf(format, v...) }
func Panicf(format string, v ...interface{}) { std.Panicf(format, v...) }
func Fatalf(format string, v ...interface{}) { std.Fatalf(format, v...) }

func Sync() error { return std.Sync() }

func GetError(e error) Field {
	return zap.Error(e)
}
