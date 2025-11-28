package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/junbin-yang/go-kitbox/pkg/binpack"
)

func runValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	pkgPath := fs.String("pkg", ".", "Package path to validate")
	typeName := fs.String("type", "", "Specific type name to validate (optional)")

	fs.Usage = func() {
		fmt.Println("Usage: binpack validate [options]")
		fmt.Println()
		fmt.Println("Validate struct tags in Go source files")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  binpack validate -pkg ./protocol")
		fmt.Println("  binpack validate -pkg ./protocol -type Packet")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if err := validatePackage(*pkgPath, *typeName); err != nil {
		fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ All struct tags are valid")
}

func validatePackage(pkgPath, typeName string) error {
	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, pkgPath, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		return fmt.Errorf("failed to parse package: %w", err)
	}

	var errors []error

	for _, pkg := range pkgs {
		for filename, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				typeSpec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					return true
				}

				// 如果指定了类型名，只验证该类型
				if typeName != "" && typeSpec.Name.Name != typeName {
					return true
				}

				// 验证结构体标签
				if err := validateStructTags(typeSpec.Name.Name, structType, filename); err != nil {
					errors = append(errors, err)
				}

				return true
			})
		}
	}

	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "  %v\n", err)
		}
		return fmt.Errorf("found %d validation error(s)", len(errors))
	}

	return nil
}

func validateStructTags(structName string, structType *ast.StructType, filename string) error {
	fieldMap := make(map[string]int)
	offsetMap := make(map[int]string)
	lenFields := make(map[string]bool)
	condFields := make(map[string]bool)

	// 第一遍：收集字段信息
	for i, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name
		fieldMap[fieldName] = i

		if field.Tag == nil {
			continue
		}

		tag := strings.Trim(field.Tag.Value, "`")
		binTag := extractBinTag(tag)
		if binTag == "" || binTag == "-" {
			continue
		}

		info, err := binpack.ParseTag(binTag)
		if err != nil {
			return fmt.Errorf("%s: struct %s, field %s: invalid tag: %v",
				filepath.Base(filename), structName, fieldName, err)
		}

		// 检查偏移量冲突
		if info.Offset >= 0 {
			if existingField, exists := offsetMap[info.Offset]; exists {
				return fmt.Errorf("%s: struct %s, field %s: offset %d conflicts with field %s",
					filepath.Base(filename), structName, fieldName, info.Offset, existingField)
			}
			offsetMap[info.Offset] = fieldName
		}

		// 收集依赖字段
		if info.LenField != "" {
			lenFields[info.LenField] = true
		}
		if info.Condition != "" {
			if condField := extractConditionField(info.Condition); condField != "" {
				condFields[condField] = true
			}
		}
	}

	// 第二遍：验证依赖字段
	for i, field := range structType.Fields.List {
		if len(field.Names) == 0 || field.Tag == nil {
			continue
		}

		fieldName := field.Names[0].Name
		tag := strings.Trim(field.Tag.Value, "`")
		binTag := extractBinTag(tag)
		if binTag == "" || binTag == "-" {
			continue
		}

		info, _ := binpack.ParseTag(binTag)

		// 验证长度字段
		if info.LenField != "" {
			if _, exists := fieldMap[info.LenField]; !exists {
				return fmt.Errorf("%s: struct %s, field %s: length field %s does not exist",
					filepath.Base(filename), structName, fieldName, info.LenField)
			}
			if fieldMap[info.LenField] >= i {
				return fmt.Errorf("%s: struct %s, field %s: length field %s must appear before this field",
					filepath.Base(filename), structName, fieldName, info.LenField)
			}
		}

		// 验证条件字段
		if info.Condition != "" {
			condField := extractConditionField(info.Condition)
			if condField != "" {
				if _, exists := fieldMap[condField]; !exists {
					return fmt.Errorf("%s: struct %s, field %s: condition field %s does not exist",
						filepath.Base(filename), structName, fieldName, condField)
				}
				if fieldMap[condField] >= i {
					return fmt.Errorf("%s: struct %s, field %s: condition field %s must appear before this field",
						filepath.Base(filename), structName, fieldName, condField)
				}
			}
		}

		// 验证变长字段
		if info.Size == -1 && info.LenField == "" && !info.IsRepeat {
			return fmt.Errorf("%s: struct %s, field %s: variable-length field must specify len field",
				filepath.Base(filename), structName, fieldName)
		}

		// 验证数组字段
		if info.IsRepeat && info.LenField == "" {
			return fmt.Errorf("%s: struct %s, field %s: repeat field must specify len field",
				filepath.Base(filename), structName, fieldName)
		}
	}

	return nil
}


