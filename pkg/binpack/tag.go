package binpack

import (
	"fmt"
	"strconv"
	"strings"
)

// TagInfo 表示解析后的 tag 信息
type TagInfo struct {
	Offset      int    // 字节偏移量，-1 表示末尾
	Size        int    // 字节大小，-1 表示变长
	ByteOrder   string // 字节序："be" 或 "le"
	LenField    string // 变长字段的长度来源字段名
	Encoding    string // 字符串编码：utf8, ascii, hex, gbk
	Bits        string // 位字段范围："0-3"
	Condition   string // 条件表达式："Field==Value"
	Checksum    string // 校验和类型和范围："crc16:0-100"
	IsRepeat    bool   // 是否为数组字段
	ElementSize int    // 每个元素的字节大小
	Skip        bool   // 是否跳过该字段
}

// tagInfo 内部使用的别名
type tagInfo = TagInfo

// ParseTag 解析 bin tag（导出版本）
func ParseTag(tag string) (*TagInfo, error) {
	return parseTag(tag)
}

// parseTag 解析 bin tag
// 格式: bin:"offset:size:endian,option1:value1,option2:value2"
// 示例:
//   bin:"0:4:be"           - 偏移 0，大小 4，大端序
//   bin:"4:1"              - 偏移 4，大小 1，默认大端序
//   bin:"5:var,len:Length" - 偏移 5，变长，长度由 Length 字段指定
//   bin:"-"                - 跳过字段
func parseTag(tag string) (*tagInfo, error) {
	if tag == "" {
		return nil, fmt.Errorf("empty tag")
	}

	if tag == "-" {
		return &tagInfo{Skip: true}, nil
	}

	info := &tagInfo{
		ByteOrder: "be", // 默认大端序
	}

	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid tag format")
	}

	// 解析基础格式: offset:size:endian
	base := strings.Split(parts[0], ":")
	if len(base) < 2 {
		return nil, fmt.Errorf("invalid base format, expected offset:size[:endian]")
	}

	// 解析偏移量
	if base[0] == "-1" {
		info.Offset = -1
	} else {
		offset, err := strconv.Atoi(base[0])
		if err != nil {
			return nil, fmt.Errorf("invalid offset: %w", err)
		}
		info.Offset = offset
	}

	// 解析大小
	if base[1] == "var" {
		info.Size = -1
	} else {
		size, err := strconv.Atoi(base[1])
		if err != nil {
			return nil, fmt.Errorf("invalid size: %w", err)
		}
		info.Size = size
	}

	// 解析字节序（可选）
	if len(base) >= 3 {
		if base[2] != "be" && base[2] != "le" {
			return nil, fmt.Errorf("invalid endian, must be 'be' or 'le'")
		}
		info.ByteOrder = base[2]
	}

	// 解析选项
	for i := 1; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])

		// 处理 repeat 标记（无值选项）
		if part == "repeat" {
			info.IsRepeat = true
			continue
		}

		opt := strings.SplitN(part, ":", 2)
		if len(opt) != 2 {
			return nil, fmt.Errorf("invalid option format: %s", part)
		}

		key := strings.TrimSpace(opt[0])
		value := strings.TrimSpace(opt[1])

		switch key {
		case "len":
			info.LenField = value
		case "enc":
			info.Encoding = value
		case "size":
			// 解析元素大小和字节序: "2:be" 或 "2"
			sizeparts := strings.Split(value, ":")
			elemSize, err := strconv.Atoi(sizeparts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid element size: %w", err)
			}
			info.ElementSize = elemSize
			if len(sizeparts) >= 2 {
				if sizeparts[1] != "be" && sizeparts[1] != "le" {
					return nil, fmt.Errorf("invalid endian in size, must be 'be' or 'le'")
				}
				info.ByteOrder = sizeparts[1]
			}
		case "bits":
			// 验证位字段格式: "0-3" 或 "4"
			if !strings.Contains(value, "-") {
				bit, err := strconv.Atoi(value)
				if err != nil || bit < 0 || bit > 7 {
					return nil, fmt.Errorf("invalid bit index: %s", value)
				}
			} else {
				bitRange := strings.Split(value, "-")
				if len(bitRange) != 2 {
					return nil, fmt.Errorf("invalid bit range: %s", value)
				}
				start, err1 := strconv.Atoi(bitRange[0])
				end, err2 := strconv.Atoi(bitRange[1])
				if err1 != nil || err2 != nil || start < 0 || end > 7 || start > end {
					return nil, fmt.Errorf("invalid bit range: %s", value)
				}
			}
			info.Bits = value
		case "if":
			info.Condition = value
		case "crc16", "crc32", "checksum":
			info.Checksum = key + ":" + value
		default:
			return nil, fmt.Errorf("unknown option: %s", key)
		}
	}

	return info, nil
}
