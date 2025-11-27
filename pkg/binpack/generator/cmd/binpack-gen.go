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

var (
	pkg    = flag.String("pkg", "", "包路径（如：./mypackage）")
	typ    = flag.String("type", "", "结构体类型名")
	output = flag.String("output", "", "输出文件路径")
)

func main() {
	flag.Parse()

	if *pkg == "" || *typ == "" {
		fmt.Println("Usage: binpack-gen -pkg <package> -type <struct> [-output <file>]")
		fmt.Println("\nExample:")
		fmt.Println("  binpack-gen -pkg ./mypackage -type Packet -output packet_gen.go")
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

	// 转换为 reflect.Type（通过结构体信息）
	structType, ok := targetType.Underlying().(*types.Struct)
	if !ok {
		fmt.Fprintf(os.Stderr, "Type %s is not a struct\n", *typ)
		os.Exit(1)
	}

	// 构建 fieldInfo 用于生成
	fields := make([]generator.FieldInfo, 0)
	maxSize := 0

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		binTag := ""
		// 解析 struct tag
		if tag != "" {
			// 简单解析 bin tag
			for j := 0; j < len(tag); j++ {
				if j+4 < len(tag) && tag[j:j+4] == "bin:" {
					start := j + 5 // 跳过 bin:"
					end := start
					for end < len(tag) && tag[end] != '"' {
						end++
					}
					binTag = tag[start:end]
					break
				}
			}
		}

		if binTag == "" || binTag == "-" {
			continue
		}

		// 解析 tag
		tagInfo, err := generator.ParseTagForGen(binTag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse tag for field %s: %v\n", field.Name(), err)
			os.Exit(1)
		}

		// 简化类型名（去掉包名）
		typeName := field.Type().String()
		if basic, ok := field.Type().(*types.Basic); ok {
			typeName = basic.Name()
		}

		fi := generator.FieldInfo{
			Name:   field.Name(),
			Type:   typeName,
			Offset: tagInfo.Offset,
			Size:   tagInfo.Size,
		}

		if tagInfo.ByteOrder == "le" {
			fi.ByteOrder = "binary.LittleEndian"
		} else {
			fi.ByteOrder = "binary.BigEndian"
		}

		if tagInfo.Size == -1 {
			fi.IsVar = true
			fi.LenField = tagInfo.LenField
		}

		fields = append(fields, fi)
		if !fi.IsVar && tagInfo.Offset+tagInfo.Size > maxSize {
			maxSize = tagInfo.Offset + tagInfo.Size
		}
	}

	// 生成代码（使用原包名）
	pkgName := pkgs[0].Name
	if pkgName == "" {
		// 从 ID 提取包名
		pkgName = pkgs[0].ID
		if idx := strings.LastIndex(pkgName, "/"); idx >= 0 {
			pkgName = pkgName[idx+1:]
		}
	}
	code, err := generator.GenerateFromFields(*typ, pkgName, fields, maxSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate code: %v\n", err)
		os.Exit(1)
	}

	// 输出
	if *output != "" {
		if err := os.WriteFile(*output, code, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated code written to: %s\n", *output)
	} else {
		fmt.Println(string(code))
	}
}
