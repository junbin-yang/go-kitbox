package main

import "strings"

// extractBinTag 从 struct tag 中提取 bin tag
func extractBinTag(tag string) string {
	if tag == "" {
		return ""
	}

	// 查找 bin:"..."
	for i := 0; i < len(tag); i++ {
		if i+4 < len(tag) && tag[i:i+4] == "bin:" {
			start := i + 5 // 跳过 bin:"
			end := start
			for end < len(tag) && tag[end] != '"' {
				end++
			}
			return tag[start:end]
		}
	}
	return ""
}

// extractConditionField 从条件表达式中提取字段名
func extractConditionField(condition string) string {
	for _, op := range []string{"==", "!="} {
		if idx := strings.Index(condition, op); idx >= 0 {
			return condition[:idx]
		}
	}
	return ""
}
