package zallocrout

import (
	"testing"
)

// 测试路径规范化
func TestNormalizePathBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"正常路径", "/api/v1/users", "/api/v1/users"},
		{"根路径", "/", "/"},
		{"空路径", "", ""},
		{"无开头斜杠", "api/users", "/api/users"},
		{"结尾斜杠", "/api/users/", "/api/users"},
		{"双斜杠", "/api//users", "/api/users"},
		{"多个双斜杠", "/api///users//list", "/api/users/list"},
		{"当前目录", "/api/./users", "/api/users"},
		{"多个当前目录", "/api/./v1/./users", "/api/v1/users"},
		{"父目录", "/api/v1/../users", "/api/users"},
		{"多个父目录", "/api/v1/v2/../../users", "/api/users"},
		{"复杂路径", "/api/./v1/../users//list", "/api/users/list"},
		{"根目录父目录", "/../api", "/api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.input)
			result := normalizePathBytes(input)
			if string(result) != tt.expected {
				t.Errorf("normalizePathBytes(%q) = %q, want %q", tt.input, string(result), tt.expected)
			}
		})
	}
}

// 测试路径拆分
func TestSplitPathToCompressedSegs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"正常路径", "/api/v1/users", []string{"api", "v1", "users"}},
		{"根路径", "/", []string{}},
		{"空路径", "", []string{}},
		{"单片段", "/api", []string{"api"}},
		{"参数路径", "/users/:id", []string{"users", ":id"}},
		{"通配符路径", "/files/*path", []string{"files", "*path"}},
		{"复杂路径", "/api/v1/users/:id/posts", []string{"api", "v1", "users", ":id", "posts"}},
		{"超长路径", "/a/b/c/d/e/f/g/h/i/j", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf [8]string
			result := splitPathToCompressedSegs(tt.input, buf[:0])

			if len(result) != len(tt.expected) {
				t.Errorf("splitPathToCompressedSegs(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splitPathToCompressedSegs(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// 测试栈分配扩容
func TestSplitPathToCompressedSegs_Expansion(t *testing.T) {
	// 测试超过 8 个片段的路径（触发扩容）
	path := "/a/b/c/d/e/f/g/h/i/j/k/l"
	var buf [8]string
	result := splitPathToCompressedSegs(path, buf[:0])

	expected := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"}
	if len(result) != len(expected) {
		t.Errorf("splitPathToCompressedSegs length = %d, want %d", len(result), len(expected))
		return
	}

	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("splitPathToCompressedSegs[%d] = %q, want %q", i, result[i], expected[i])
		}
	}
}

// 测试参数片段判断
func TestIsParamSeg(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{":id", true},
		{":userId", true},
		{":user_id", true},
		{"id", false},
		{":", false},
		{"", false},
		{"*path", false},
		{"api", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isParamSeg(tt.input)
			if result != tt.expected {
				t.Errorf("isParamSeg(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// 测试通配符片段判断
func TestIsWildcardSeg(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"*path", true},
		{"*filepath", true},
		{"*", false},
		{"", false},
		{":id", false},
		{"api", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isWildcardSeg(tt.input)
			if result != tt.expected {
				t.Errorf("isWildcardSeg(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// 测试静态片段判断
func TestIsStaticSeg(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"api", true},
		{"users", true},
		{"v1", true},
		{":id", false},
		{"*path", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isStaticSeg(tt.input)
			if result != tt.expected {
				t.Errorf("isStaticSeg(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// 测试路由预检查
func TestValidateRoute(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"正常路径", "/api/v1/users", false},
		{"参数路径", "/users/:id", false},
		{"通配符路径", "/files/*path", false},
		{"复杂路径", "/api/v1/users/:id/posts", false},
		{"空路径", "", true},
		{"无开头斜杠", "api/users", true},
		{"双斜杠", "/api//users", true},
		{"当前目录", "/api/./users", true},
		{"父目录", "/api/../users", true},
		{"空参数", "/users/:", true},
		{"空通配符", "/files/*", true},
		{"通配符不在末尾", "/files/*/list", true},
		{"非法参数名", "/users/:id-name", true},
		{"参数名包含特殊字符", "/users/:id@name", true},
		{"合法参数名", "/users/:user_id", false},
		{"合法参数名2", "/users/:userId123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRoute(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRoute(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

// 基准测试：路径规范化
func BenchmarkNormalizePathBytes(b *testing.B) {
	path := []byte("/api/v1/users/123/posts")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = normalizePathBytes(path)
	}
}

// 基准测试：路径拆分
func BenchmarkSplitPathToCompressedSegs(b *testing.B) {
	path := "/api/v1/users/123/posts"
	var buf [8]string
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = splitPathToCompressedSegs(path, buf[:0])
	}
}

// 基准测试：路由预检查
func BenchmarkValidateRoute(b *testing.B) {
	path := "/api/v1/users/:id/posts"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = validateRoute(path)
	}
}

// 测试needsNormalization边界情况
func TestNeedsNormalization_EdgeCases(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"", true},
		{"/", false},
		{"/a", false},
		{"/a/", true},
		{"//", true},
		{"/./", true},
		{"/../", true},
		{"/a/b", false},
	}

	for _, tt := range tests {
		got := needsNormalization(tt.path)
		if got != tt.want {
			t.Errorf("needsNormalization(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// 测试needsNormalization所有分支
func TestNeedsNormalization_AllBranches(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"空路径", "", true},
		{"无开头斜杠", "users", true},
		{"结尾斜杠", "/users/", true},
		{"双斜杠", "/users//posts", true},
		{"点斜杠", "/users/./posts", true},
		{"正常路径", "/users/posts", false},
		{"单字符路径", "/", false},
		{"两字符路径", "/a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsNormalization(tt.path)
			if got != tt.want {
				t.Errorf("needsNormalization(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
