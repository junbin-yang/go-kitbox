package config

import "time"

// Config 应用配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Logger   LoggerConfig   `yaml:"logger"`
	Business BusinessConfig `yaml:"business"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Name            string         `yaml:"name"`
	Version         string         `yaml:"version"`
	HTTP            HTTPConfig     `yaml:"http"`
	ShutdownTimeout time.Duration  `yaml:"shutdown_timeout"`
}

// HTTPConfig HTTP配置
type HTTPConfig struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level            string `yaml:"level"`
	Output           string `yaml:"output"`
	Encoding         string `yaml:"encoding"`
	EnableCaller     bool   `yaml:"enable_caller"`
	EnableStacktrace bool   `yaml:"enable_stacktrace"`
	MaxSize          int    `yaml:"max_size"`
	MaxBackups       int    `yaml:"max_backups"`
	MaxAge           int    `yaml:"max_age"`
	Compress         bool   `yaml:"compress"`
}

// BusinessConfig 业务配置
type BusinessConfig struct {
	CacheTTL   time.Duration `yaml:"cache_ttl"`
	MaxWorkers int           `yaml:"max_workers"`
}
