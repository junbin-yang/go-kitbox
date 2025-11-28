package binpack

import (
	"fmt"
	"reflect"
	"sort"
)

// ValidateError 验证错误
type ValidateError struct {
	Field   string
	Message string
}

func (e *ValidateError) Error() string {
	return fmt.Sprintf("field %s: %s", e.Field, e.Message)
}

// ValidateStruct 验证结构体标签是否合法
func ValidateStruct(v interface{}) error {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("v must be a struct or pointer to struct")
	}

	return validateStructType(typ)
}

// validateStructType 验证结构体类型
func validateStructType(typ reflect.Type) error {
	fieldMap := make(map[string]int)         // 字段名 -> 索引
	offsetMap := make(map[int][]string)      // 偏移量 -> 字段名列表（允许位字段和条件字段共享）
	var offsets []int                        // 所有偏移量（用于排序）
	lenFields := make(map[string]bool)       // 长度字段名集合
	condFields := make(map[string]bool)      // 条件字段名集合
	fieldInfoMap := make(map[string]*TagInfo) // 字段名 -> TagInfo

	// 第一遍：收集所有字段信息
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldMap[field.Name] = i

		tag := field.Tag.Get("bin")
		if tag == "" || tag == "-" {
			continue
		}

		info, err := parseTag(tag)
		if err != nil {
			return &ValidateError{
				Field:   field.Name,
				Message: fmt.Sprintf("invalid tag: %v", err),
			}
		}

		fieldInfoMap[field.Name] = info

		// 检查偏移量冲突（位字段和条件字段可以共享偏移量）
		if info.Offset >= 0 {
			existingFields := offsetMap[info.Offset]

			// 检查是否可以共享偏移量
			canShare := info.Bits != "" || info.Condition != ""
			if len(existingFields) > 0 && !canShare {
				// 检查已存在的字段是否都是位字段或条件字段
				allCanShare := true
				for _, existingField := range existingFields {
					existingInfo := fieldInfoMap[existingField]
					if existingInfo.Bits == "" && existingInfo.Condition == "" {
						allCanShare = false
						break
					}
				}
				if !allCanShare {
					return &ValidateError{
						Field:   field.Name,
						Message: fmt.Sprintf("offset %d conflicts with field %s", info.Offset, existingFields[0]),
					}
				}
			}

			offsetMap[info.Offset] = append(offsetMap[info.Offset], field.Name)
			if len(existingFields) == 0 {
				offsets = append(offsets, info.Offset)
			}
		}

		// 收集依赖字段
		if info.LenField != "" {
			lenFields[info.LenField] = true
		}
		if info.Condition != "" {
			// 解析条件字段名: "Field==Value"
			if idx := findConditionField(info.Condition); idx != "" {
				condFields[idx] = true
			}
		}
	}

	// 第二遍：验证依赖字段是否存在
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("bin")
		if tag == "" || tag == "-" {
			continue
		}

		info, _ := parseTag(tag)

		// 验证长度字段是否存在
		if info.LenField != "" {
			if _, exists := fieldMap[info.LenField]; !exists {
				return &ValidateError{
					Field:   field.Name,
					Message: fmt.Sprintf("length field %s does not exist", info.LenField),
				}
			}
			// 验证长度字段必须在当前字段之前
			if fieldMap[info.LenField] >= i {
				return &ValidateError{
					Field:   field.Name,
					Message: fmt.Sprintf("length field %s must appear before this field", info.LenField),
				}
			}
		}

		// 验证条件字段是否存在
		if info.Condition != "" {
			condFieldName := findConditionField(info.Condition)
			if condFieldName != "" {
				if _, exists := fieldMap[condFieldName]; !exists {
					return &ValidateError{
						Field:   field.Name,
						Message: fmt.Sprintf("condition field %s does not exist", condFieldName),
					}
				}
				// 验证条件字段必须在当前字段之前
				if fieldMap[condFieldName] >= i {
					return &ValidateError{
						Field:   field.Name,
						Message: fmt.Sprintf("condition field %s must appear before this field", condFieldName),
					}
				}
			}
		}

		// 验证变长字段必须有长度字段
		if info.Size == -1 && info.LenField == "" && !info.IsRepeat {
			return &ValidateError{
				Field:   field.Name,
				Message: "variable-length field must specify len field",
			}
		}

		// 验证数组字段必须有长度字段
		if info.IsRepeat && info.LenField == "" {
			return &ValidateError{
				Field:   field.Name,
				Message: "repeat field must specify len field",
			}
		}
	}

	// 第三遍：验证偏移量是否重叠（跳过位字段和条件字段）
	sort.Ints(offsets)
	for i := 0; i < len(offsets)-1; i++ {
		currentOffset := offsets[i]
		nextOffset := offsets[i+1]
		currentFields := offsetMap[currentOffset]

		// 检查当前偏移量的所有字段
		for _, currentField := range currentFields {
			info := fieldInfoMap[currentField]
			// 位字段和条件字段可以共享偏移量，跳过重叠检查
			if info.Bits != "" || info.Condition != "" {
				continue
			}
			if info.Size > 0 && currentOffset+info.Size > nextOffset {
				return &ValidateError{
					Field:   currentField,
					Message: fmt.Sprintf("field overlaps with %s (offset %d + size %d > %d)", offsetMap[nextOffset][0], currentOffset, info.Size, nextOffset),
				}
			}
		}
	}

	return nil
}

// findConditionField 从条件表达式中提取字段名
// 支持格式: "Field==Value", "Field!=Value"
func findConditionField(condition string) string {
	for _, op := range []string{"==", "!="} {
		if idx := findString(condition, op); idx >= 0 {
			return condition[:idx]
		}
	}
	return ""
}

func findString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
