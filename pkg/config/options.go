package config

import "time"

// Option 配置管理器选项
type Option func(*ConfigManager)

// WithAppName 设置应用名称（用于默认配置文件名）
func WithAppName(name string) Option {
	return func(cm *ConfigManager) {
		cm.appName = name
	}
}

// WithSerializer 设置默认序列化器
func WithSerializer(s Serializer) Option {
	return func(cm *ConfigManager) {
		cm.serializer = s
	}
}

// WithForceFormat 强制指定配置格式（无视文件后缀）
func WithForceFormat(s Serializer) Option {
	return func(cm *ConfigManager) {
		cm.forceFormat = s
	}
}

// WithDefaultPaths 设置默认配置文件查找路径
func WithDefaultPaths(paths ...string) Option {
	return func(cm *ConfigManager) {
		cm.defaultPaths = paths
	}
}

// WithConfigFormats 设置支持的配置格式列表
func WithConfigFormats(formats ...Serializer) Option {
	return func(cm *ConfigManager) {
		cm.supportedFormats = formats
	}
}

// WithConfigWatch 启用配置文件监听（文件变化自动重载）
func WithConfigWatch(enable bool, interval time.Duration) Option {
	return func(cm *ConfigManager) {
		cm.enableWatch = enable
		cm.watchDebounceInterval = interval
		if interval == 0 {
			cm.watchDebounceInterval = 500 * time.Millisecond
		}
	}
}
