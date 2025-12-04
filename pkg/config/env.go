package config

import (
	"os"
	"reflect"
	"strconv"
)

// applyEnvOverrides 应用环境变量覆盖
func applyEnvOverrides(v interface{}) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return
	}
	applyEnvToStruct(val.Elem())
}

func applyEnvToStruct(val reflect.Value) {
	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanSet() {
			continue
		}

		envKey := fieldType.Tag.Get("env")
		if envKey != "" {
			if envVal := os.Getenv(envKey); envVal != "" {
				setFieldValue(field, envVal)
			}
		}

		if field.Kind() == reflect.Struct {
			applyEnvToStruct(field)
		}
	}
}

func setFieldValue(field reflect.Value, value string) {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			field.SetInt(intVal)
		}
	case reflect.Bool:
		if boolVal, err := strconv.ParseBool(value); err == nil {
			field.SetBool(boolVal)
		}
	}
}
