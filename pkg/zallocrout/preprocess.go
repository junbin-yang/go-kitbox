package zallocrout

import (
	"errors"
	"strings"
)

// 快速检查路径是否需要规范化
// 大多数路径是规范的，这个检查可以避免不必要的内存分配
//
//go:inline
func needsNormalization(path string) bool {
	if len(path) == 0 {
		return true
	}
	if path[0] != '/' {
		return true
	}
	if len(path) > 1 && path[len(path)-1] == '/' {
		return true
	}
	// 检查是否包含需要规范化的模式
	for i := 1; i < len(path); i++ {
		if path[i] == '/' && path[i-1] == '/' {
			return true // 包含 '//'
		}
		if path[i] == '.' && path[i-1] == '/' {
			return true // 可能包含 '/./' 或 '/../'
		}
	}
	return false
}

// 路径规范化（零分配实现）
// 处理 //、/./、/../ 等特殊情况
// 使用栈式算法避免内存分配
//
//go:nosplit
//go:inline
func normalizePathBytes(path []byte) []byte {
	if len(path) == 0 {
		return path
	}

	// 确保路径以 '/' 开头
	if path[0] != '/' {
		path = append([]byte{'/'}, path...)
	}

	// 使用栈式算法处理路径片段
	stack := make([]int, 0, 8) // 存储每个片段的起始位置
	writePos := 1              // 写入位置（跳过开头的 '/'）

	i := 1
	for i < len(path) {
		if path[i] == '/' {
			i++
			continue
		}

		// 找到片段的结束位置
		start := i
		for i < len(path) && path[i] != '/' {
			i++
		}

		segLen := i - start

		// 判断片段类型
		if segLen == 1 && path[start] == '.' {
			// 当前目录，跳过
			continue
		} else if segLen == 2 && path[start] == '.' && path[start+1] == '.' {
			// 父目录，弹出栈
			if len(stack) > 0 {
				writePos = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}
		} else {
			// 普通片段，压入栈
			stack = append(stack, writePos)
			copy(path[writePos:], path[start:i])
			writePos += segLen
			if writePos < len(path) {
				path[writePos] = '/'
			}
			writePos++
		}
	}

	// 移除末尾的 '/'（除非是根路径）
	if writePos > 1 {
		writePos--
	}

	return path[:writePos]
}

// 路径拆分（零分配实现）
// 将路径拆分为片段，支持栈分配优先
// 小路径使用栈分配，大路径动态扩容
func splitPathToCompressedSegs(path string, buf []string) []string {
	if len(path) == 0 || path == "/" {
		return buf[:0]
	}

	// 跳过开头的 '/'
	start := 1
	if path[0] != '/' {
		start = 0
	}

	result := buf[:0]
	for i := start; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				result = append(result, path[start:i])
			}
			start = i + 1
		}
	}

	// 添加最后一个片段
	if start < len(path) {
		result = append(result, path[start:])
	}

	return result
}

// 判断是否为参数片段（:id）
//
//go:inline
func isParamSeg(seg string) bool {
	return len(seg) > 1 && seg[0] == ':'
}

// 判断是否为通配符片段（*path）
//
//go:inline
func isWildcardSeg(seg string) bool {
	return len(seg) > 1 && seg[0] == '*'
}

// 判断是否为静态片段
//
//go:inline
func isStaticSeg(seg string) bool {
	return len(seg) > 0 && seg[0] != ':' && seg[0] != '*'
}

// 路由预检查（拒绝非法路径）
// 在注册时检查，避免运行时错误
func validateRoute(path string) error {
	if len(path) == 0 {
		return errors.New("path cannot be empty")
	}

	if path[0] != '/' {
		return errors.New("path must begin with '/'")
	}

	if strings.Contains(path, "//") {
		return errors.New("path cannot contain '//'")
	}

	if strings.Contains(path, "/./") {
		return errors.New("path cannot contain '/./'")
	}

	if strings.Contains(path, "/../") {
		return errors.New("path cannot contain '/../'")
	}

	// 检查参数和通配符格式
	segs := strings.Split(path, "/")
	for i, seg := range segs {
		if len(seg) == 0 {
			continue
		}

		// 参数片段检查
		if seg[0] == ':' {
			if len(seg) == 1 {
				return errors.New("param segment cannot be empty")
			}
			// 参数名只能包含字母、数字、下划线
			for _, c := range seg[1:] {
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
					(c >= '0' && c <= '9') || c == '_') {
					return errors.New("param name can only contain letters, numbers, and underscores")
				}
			}
		}

		// 通配符片段检查
		if seg[0] == '*' {
			if len(seg) == 1 {
				return errors.New("wildcard segment cannot be empty")
			}
			// 通配符必须是最后一个片段
			if i != len(segs)-1 {
				return errors.New("wildcard must be the last segment")
			}
		}
	}

	return nil
}
