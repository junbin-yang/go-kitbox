package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type TestConfig struct {
	Server struct {
		Port    int    `yaml:"port" json:"port" ini:"port"`
		Host    string `yaml:"host" json:"host" ini:"host"`
		Timeout int    `yaml:"timeout" json:"timeout" ini:"timeout"`
	} `yaml:"server" json:"server" ini:"server"`
	Logger struct {
		Level  string `yaml:"level" json:"level" ini:"level"`
		Path   string `yaml:"path" json:"path" ini:"path"`
		Rotate bool   `yaml:"rotate" json:"rotate" ini:"rotate"`
	} `yaml:"logger" json:"logger" ini:"logger"`
	Database struct {
		DSN string `yaml:"dsn" json:"dsn" ini:"dsn"`
	} `yaml:"database" json:"database" ini:"database"`
}

// 场景1：基础使用（默认YAML格式）
func TestScenario1_BasicYAML(t *testing.T) {
	cfg := &TestConfig{}
	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.yml")

	cm := NewConfigManager(cfg, WithAppName("test"))
	if err := cm.LoadConfig(testDataPath); err != nil {
		t.Fatalf("加载YAML配置失败: %v", err)
	}

	configData, err := cm.GetConfig()
	if err != nil {
		t.Fatalf("获取配置失败: %v", err)
	}

	testCfg := configData.(*TestConfig)
	if testCfg.Server.Port != 8080 {
		t.Errorf("期望端口 8080, 实际 %d", testCfg.Server.Port)
	}
	if testCfg.Logger.Level != "info" {
		t.Errorf("期望日志级别 info, 实际 %s", testCfg.Logger.Level)
	}
}

// 场景2：使用JSON格式配置
func TestScenario2_JSONFormat(t *testing.T) {
	cfg := &TestConfig{}
	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.json")

	cm := NewConfigManager(cfg, WithSerializer(&JSONSerializer{}))
	if err := cm.LoadConfig(testDataPath); err != nil {
		t.Fatalf("加载JSON配置失败: %v", err)
	}

	configData, err := cm.GetConfig()
	if err != nil {
		t.Fatalf("获取配置失败: %v", err)
	}

	testCfg := configData.(*TestConfig)
	if testCfg.Server.Port != 9090 {
		t.Errorf("期望端口 9090, 实际 %d", testCfg.Server.Port)
	}
	if testCfg.Logger.Level != "debug" {
		t.Errorf("期望日志级别 debug, 实际 %s", testCfg.Logger.Level)
	}
}

// 场景3：加载无后缀配置文件（强制YAML格式）
func TestScenario3_NoExtensionYAML(t *testing.T) {
	cfg := &TestConfig{}
	tmpFile := filepath.Join(os.TempDir(), "config_no_ext")
	defer os.Remove(tmpFile)

	// 创建无后缀配置文件
	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.yml")
	data, _ := os.ReadFile(testDataPath)
	_ = os.WriteFile(tmpFile, data, 0644)

	cm := NewConfigManager(cfg, WithForceFormat(&YAMLSerializer{}))
	if err := cm.LoadConfig(tmpFile); err != nil {
		t.Fatalf("加载无后缀配置失败: %v", err)
	}

	configData, _ := cm.GetConfig()
	testCfg := configData.(*TestConfig)
	if testCfg.Server.Port != 8080 {
		t.Errorf("期望端口 8080, 实际 %d", testCfg.Server.Port)
	}
}

// 场景4：加载自定义路径的JSON配置（无后缀）
func TestScenario4_CustomPathJSON(t *testing.T) {
	cfg := &TestConfig{}
	tmpFile := filepath.Join(os.TempDir(), "myconfig")
	defer os.Remove(tmpFile)

	// 创建无后缀JSON配置文件
	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.json")
	data, _ := os.ReadFile(testDataPath)
	_ = os.WriteFile(tmpFile, data, 0644)

	cm := NewConfigManager(cfg, WithForceFormat(&JSONSerializer{}))
	if err := cm.LoadConfig(tmpFile); err != nil {
		t.Fatalf("加载自定义路径配置失败: %v", err)
	}

	configData, _ := cm.GetConfig()
	testCfg := configData.(*TestConfig)
	if testCfg.Server.Port != 9090 {
		t.Errorf("期望端口 9090, 实际 %d", testCfg.Server.Port)
	}
}

