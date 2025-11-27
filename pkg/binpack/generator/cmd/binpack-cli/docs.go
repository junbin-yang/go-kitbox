package main

import (
	"flag"
	"fmt"
	"go/types"
	"os"
	"strings"

	"github.com/junbin-yang/go-kitbox/pkg/binpack/generator"
	"golang.org/x/tools/go/packages"
)

// FieldDoc 字段文档信息
type FieldDoc struct {
	Name      string
	Type      string
	Offset    int
	Size      int
	Tag       string
	ByteOrder string
	IsVar     bool
	LenField  string
	Bits      string
	IsRepeat  bool
}

func runDocs(args []string) {
	fs := flag.NewFlagSet("docs", flag.ExitOnError)
	pkg := fs.String("pkg", "", "包路径（如：./mypackage）")
	typ := fs.String("type", "", "结构体类型名")
	output := fs.String("output", "", "输出文件路径（可选，默认输出到标准输出）")

	fs.Usage = func() {
		fmt.Println("Usage: binpack docs -pkg <package> -type <struct> [-output <file>]")
		fmt.Println()
		fmt.Println("Generate protocol documentation with ASCII visualization.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  binpack docs -pkg ./mypackage -type Packet -output protocol.txt")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *pkg == "" || *typ == "" {
		fs.Usage()
		os.Exit(1)
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
	fields := make([]FieldDoc, 0)
	maxSize := 0

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

		fd := FieldDoc{
			Name:      field.Name(),
			Type:      typeName,
			Offset:    tagInfo.Offset,
			Size:      tagInfo.Size,
			Tag:       binTag,
			ByteOrder: tagInfo.ByteOrder,
			Bits:      tagInfo.Bits,
			IsRepeat:  tagInfo.IsRepeat,
		}

		if tagInfo.Size == -1 {
			fd.IsVar = true
			fd.LenField = tagInfo.LenField
		}

		fields = append(fields, fd)
		if !fd.IsVar && tagInfo.Offset >= 0 && tagInfo.Offset+tagInfo.Size > maxSize {
			maxSize = tagInfo.Offset + tagInfo.Size
		}
	}

	// 生成文档
	doc := generateDocs(*typ, fields, maxSize)

	// 输出
	if *output != "" {
		if err := os.WriteFile(*output, []byte(doc), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Documentation written to: %s\n", *output)
	} else {
		fmt.Print(doc)
	}
}

func generateDocs(typeName string, fields []FieldDoc, maxSize int) string {
	var sb strings.Builder

	// 标题
	sb.WriteString("╔═══════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString(fmt.Sprintf("║  Protocol Documentation: %-48s║\n", typeName))
	sb.WriteString("╚═══════════════════════════════════════════════════════════════════════════╝\n\n")

	// 字段表格
	sb.WriteString("Fields:\n")
	sb.WriteString("┌─────────────────┬──────────────┬────────┬──────┬──────────┬─────────────┐\n")
	sb.WriteString("│ Field Name      │ Type         │ Offset │ Size │ Endian   │ Options     │\n")
	sb.WriteString("├─────────────────┼──────────────┼────────┼──────┼──────────┼─────────────┤\n")

	for _, fd := range fields {
		sizeStr := fmt.Sprintf("%d", fd.Size)
		if fd.IsVar {
			sizeStr = "var"
		}

		endian := "BE"
		if fd.ByteOrder == "le" {
			endian = "LE"
		}

		options := ""
		if fd.LenField != "" {
			options = fmt.Sprintf("len:%s", fd.LenField)
		}
		if fd.Bits != "" {
			if options != "" {
				options += ","
			}
			options += fmt.Sprintf("bits:%s", fd.Bits)
		}
		if fd.IsRepeat {
			if options != "" {
				options += ","
			}
			options += "repeat"
		}

		offsetStr := fmt.Sprintf("%d", fd.Offset)
		if fd.Offset == -1 {
			offsetStr = "end"
		}

		// 截断过长的字段
		name := fd.Name
		if len(name) > 15 {
			name = name[:15]
		}
		typeName := fd.Type
		if len(typeName) > 12 {
			typeName = typeName[:12]
		}
		if len(options) > 11 {
			options = options[:11]
		}

		sb.WriteString(fmt.Sprintf("│ %-15s │ %-12s │ %-6s │ %-4s │ %-8s │ %-11s │\n",
			name, typeName, offsetStr, sizeStr, endian, options))
	}

	sb.WriteString("└─────────────────┴──────────────┴────────┴──────┴──────────┴─────────────┘\n\n")

	// 字节布局可视化
	sb.WriteString("Byte Layout:\n")
	sb.WriteString("────────────────────────────────────────────────────────────────────────────\n\n")

	if maxSize > 0 && maxSize < 256 {
		// 每行显示 16 字节
		for offset := 0; offset < maxSize; offset += 16 {
			// 偏移量标题
			sb.WriteString(fmt.Sprintf("%04X: ", offset))

			// 字节位置标记
			lineFields := make([]string, 16)
			for i := 0; i < 16; i++ {
				bytePos := offset + i
				if bytePos >= maxSize {
					break
				}

				// 查找该字节属于哪个字段
				fieldName := ""
				for _, fd := range fields {
					if !fd.IsVar && fd.Offset >= 0 && bytePos >= fd.Offset && bytePos < fd.Offset+fd.Size {
						fieldName = fd.Name
						break
					}
				}
				lineFields[i] = fieldName
			}

			// 绘制字节格子
			for i := 0; i < 16; i++ {
				bytePos := offset + i
				if bytePos >= maxSize {
					sb.WriteString("   ")
					continue
				}

				if lineFields[i] != "" {
					// 检查是否是字段起始
					isStart := (i == 0) || (lineFields[i] != lineFields[i-1])
					if isStart {
						sb.WriteString("[")
					} else {
						sb.WriteString(" ")
					}
					sb.WriteString("██")
					// 检查是否是字段结束
					isEnd := (i == 15) || (bytePos+1 >= maxSize) || (lineFields[i] != lineFields[i+1])
					if isEnd {
						sb.WriteString("]")
					} else {
						sb.WriteString(" ")
					}
				} else {
					sb.WriteString(" -- ")
				}
			}

			sb.WriteString(" │ ")

			// 字段名称标注
			printed := false
			for i := 0; i < 16; i++ {
				if lineFields[i] != "" {
					isStart := (i == 0) || (lineFields[i] != lineFields[i-1])
					if isStart {
						sb.WriteString(lineFields[i])
						printed = true
						break
					}
				}
			}
			if !printed {
				sb.WriteString("-")
			}

			sb.WriteString("\n")
		}
	} else if maxSize >= 256 {
		sb.WriteString("(Layout too large to display, total size: ")
		sb.WriteString(fmt.Sprintf("%d bytes)\n", maxSize))
	}

	sb.WriteString("\n")

	// Tag 语法说明
	sb.WriteString("Tag Syntax:\n")
	sb.WriteString("────────────────────────────────────────────────────────────────────────────\n\n")

	for _, fd := range fields {
		sb.WriteString(fmt.Sprintf("%-20s `bin:\"%s\"`\n", fd.Name+":", fd.Tag))
	}

	sb.WriteString("\n")

	// 示例数据
	sb.WriteString("Example Data (hexadecimal):\n")
	sb.WriteString("────────────────────────────────────────────────────────────────────────────\n\n")

	if maxSize > 0 && maxSize < 256 {
		for offset := 0; offset < maxSize; offset += 16 {
			sb.WriteString(fmt.Sprintf("%04X: ", offset))
			for i := 0; i < 16 && offset+i < maxSize; i++ {
				sb.WriteString("00 ")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
