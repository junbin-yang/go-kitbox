package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// replacePathVars 替换路径模板变量
func replacePathVars(tpl string, vars map[string]string) string {
	result := tpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{."+k+"}}", v)
	}
	return result
}

// validateConfigPath 校验配置路径合法性
func validateConfigPath(path string) error {
	if path == "" {
		return errors.New("path is empty")
	}

	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", path)
		}
		return fmt.Errorf("stat path failed: %w", err)
	}

	if fi.IsDir() {
		return fmt.Errorf("path is a directory: %s", path)
	}

	return nil
}
