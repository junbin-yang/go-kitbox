package main

import (
	hexlib "encoding/hex"
	"flag"
	"fmt"
	"go/types"
	"os"
	"strings"

	"github.com/junbin-yang/go-kitbox/pkg/binpack/generator"
	"golang.org/x/tools/go/packages"
)

// FieldDebug 字段调试信息
type FieldDebug struct {
	Name      string
	Type      string
	Offset    int
	Size      int
	ByteOrder string
	IsVar     bool
	LenField  string
	Bits      string
}

func runDebug(args []string) {
	fs := flag.NewFlagSet("debug", flag.ExitOnError)
	pkg := fs.String("pkg", "", "包路径（如：./mypackage）")
	typ := fs.String("type", "", "结构体类型名")
	dataFile := fs.String("data", "", "二进制数据文件路径")
	hexStr := fs.String("hex", "", "十六进制字符串（如：0x1234ABCD 或 1234ABCD）")

	fs.Usage = func() {
		fmt.Println("Usage: binpack debug -pkg <package> -type <struct> (-data <file> | -hex <string>)")
		fmt.Println()
		fmt.Println("Debug binary data with protocol definition visualization.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  binpack debug -pkg ./mypackage -type Packet -data packet.bin")
		fmt.Println("  binpack debug -pkg ./mypackage -type Packet -hex \"1234567890ABCDEF\"")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *pkg == "" || *typ == "" {
		fs.Usage()
		os.Exit(1)
	}

	if *dataFile == "" && *hexStr == "" {
		fmt.Fprintf(os.Stderr, "Error: either -data or -hex must be specified\n\n")
		fs.Usage()
		os.Exit(1)
	}

	// 读取二进制数据
	var binaryData []byte
	var err error

	if *dataFile != "" {
		binaryData, err = os.ReadFile(*dataFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read data file: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 解析十六进制字符串
		hexString := strings.TrimPrefix(*hexStr, "0x")
		hexString = strings.TrimPrefix(hexString, "0X")
		hexString = strings.ReplaceAll(hexString, " ", "")
		hexString = strings.ReplaceAll(hexString, ":", "")
		hexString = strings.ReplaceAll(hexString, "-", "")

		binaryData, err = hexlib.DecodeString(hexString)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to decode hex string: %v\n", err)
			os.Exit(1)
		}
	}

	// 加载包
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
	}
	pkgs, err := packages.Load(cfg, *pkg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load package: %v\n", err)
		os.Exit(1)
	}

	if len(pkgs) == 0 {
		fmt.Fprintf(os.Stderr, "No packages found\n")
		os.Exit(1)
	}

	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	// 查找类型
	var targetType types.Type
	for _, p := range pkgs {
		obj := p.Types.Scope().Lookup(*typ)
		if obj != nil {
			if tn, ok := obj.(*types.TypeName); ok {
				targetType = tn.Type()
				break
			}
		}
	}

	if targetType == nil {
		fmt.Fprintf(os.Stderr, "Type %s not found in package %s\n", *typ, *pkg)
		os.Exit(1)
	}

	// 转换为结构体
	structType, ok := targetType.Underlying().(*types.Struct)
	if !ok {
		fmt.Fprintf(os.Stderr, "Type %s is not a struct\n", *typ)
		os.Exit(1)
	}

	// 收集字段信息
	fields := make([]FieldDebug, 0)

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		binTag := extractBinTag(tag)
		if binTag == "" || binTag == "-" {
			continue
		}

		// 解析 tag
		tagInfo, err := generator.ParseTagForGen(binTag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse tag for field %s: %v\n", field.Name(), err)
			os.Exit(1)
		}

		// 简化类型名
		typeName := field.Type().String()
		if basic, ok := field.Type().(*types.Basic); ok {
			typeName = basic.Name()
		}

		fd := FieldDebug{
			Name:      field.Name(),
			Type:      typeName,
			Offset:    tagInfo.Offset,
			Size:      tagInfo.Size,
			ByteOrder: tagInfo.ByteOrder,
			Bits:      tagInfo.Bits,
		}

		if tagInfo.Size == -1 {
			fd.IsVar = true
			fd.LenField = tagInfo.LenField
		}

		fields = append(fields, fd)
	}

	// 生成调试输出
	debugOutput := generateDebugOutput(*typ, fields, binaryData)
	fmt.Print(debugOutput)
}

