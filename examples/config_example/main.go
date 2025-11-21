package main

import (
	"log"

	"github.com/junbin-yang/go-kitbox/pkg/config"
)

// AppConfig 应用配置结构体
type AppConfig struct {
	Server struct {
		Port    int    `yaml:"port" json:"port"`
		Host    string `yaml:"host" json:"host"`
		Timeout int    `yaml:"timeout" json:"timeout"`
	} `yaml:"server" json:"server"`
	Logger struct {
		Level  string `yaml:"level" json:"level"`
		Path   string `yaml:"path" json:"path"`
		Rotate bool   `yaml:"rotate" json:"rotate"`
	} `yaml:"logger" json:"logger"`
	Database struct {
		DSN string `yaml:"dsn" json:"dsn"`
	} `yaml:"database" json:"database"`
}

func main() {
	// 创建配置实例
	cfg := &AppConfig{}

	// 创建配置管理器
	cm := config.NewConfigManager(
		cfg,
		config.WithAppName("app"),
	)

	// 加载配置
	if err := cm.LoadConfig(""); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 获取配置
	configData, err := cm.GetConfig()
	if err != nil {
		log.Fatalf("获取配置失败: %v", err)
	}

	// 使用配置
	appConfig := configData.(*AppConfig)
	log.Printf("服务器端口: %d", appConfig.Server.Port)
	log.Printf("服务器地址: %s", appConfig.Server.Host)
	log.Printf("日志级别: %s", appConfig.Logger.Level)
	log.Printf("数据库DSN: %s", appConfig.Database.DSN)
}