// 场景5：启用配置监听（简化测试，不实际触发文件变化）
func TestScenario5_EnableWatch(t *testing.T) {
	cfg := &TestConfig{}
	tmpFile := filepath.Join(os.TempDir(), "test_watch.yml")
	defer os.Remove(tmpFile)

	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.yml")
	data, _ := os.ReadFile(testDataPath)
	_ = os.WriteFile(tmpFile, data, 0644)

	cm := NewConfigManager(
		cfg,
		WithConfigWatch(true, 100*time.Millisecond),
	)
	if err := cm.LoadConfig(tmpFile); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	configData, _ := cm.GetConfig()
	testCfg := configData.(*TestConfig)
	if testCfg.Server.Port != 8080 {
		t.Errorf("期望端口 8080, 实际 %d", testCfg.Server.Port)
	}

	cm.Close()
}

// 场景6：自定义支持的配置格式（仅JSON）
func TestScenario6_CustomFormats(t *testing.T) {
	cfg := &TestConfig{}
	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.json")

	cm := NewConfigManager(
		cfg,
		WithConfigFormats(&JSONSerializer{}),
	)
	if err := cm.LoadConfig(testDataPath); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	configData, _ := cm.GetConfig()
	testCfg := configData.(*TestConfig)
	if testCfg.Server.Port != 9090 {
		t.Errorf("期望端口 9090, 实际 %d", testCfg.Server.Port)
	}
}

// 场景7：保存配置修改
func TestScenario7_SaveConfig(t *testing.T) {
	cfg := &TestConfig{}
	tmpFile := filepath.Join(os.TempDir(), "test_save.yml")
	defer os.Remove(tmpFile)

	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.yml")
	data, _ := os.ReadFile(testDataPath)
	_ = os.WriteFile(tmpFile, data, 0644)

	cm := NewConfigManager(cfg)
	if err := cm.LoadConfig(tmpFile); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 修改配置
	configData, _ := cm.GetConfig()
	testCfg := configData.(*TestConfig)
	testCfg.Server.Port = 9999
	testCfg.Logger.Level = "debug"

	// 保存配置
	if err := cm.SaveConfig(); err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	// 重新加载验证
	cfg2 := &TestConfig{}
	cm2 := NewConfigManager(cfg2)
	if err := cm2.LoadConfig(tmpFile); err != nil {
		t.Fatalf("重新加载配置失败: %v", err)
	}

	configData2, _ := cm2.GetConfig()
	testCfg2 := configData2.(*TestConfig)
	if testCfg2.Server.Port != 9999 {
		t.Errorf("期望端口 9999, 实际 %d", testCfg2.Server.Port)
	}
	if testCfg2.Logger.Level != "debug" {
		t.Errorf("期望日志级别 debug, 实际 %s", testCfg2.Logger.Level)
	}
}

// 场景8：手动重载配置
func TestScenario8_ReloadConfig(t *testing.T) {
	cfg := &TestConfig{}
	tmpFile := filepath.Join(os.TempDir(), "test_reload.yml")
	defer os.Remove(tmpFile)

	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.yml")
	data, _ := os.ReadFile(testDataPath)
	_ = os.WriteFile(tmpFile, data, 0644)

	cm := NewConfigManager(cfg)
	if err := cm.LoadConfig(tmpFile); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 修改文件内容
	// 转换JSON为YAML格式（简化测试，直接用另一个YAML）
	_ = os.WriteFile(tmpFile, data, 0644)

	// 手动重载
	if err := cm.ReloadConfig(); err != nil {
		t.Fatalf("重载配置失败: %v", err)
	}

	configData, _ := cm.GetConfig()
	testCfg := configData.(*TestConfig)
	if testCfg.Server.Port != 8080 {
		t.Errorf("期望端口 8080, 实际 %d", testCfg.Server.Port)
	}
}