func generateDebugOutput(typeName string, fields []FieldDebug, data []byte) string {
	var sb strings.Builder

	// 检查是否有错误
	hasErrors := false
	for _, fd := range fields {
		if !fd.IsVar && (fd.Offset < 0 || fd.Offset+fd.Size > len(data)) {
			hasErrors = true
			break
		}
	}

	// 标题
	sb.WriteString("╔═══════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString(fmt.Sprintf("║  Binary Data Debug: %-52s║\n", typeName))
	sb.WriteString(fmt.Sprintf("║  Data Length: %-59d║\n", len(data)))
	if hasErrors {
		sb.WriteString("║  Status: ⚠️  ERRORS DETECTED                                              ║\n")
	}
	sb.WriteString("╚═══════════════════════════════════════════════════════════════════════════╝\n\n")

	// 计算期望的最小数据长度
	expectedSize := 0
	for _, fd := range fields {
		if !fd.IsVar && fd.Offset >= 0 && fd.Offset+fd.Size > expectedSize {
			expectedSize = fd.Offset + fd.Size
		}
	}

	// 十六进制视图
	sb.WriteString("Hexadecimal View:\n")
	sb.WriteString("────────────────────────────────────────────────────────────────────────────\n\n")

	maxLen := len(data)
	if expectedSize > maxLen {
		maxLen = expectedSize
	}

	for offset := 0; offset < maxLen; offset += 16 {
		// 偏移量
		sb.WriteString(fmt.Sprintf("%04X: ", offset))

		// 十六进制字节
		for i := 0; i < 16; i++ {
			bytePos := offset + i
			if bytePos >= maxLen {
				sb.WriteString("   ")
				continue
			}

			// 检查是否缺失数据
			if bytePos >= len(data) {
				sb.WriteString(" XX ")
				continue
			}

			// 检查是否为字段起始
			isFieldStart := false
			isErrorField := false
			for _, fd := range fields {
				if !fd.IsVar && fd.Offset >= 0 && bytePos == fd.Offset {
					isFieldStart = true
					// 检查该字段是否有错误
					if fd.Offset+fd.Size > len(data) {
						isErrorField = true
					}
					break
				}
			}

			if isFieldStart {
				if isErrorField {
					sb.WriteString(fmt.Sprintf("!%02X!", data[bytePos]))
				} else {
					sb.WriteString(fmt.Sprintf("[%02X]", data[bytePos]))
				}
			} else {
				sb.WriteString(fmt.Sprintf(" %02X ", data[bytePos]))
			}
		}

		sb.WriteString(" │ ")

		// ASCII 表示
		for i := 0; i < 16; i++ {
			bytePos := offset + i
			if bytePos >= len(data) {
				if bytePos < maxLen {
					sb.WriteString("?")
				}
				continue
			}

			if data[bytePos] >= 32 && data[bytePos] <= 126 {
				sb.WriteString(string(data[bytePos]))
			} else {
				sb.WriteString(".")
			}
		}

		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// 字段解析
	sb.WriteString("Field Parsing:\n")
	sb.WriteString("────────────────────────────────────────────────────────────────────────────\n\n")

	for _, fd := range fields {
		if fd.IsVar {
			sb.WriteString(fmt.Sprintf("%-15s (%-10s) @ 0x%04X : variable length (len:%s)\n",
				fd.Name, fd.Type, fd.Offset, fd.LenField))
			continue
		}

		// 检查数据是否足够
		if fd.Offset < 0 || fd.Offset+fd.Size > len(data) {
			missing := fd.Offset + fd.Size - len(data)
			sb.WriteString(fmt.Sprintf("❌ %-12s (%-10s) @ 0x%04X : !!! ERROR !!! insufficient data (missing %d bytes)\n",
				fd.Name, fd.Type, fd.Offset, missing))
			continue
		}

		// 提取字段数据
		fieldData := data[fd.Offset : fd.Offset+fd.Size]
		hexStr := hexlib.EncodeToString(fieldData)

		// 解析值
		valueStr := ""
		if fd.Bits != "" {
			// 位字段
			byteVal := fieldData[0]
			valueStr = fmt.Sprintf("0x%02X (bits:%s)", byteVal, fd.Bits)
		} else {
			// 普通字段
			switch fd.Size {
			case 1:
				valueStr = fmt.Sprintf("0x%02X (%d)", fieldData[0], fieldData[0])
			case 2:
				var val uint16
				if fd.ByteOrder == "le" {
					val = uint16(fieldData[0]) | uint16(fieldData[1])<<8
				} else {
					val = uint16(fieldData[1]) | uint16(fieldData[0])<<8
				}
				valueStr = fmt.Sprintf("0x%04X (%d)", val, val)
			case 4:
				var val uint32
				if fd.ByteOrder == "le" {
					val = uint32(fieldData[0]) | uint32(fieldData[1])<<8 |
						uint32(fieldData[2])<<16 | uint32(fieldData[3])<<24
				} else {
					val = uint32(fieldData[3]) | uint32(fieldData[2])<<8 |
						uint32(fieldData[1])<<16 | uint32(fieldData[0])<<24
				}
				valueStr = fmt.Sprintf("0x%08X (%d)", val, val)
			case 8:
				var val uint64
				if fd.ByteOrder == "le" {
					val = uint64(fieldData[0]) | uint64(fieldData[1])<<8 |
						uint64(fieldData[2])<<16 | uint64(fieldData[3])<<24 |
						uint64(fieldData[4])<<32 | uint64(fieldData[5])<<40 |
						uint64(fieldData[6])<<48 | uint64(fieldData[7])<<56
				} else {
					val = uint64(fieldData[7]) | uint64(fieldData[6])<<8 |
						uint64(fieldData[5])<<16 | uint64(fieldData[4])<<24 |
						uint64(fieldData[3])<<32 | uint64(fieldData[2])<<40 |
						uint64(fieldData[1])<<48 | uint64(fieldData[0])<<56
				}
				valueStr = fmt.Sprintf("0x%016X (%d)", val, val)
			default:
				valueStr = "0x" + hexStr
			}
		}

		endian := "BE"
		if fd.ByteOrder == "le" {
			endian = "LE"
		}

		sb.WriteString(fmt.Sprintf("%-15s (%-10s) @ 0x%04X [%2d bytes, %s] : %s\n",
			fd.Name, fd.Type, fd.Offset, fd.Size, endian, valueStr))
	}

	sb.WriteString("\n")

	// 字节映射图
	sb.WriteString("Byte Mapping:\n")
	sb.WriteString("────────────────────────────────────────────────────────────────────────────\n\n")

	// 创建字节到字段的映射
	byteMap := make([]string, len(data))
	for i := range byteMap {
		byteMap[i] = ""
	}

	for _, fd := range fields {
		if !fd.IsVar && fd.Offset >= 0 && fd.Offset < len(data) {
			endPos := fd.Offset + fd.Size
			if endPos > len(data) {
				endPos = len(data)
			}
			for i := fd.Offset; i < endPos; i++ {
				byteMap[i] = fd.Name
			}
		}
	}

	// 绘制映射图
	for offset := 0; offset < len(data); offset += 16 {
		sb.WriteString(fmt.Sprintf("%04X: ", offset))

		i := 0
		for i < 16 {
			bytePos := offset + i
			if bytePos >= len(data) {
				break
			}

			if byteMap[bytePos] != "" {
				// 检查是否是字段的第一个字节
				isFirst := (bytePos == 0) || (byteMap[bytePos] != byteMap[bytePos-1])
				if isFirst {
					fieldName := byteMap[bytePos]
					sb.WriteString("[" + fieldName)

					// 计算字段在本行的长度
					fieldLen := 0
					for j := bytePos; j < len(data) && j < offset+16 && byteMap[j] == fieldName; j++ {
						fieldLen++
					}

					// 检查字段是否在本行结束
					isLast := (bytePos+fieldLen >= len(data)) ||
					          (bytePos+fieldLen >= offset+16) ||
					          (byteMap[bytePos+fieldLen] != fieldName)

					if isLast {
						sb.WriteString("]")
					}

					i += fieldLen
				}
			} else {
				sb.WriteString("--")
				i++
			}

			if i < 16 && offset+i < len(data) {
				sb.WriteString(" ")
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}
