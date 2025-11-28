package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ConfigManager 通用配置管理器
type ConfigManager struct {
	instance         interface{}  // 配置实例
	configPath       string       // 配置文件路径
	appName          string       // 应用名称
	serializer       Serializer   // 当前使用的序列化器
	forceFormat      Serializer   // 强制指定的格式（优先级最高）
	supportedFormats []Serializer // 支持的配置格式列表
	defaultPaths     []string     // 默认配置路径模板
	once             sync.Once    // 确保配置只加载一次
	mu               sync.RWMutex // 读写锁
	loadErr          error        // 加载错误

	// 配置监听相关
	enableWatch           bool              // 是否启用配置监听
	watchDebounceInterval time.Duration     // 防抖间隔
	watcher               *fsnotify.Watcher // 文件监听器
	watchQuit             chan struct{}     // 监听退出信号
	watchOnce             sync.Once         // 确保监听只启动一次

	// 配置变更回调
	callbacks []func(old, new interface{})
}

// NewConfigManager 创建配置管理器实例
// cfg: 配置结构体指针（必须传入指针）
// options: 配置选项
func NewConfigManager(cfg interface{}, options ...Option) *ConfigManager {
	if cfg == nil {
		panic("config instance cannot be nil")
	}
	if reflect.ValueOf(cfg).Kind() != reflect.Ptr {
		panic("config instance must be a pointer")
	}

	// 默认配置
	cm := &ConfigManager{
		instance:         cfg,
		appName:          "app",
		serializer:       &YAMLSerializer{},
		supportedFormats: []Serializer{&YAMLSerializer{}, &JSONSerializer{}, &INISerializer{}},
		defaultPaths: []string{
			"./{{.AppName}}",
			"{{.ExecDir}}/{{.AppName}}",
			"/etc/{{.AppName}}",
		},
		watchQuit: make(chan struct{}),
	}

	// 应用自定义选项
	for _, opt := range options {
		opt(cm)
	}

	return cm
}

// LoadConfig 加载配置文件
// customPath: 自定义配置路径，空字符串使用默认路径
func (cm *ConfigManager) LoadConfig(customPath string) error {
	cm.once.Do(func() {
		var err error

		// 1. 处理自定义路径
		if customPath != "" {
			if err = validateConfigPath(customPath); err != nil {
				cm.loadErr = fmt.Errorf("invalid custom config path: %w", err)
				return
			}
			cm.configPath = customPath
			// 选择序列化器（强制格式 > 后缀识别 > 默认）
			if err = cm.chooseSerializer(customPath); err != nil {
				cm.loadErr = fmt.Errorf("choose serializer failed: %w", err)
				return
			}
		} else {
			// 2. 查找默认路径
			if cm.configPath, err = cm.findDefaultConfigPath(); err != nil {
				cm.loadErr = fmt.Errorf("default config not found: %w", err)
				return
			}
		}

		// 3. 解析配置文件
		if err = cm.parseConfigFile(); err != nil {
			cm.loadErr = fmt.Errorf("parse config failed: %w", err)
			return
		}

		// 4. 应用环境变量覆盖
		if err = applyEnvOverrides(cm.instance); err != nil {
			cm.loadErr = fmt.Errorf("apply env overrides failed: %w", err)
			return
		}

		// 5. 启动配置监听（如果启用）
		if cm.enableWatch {
			_ = cm.startWatch()
		}
	})

	return cm.loadErr
}

// GetConfig 获取配置实例
// 返回值: 配置实例, 错误
func (cm *ConfigManager) GetConfig() (interface{}, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.loadErr != nil {
		return nil, cm.loadErr
	}
	if cm.instance == nil {
		return nil, errors.New("config not initialized, call LoadConfig first")
	}
	return cm.instance, nil
}

// SaveConfig 保存配置到文件
func (cm *ConfigManager) SaveConfig() error {
	cm.mu.RLock()
	if cm.instance == nil || cm.configPath == "" {
		cm.mu.RUnlock()
		return errors.New("config not initialized")
	}
	cm.mu.RUnlock()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 序列化配置
	data, err := cm.serializer.Marshal(cm.instance)
	if err != nil {
		return fmt.Errorf("marshal config failed: %w", err)
	}

	// 先写入临时文件（避免文件损坏）
	tmpPath := cm.configPath + ".tmp"
	if err := ioutil.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp config failed: %w", err)
	}

	// 替换原文件
	if err := os.Rename(tmpPath, cm.configPath); err != nil {
		return fmt.Errorf("rename temp config failed: %w", err)
	}

	return nil
}

// ReloadConfig 手动重新加载配置
func (cm *ConfigManager) ReloadConfig() error {
	cm.mu.RLock()
	currentPath := cm.configPath
	cm.mu.RUnlock()

	if currentPath == "" {
		return errors.New("config path not initialized")
	}
	if err := validateConfigPath(currentPath); err != nil {
		return fmt.Errorf("invalid config path: %w", err)
	}

	cm.mu.Lock()

	// 创建新实例避免覆盖原数据
	newInstance := cm.createNewInstance()
	if newInstance == nil {
		cm.mu.Unlock()
		return errors.New("create new config instance failed")
	}

	// 读取并解析配置
	data, err := ioutil.ReadFile(currentPath)
	if err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("read config file failed: %w", err)
	}
	if err := cm.serializer.Unmarshal(data, newInstance); err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("unmarshal config failed: %w", err)
	}

	// 应用环境变量覆盖
	if err := applyEnvOverrides(newInstance); err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("apply env overrides failed: %w", err)
	}

	oldInstance := cm.instance
	cm.instance = newInstance
	cm.loadErr = nil

	// 复制回调列表（避免死锁）
	callbacks := make([]func(old, new interface{}), len(cm.callbacks))
	copy(callbacks, cm.callbacks)
	cm.mu.Unlock()

	// 触发配置变更回调（在锁外执行）
	for _, callback := range callbacks {
		callback(oldInstance, newInstance)
	}

	return nil
}

