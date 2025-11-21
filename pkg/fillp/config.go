package fillp

import (
	"github.com/junbin-yang/go-kitbox/pkg/congestion"
)

// ConnectionConfig FILLP连接配置
type ConnectionConfig struct {
	// 拥塞控制算法（可选，默认使用内置算法）
	CongestionAlgorithm congestion.AlgorithmType

	// 拥塞控制算法配置（可选）
	CongestionConfig interface{}

	// 其他配置项可以在这里扩展
}

// DefaultConfig 返回默认配置
func DefaultConfig() ConnectionConfig {
	return ConnectionConfig{
		// 默认不指定算法，使用FILLP内置
		CongestionAlgorithm: "",
	}
}
