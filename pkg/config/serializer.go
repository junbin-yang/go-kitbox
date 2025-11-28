package config

import (
	"bytes"
	"encoding/json"

	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v2"
)

// Serializer 定义序列化/反序列化接口，支持扩展不同格式
type Serializer interface {
	Marshal(v interface{}) ([]byte, error)      // 序列化
	Unmarshal(data []byte, v interface{}) error // 反序列化
	GetFileExt() string                         // 获取文件扩展名（如.yml/.json）
	GetName() string                            // 获取格式名称（如yaml/json）
}

// YAMLSerializer YAML序列化实现
type YAMLSerializer struct{}

func (y *YAMLSerializer) Marshal(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

func (y *YAMLSerializer) Unmarshal(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}

func (y *YAMLSerializer) GetFileExt() string {
	return ".yml"
}

func (y *YAMLSerializer) GetName() string {
	return "yaml"
}

// JSONSerializer JSON序列化实现
type JSONSerializer struct{}

func (j *JSONSerializer) Marshal(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func (j *JSONSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (j *JSONSerializer) GetFileExt() string {
	return ".json"
}

func (j *JSONSerializer) GetName() string {
	return "json"
}

// INISerializer INI序列化实现
type INISerializer struct{}

func (i *INISerializer) Marshal(v interface{}) ([]byte, error) {
	cfg := ini.Empty()
	if err := cfg.ReflectFrom(v); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (i *INISerializer) Unmarshal(data []byte, v interface{}) error {
	cfg, err := ini.Load(data)
	if err != nil {
		return err
	}
	return cfg.MapTo(v)
}

func (i *INISerializer) GetFileExt() string {
	return ".ini"
}

func (i *INISerializer) GetName() string {
	return "ini"
}
