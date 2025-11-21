package fillp

import (
	"net"

	"github.com/junbin-yang/go-kitbox/pkg/congestion"
)

// NewConnectionWithConfig 创建带配置的FILLP连接
func NewConnectionWithConfig(localAddr, remoteAddr net.Addr, config ConnectionConfig) (*Connection, error) {
	// 先创建基础连接
	conn, err := NewConnection(localAddr, remoteAddr)
	if err != nil {
		return nil, err
	}

	// 初始化拥塞控制
	if err := conn.initCongestionControl(config); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

// 为了方便使用，添加快捷函数

// NewConnectionWithCUBIC 创建使用CUBIC算法的连接
func NewConnectionWithCUBIC(localAddr, remoteAddr net.Addr) (*Connection, error) {
	config := ConnectionConfig{
		CongestionAlgorithm: congestion.AlgorithmCubic,
	}
	return NewConnectionWithConfig(localAddr, remoteAddr, config)
}

// NewConnectionWithBBR 创建使用BBR算法的连接
func NewConnectionWithBBR(localAddr, remoteAddr net.Addr) (*Connection, error) {
	config := ConnectionConfig{
		CongestionAlgorithm: congestion.AlgorithmBBR,
	}
	return NewConnectionWithConfig(localAddr, remoteAddr, config)
}

// NewConnectionWithVegas 创建使用Vegas算法的连接
func NewConnectionWithVegas(localAddr, remoteAddr net.Addr) (*Connection, error) {
	config := ConnectionConfig{
		CongestionAlgorithm: congestion.AlgorithmVegas,
	}
	return NewConnectionWithConfig(localAddr, remoteAddr, config)
}