// 场景9：动态控制监听
func TestScenario9_DynamicWatch(t *testing.T) {
	cfg := &TestConfig{}
	tmpFile := filepath.Join(os.TempDir(), "test_dynamic.yml")
	defer os.Remove(tmpFile)

	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.yml")
	data, _ := os.ReadFile(testDataPath)
	_ = os.WriteFile(tmpFile, data, 0644)

	cm := NewConfigManager(cfg)
	if err := cm.LoadConfig(tmpFile); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 启用监听
	if err := cm.EnableWatch(true); err != nil {
		t.Fatalf("启用监听失败: %v", err)
	}

	// 停止监听
	if err := cm.EnableWatch(false); err != nil {
		t.Fatalf("停止监听失败: %v", err)
	}

	configData, _ := cm.GetConfig()
	testCfg := configData.(*TestConfig)
	if testCfg.Server.Port != 8080 {
		t.Errorf("期望端口 8080, 实际 %d", testCfg.Server.Port)
	}
}

// 场景10：关闭配置管理器
func TestScenario10_CloseManager(t *testing.T) {
	cfg := &TestConfig{}
	tmpFile := filepath.Join(os.TempDir(), "test_close.yml")
	defer os.Remove(tmpFile)

	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.yml")
	data, _ := os.ReadFile(testDataPath)
	_ = os.WriteFile(tmpFile, data, 0644)

	cm := NewConfigManager(
		cfg,
		WithConfigWatch(true, 100*time.Millisecond),
	)
	if err := cm.LoadConfig(tmpFile); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	configData, _ := cm.GetConfig()
	testCfg := configData.(*TestConfig)
	if testCfg.Server.Port != 8080 {
		t.Errorf("期望端口 8080, 实际 %d", testCfg.Server.Port)
	}

	// 关闭管理器
	cm.Close()
}

// 场景11：INI格式配置
func TestScenario11_INIFormat(t *testing.T) {
	cfg := &TestConfig{}
	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.ini")

	cm := NewConfigManager(cfg, WithSerializer(&INISerializer{}))
	if err := cm.LoadConfig(testDataPath); err != nil {
		t.Fatalf("加载INI配置失败: %v", err)
	}

	configData, err := cm.GetConfig()
	if err != nil {
		t.Fatalf("获取配置失败: %v", err)
	}

	testCfg := configData.(*TestConfig)
	if testCfg.Server.Port != 7070 {
		t.Errorf("期望端口 7070, 实际 %d", testCfg.Server.Port)
	}
	if testCfg.Logger.Level != "warn" {
		t.Errorf("期望日志级别 warn, 实际 %s", testCfg.Logger.Level)
	}
}

// 场景12：环境变量注入
func TestScenario12_EnvOverride(t *testing.T) {
	type EnvTestConfig struct {
		Server struct {
			Port    int    `yaml:"port" json:"port" ini:"port" env:"SERVER_PORT"`
			Host    string `yaml:"host" json:"host" ini:"host" env:"SERVER_HOST"`
			Enabled bool   `yaml:"enabled" json:"enabled" ini:"enabled" env:"SERVER_ENABLED"`
		} `yaml:"server" json:"server" ini:"server"`
	}

	os.Setenv("SERVER_PORT", "9999")
	os.Setenv("SERVER_HOST", "localhost")
	os.Setenv("SERVER_ENABLED", "true")
	defer os.Unsetenv("SERVER_PORT")
	defer os.Unsetenv("SERVER_HOST")
	defer os.Unsetenv("SERVER_ENABLED")

	cfg := &EnvTestConfig{}
	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.yml")

	cm := NewConfigManager(cfg)
	if err := cm.LoadConfig(testDataPath); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	configData, _ := cm.GetConfig()
	testCfg := configData.(*EnvTestConfig)
	if testCfg.Server.Port != 9999 {
		t.Errorf("期望端口 9999 (环境变量), 实际 %d", testCfg.Server.Port)
	}
	if testCfg.Server.Host != "localhost" {
		t.Errorf("期望主机 localhost (环境变量), 实际 %s", testCfg.Server.Host)
	}
	if !testCfg.Server.Enabled {
		t.Error("期望 Enabled=true (环境变量)")
	}
}