// EnableWatch 动态启用/禁用配置监听
func (cm *ConfigManager) EnableWatch(enable bool) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.enableWatch = enable
	if enable && cm.configPath != "" {
		return cm.startWatch()
	} else {
		cm.stopWatch()
		return nil
	}
}

// Close 关闭配置管理器（停止监听）
func (cm *ConfigManager) Close() {
	cm.stopWatch()
	close(cm.watchQuit)
}

// OnChange 注册配置变更回调
func (cm *ConfigManager) OnChange(callback func(old, new interface{})) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.callbacks = append(cm.callbacks, callback)
}


/* ------------------------------ 内部方法 ------------------------------ */

// chooseSerializer 选择序列化器
func (cm *ConfigManager) chooseSerializer(path string) error {
	// 强制格式优先级最高
	if cm.forceFormat != nil {
		cm.serializer = cm.forceFormat
		return nil
	}

	// 根据文件后缀选择
	ext := filepath.Ext(path)
	for _, format := range cm.supportedFormats {
		if format.GetFileExt() == ext {
			cm.serializer = format
			return nil
		}
	}

	// 无后缀时使用默认序列化器
	return nil
}

// findDefaultConfigPath 查找默认配置路径
func (cm *ConfigManager) findDefaultConfigPath() (string, error) {
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)

	// 遍历默认路径模板
	for _, pathTpl := range cm.defaultPaths {
		// 替换路径变量
		basePath := replacePathVars(pathTpl, map[string]string{
			"AppName": cm.appName,
			"ExecDir": execDir,
		})

		// 先尝试无后缀文件
		if err := validateConfigPath(basePath); err == nil {
			// 无后缀文件使用默认或强制格式
			_ = cm.chooseSerializer(basePath)
			return basePath, nil
		}

		// 尝试带后缀的文件
		for _, format := range cm.supportedFormats {
			fullPath := basePath + format.GetFileExt()
			if err := validateConfigPath(fullPath); err == nil {
				cm.serializer = format
				return fullPath, nil
			}
		}
	}

	return "", errors.New("no valid config file found (tried default paths and formats)")
}

// startWatch 启动配置文件监听
func (cm *ConfigManager) startWatch() error {
	var err error
	cm.watchOnce.Do(func() {
		if cm.watcher, err = fsnotify.NewWatcher(); err != nil {
			err = fmt.Errorf("create watcher failed: %w", err)
			return
		}

		// 添加配置文件监听
		if err = cm.watcher.Add(cm.configPath); err != nil {
			err = fmt.Errorf("add watch path failed: %w", err)
			return
		}

		// 启动监听协程
		go cm.watchLoop()
	})
	return err
}

// stopWatch 停止配置文件监听
func (cm *ConfigManager) stopWatch() {
	cm.watchOnce = sync.Once{}
	if cm.watcher != nil {
		cm.watcher.Close()
		cm.watcher = nil
	}
}

// watchLoop 监听文件变化循环
func (cm *ConfigManager) watchLoop() {
	debounceTimer := time.NewTimer(0)
	if !debounceTimer.Stop() {
		<-debounceTimer.C
	}

	for {
		select {
		case event, ok := <-func() <-chan fsnotify.Event {
			cm.mu.RLock()
			defer cm.mu.RUnlock()
			if cm.watcher == nil {
				ch := make(chan fsnotify.Event)
				close(ch)
				return ch
			}
			return cm.watcher.Events
		}():
			if !ok {
				return
			}
			// 处理文件修改/创建/重命名事件
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				debounceTimer.Reset(cm.watchDebounceInterval)
			}

		case <-debounceTimer.C:
			// 自动重载配置
			if err := cm.ReloadConfig(); err != nil {
				fmt.Printf("[CONFIG] auto reload failed: %v\n", err)
			} else {
				fmt.Printf("[CONFIG] auto reloaded from: %s\n", cm.configPath)
			}

		case err, ok := <-func() <-chan error {
			cm.mu.RLock()
			defer cm.mu.RUnlock()
			if cm.watcher == nil {
				ch := make(chan error)
				close(ch)
				return ch
			}
			return cm.watcher.Errors
		}():
			if !ok {
				return
			}
			fmt.Printf("[CONFIG] watch error: %v\n", err)

		case <-cm.watchQuit:
			return
		}
	}
}

// parseConfigFile 解析配置文件
func (cm *ConfigManager) parseConfigFile() error {
	data, err := ioutil.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("read file failed: %w", err)
	}

	if err := cm.serializer.Unmarshal(data, cm.instance); err != nil {
		return fmt.Errorf("unmarshal failed (%s): %w", cm.serializer.GetName(), err)
	}

	return nil
}

// createNewInstance 创建新的配置实例
func (cm *ConfigManager) createNewInstance() interface{} {
	val := reflect.ValueOf(cm.instance)
	if val.Kind() != reflect.Ptr {
		return nil
	}
	return reflect.New(val.Elem().Type()).Interface()
}