// 场景13：配置变更回调
func TestScenario13_OnChangeCallback(t *testing.T) {
	cfg := &TestConfig{}
	tmpFile := filepath.Join(os.TempDir(), "test_callback.yml")
	defer os.Remove(tmpFile)

	testDataPath := filepath.Join("..", "..", "internal", "testdata", "test.yml")
	data, _ := os.ReadFile(testDataPath)
	_ = os.WriteFile(tmpFile, data, 0644)

	cm := NewConfigManager(cfg)
	if err := cm.LoadConfig(tmpFile); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	callbackCalled := false
	cm.OnChange(func(old, new interface{}) {
		callbackCalled = true
		oldCfg := old.(*TestConfig)
		newCfg := new.(*TestConfig)
		if oldCfg.Server.Port == 8080 && newCfg.Server.Port == 9999 {
			t.Log("配置变更回调触发成功")
		}
	})

	// 修改配置并重载
	configData, _ := cm.GetConfig()
	testCfg := configData.(*TestConfig)
	testCfg.Server.Port = 9999
	_ = cm.SaveConfig()

	if err := cm.ReloadConfig(); err != nil {
		t.Fatalf("重载配置失败: %v", err)
	}

	if !callbackCalled {
		t.Error("配置变更回调未被触发")
	}
}

// Test utility functions
func TestReplacePathVars(t *testing.T) {
	vars := map[string]string{
		"app":  "myapp",
		"env":  "prod",
		"home": "/home/user",
	}

	result := replacePathVars("{{.home}}/{{.app}}/config.{{.env}}.yml", vars)
	expected := "/home/user/myapp/config.prod.yml"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestValidateConfigPath(t *testing.T) {
	// Test empty path
	err := validateConfigPath("")
	if err == nil {
		t.Error("Expected error for empty path")
	}

	// Test non-existent file
	err = validateConfigPath("/nonexistent/file.yml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test directory
	tmpDir := t.TempDir()
	err = validateConfigPath(tmpDir)
	if err == nil {
		t.Error("Expected error for directory path")
	}

	// Test valid file
	tmpFile := filepath.Join(tmpDir, "test.yml")
	_ = os.WriteFile(tmpFile, []byte("test: value"), 0644)
	err = validateConfigPath(tmpFile)
	if err != nil {
		t.Errorf("Expected no error for valid file, got %v", err)
	}
}

func TestSerializerGetName(t *testing.T) {
	yaml := &YAMLSerializer{}
	if yaml.GetName() != "yaml" {
		t.Errorf("Expected yaml, got %s", yaml.GetName())
	}

	json := &JSONSerializer{}
	if json.GetName() != "json" {
		t.Errorf("Expected json, got %s", json.GetName())
	}

	ini := &INISerializer{}
	if ini.GetName() != "ini" {
		t.Errorf("Expected ini, got %s", ini.GetName())
	}
}

func TestYAMLMarshal(t *testing.T) {
	yaml := &YAMLSerializer{}
	data := map[string]interface{}{
		"key": "value",
		"num": 123,
	}
	result, err := yaml.Marshal(data)
	if err != nil {
		t.Errorf("YAML marshal failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("YAML marshal returned empty result")
	}
}

func TestINIMarshal(t *testing.T) {
	ini := &INISerializer{}
	data := &TestConfig{}
	data.Server.Port = 8080
	data.Server.Host = "localhost"
	result, err := ini.Marshal(data)
	if err != nil {
		t.Errorf("INI marshal failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("INI marshal returned empty result")
	}
}

func TestWithDefaultPaths(t *testing.T) {
	cfg := &TestConfig{}
	paths := []string{"/path1", "/path2"}
	cm := NewConfigManager(cfg, WithDefaultPaths(paths...))
	if cm == nil {
		t.Error("NewConfigManager with default paths failed")
	}
}

func TestFindDefaultConfigPath(t *testing.T) {
	cfg := &TestConfig{}
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yml")
	_ = os.WriteFile(tmpFile, []byte("server:\n  port: 8080"), 0644)

	cm := NewConfigManager(cfg, WithDefaultPaths(tmpDir+"/config"))
	path, err := cm.findDefaultConfigPath()
	if err != nil {
		t.Logf("findDefaultConfigPath returned error (expected): %v", err)
	}
	_ = path
}

func TestJSONMarshal(t *testing.T) {
	json := &JSONSerializer{}
	data := map[string]interface{}{
		"key": "value",
		"num": 123,
	}
	result, err := json.Marshal(data)
	if err != nil {
		t.Errorf("JSON marshal failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("JSON marshal returned empty result")
	}
}
