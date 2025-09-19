package chttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type ParserResult int

const (
	ParserResultSuccess     ParserResult = 1
	ParserResultNotVerified ParserResult = 0
	ParserResultError       ParserResult = -1
)

type ParamValidation struct {
	Valid        *bool
	ValidMessage *string
}

func Valid[T any](r *http.Request) (T, ParserResult, error) {
	req, validation, err := ParseWithValidation[T](r)
	if err != nil {
		return req, ParserResultError, err
	} else if validation.Valid == nil || *validation.Valid == false {
		return req, ParserResultNotVerified, fmt.Errorf(*validation.ValidMessage)
	}
	return req, ParserResultSuccess, nil
}

func ReadRequestBody[T any](r *http.Request) (*T, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	defer r.Body.Close()
	var t T
	err = json.Unmarshal(body, &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func ParseWithValidation[T any](r *http.Request) (T, *ParamValidation, error) {
	var result T
	var validationMsg string
	var vCompleted = false

	// 用于跟踪哪些字段已经被显式设置过（包括JSON和URL参数等）
	var explicitlySetFields map[string]bool

	switch r.Method {
	case http.MethodGet:
		explicitlySetFields = make(map[string]bool)
		err := parseRequestParams(r, &result, explicitlySetFields)
		if err != nil {
			return result, &ParamValidation{Valid: &vCompleted, ValidMessage: &validationMsg}, errors.New("Invalid request params")
		}
	default:
		explicitlySetFields = make(map[string]bool)
		if contentType := r.Header.Get("Content-Type"); !strings.Contains(contentType, "multipart/form-data") {
			if r.Body != nil {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					return result, nil, errors.Wrap(err, "Read body error")
				}
				// 使用缓冲区创建新的可重复读取的请求体
				r.Body = io.NopCloser(bytes.NewBuffer(body))

				// 延迟关闭请求体
				defer r.Body.Close()

				// 如果需要解析JSON数据，可以从缓冲区重新读取
				if len(body) > 0 {
					// 先解析JSON为map来检测哪些键存在
					var jsonMap map[string]interface{}
					if err := json.Unmarshal(body, &jsonMap); err != nil {
						return result, nil, errors.Wrap(err, "body is not json")
					}

					// 根据JSON中存在的键标记字段
					markJSONKeys(&result, jsonMap, explicitlySetFields, "")

					// 先尝试正常解析JSON到结构体
					if err := json.NewDecoder(bytes.NewBuffer(body)).Decode(&result); err != nil {
						// 如果解析失败，可能是时间字段格式问题，尝试灵活解析
						if err := parseJSONWithFlexibleTime(&result, jsonMap); err != nil {
							return result, nil, errors.Wrap(err, "body is not json")
						}
					} else {
						// 如果正常解析成功，对时间字段进行灵活解析
						if err := parseTimeFieldsFromJSON(&result, jsonMap); err != nil {
							return result, nil, errors.Wrap(err, "failed to parse time fields")
						}
					}
				}
			}
		}
		err := parseRequestParams(r, &result, explicitlySetFields)
		if err != nil {
			return result, nil, errors.Wrap(err, "Invalid request params")
		}

		// URL参数拥有最高优先级，可以覆盖包括JSON在内的所有其他值
		err = overrideWithURLParams(r, &result)
		if err != nil {
			return result, nil, errors.Wrap(err, "Invalid URL params")
		}
	}
	validate := validator.New()
	validate.SetTagName("v")
	err := validate.Struct(result)
	if err != nil {
		// 验证失败，打印错误信息
		for _, err := range err.(validator.ValidationErrors) {
			// 将错误信息拼接成一个
			validationMsg += fmt.Sprintf("%s,", err.Error())
		}
		vCompleted = false
	} else {
		vCompleted = true
	}
	return result, &ParamValidation{Valid: &vCompleted, ValidMessage: &validationMsg}, nil
}

// overrideWithURLParams 专门处理URL路径参数，拥有最高优先级
func overrideWithURLParams(r *http.Request, arg interface{}) error {
	v := reflect.ValueOf(arg).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		urlTag := fieldType.Tag.Get("url")
		pTag := fieldType.Tag.Get("cv")

		// struct 类型递归处理
		if pTag != "" && field.Kind() == reflect.Struct && field.CanSet() && field.CanInterface() {
			subErr := overrideWithURLParams(r, field.Addr().Interface())
			if subErr != nil {
				return subErr
			}
			continue
		}

		// 指针类型递归处理
		if pTag != "" && field.Kind() == reflect.Pointer && field.CanSet() && field.CanInterface() {
			if field.Type().Elem().Kind() == reflect.Struct {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
				subErr := overrideWithURLParams(r, field.Interface())
				if subErr != nil {
					return subErr
				}
				continue
			}
		}

		// 处理URL路径参数
		if urlTag != "" && field.CanSet() {
			urlTag = strings.Split(urlTag, ",")[0]
			urlValue := chi.URLParam(r, urlTag)
			// URL参数如果存在就使用，即使是空字符串
			// 注意：chi.URLParam对于不存在的参数返回空字符串，这里无法区分
			// 但通常URL路径参数如果存在就应该有值
			if urlValue != "" {
				if err := setFieldValue(field, urlValue); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// markJSONKeys 根据JSON中存在的键标记字段
func markJSONKeys(structPtr interface{}, jsonMap map[string]interface{}, explicitlySetFields map[string]bool, prefix string) {
	structValue := reflect.ValueOf(structPtr).Elem()
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)
		fieldName := fieldType.Name

		// 获取json标签，如果没有就使用字段名
		jsonTag := fieldType.Tag.Get("json")
		jsonFieldName := fieldName
		if jsonTag != "" && jsonTag != "-" {
			// 处理 "fieldname,omitempty" 格式
			jsonFieldName = strings.Split(jsonTag, ",")[0]
		}

		fullFieldName := fieldName
		if prefix != "" {
			fullFieldName = prefix + "." + fieldName
		}

		// 检查JSON中是否存在这个键
		if _, exists := jsonMap[jsonFieldName]; exists {
			explicitlySetFields[fullFieldName] = true
		}

		// 如果是嵌套结构体，递归处理
		if field.Kind() == reflect.Struct {
			if nestedMap, ok := jsonMap[jsonFieldName].(map[string]interface{}); ok {
				markJSONKeys(field.Addr().Interface(), nestedMap, explicitlySetFields, fullFieldName)
			}
		}
	}
}

// markExplicitlySetFields 标记哪些字段被显式设置了（JSON等）
func markExplicitlySetFields(original, current interface{}, explicitlySetFields map[string]bool, prefix string) {
	originalValue := reflect.ValueOf(original).Elem()
	currentValue := reflect.ValueOf(current).Elem()

	for i := 0; i < originalValue.NumField(); i++ {
		fieldName := originalValue.Type().Field(i).Name
		fullFieldName := fieldName
		if prefix != "" {
			fullFieldName = prefix + "." + fieldName
		}

		originalField := originalValue.Field(i)
		currentField := currentValue.Field(i)

		// 如果是嵌套结构体，递归检查
		if originalField.Kind() == reflect.Struct && currentField.Kind() == reflect.Struct {
			markExplicitlySetFields(originalField.Addr().Interface(), currentField.Addr().Interface(), explicitlySetFields, fullFieldName)
			continue
		}

		// 比较字段值是否发生变化
		if !reflect.DeepEqual(originalField.Interface(), currentField.Interface()) {
			explicitlySetFields[fullFieldName] = true
		}
	}
}

// bool validation
// string validation failed message

// parseRequestParamsWithValidation
// error error
func parseRequestParams(r *http.Request, arg interface{}, explicitlySetFields map[string]bool) error {
	return parseRequestParamsWithPrefix(r, arg, explicitlySetFields, "")
}

func parseRequestParamsWithPrefix(r *http.Request, arg interface{}, explicitlySetFields map[string]bool, prefix string) error {
	values := r.URL.Query()
	headers := r.Header
	v := reflect.ValueOf(arg).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name

		// Create namespaced field name
		fullFieldName := fieldName
		if prefix != "" {
			fullFieldName = prefix + "." + fieldName
		}

		urlTag := v.Type().Field(i).Tag.Get("url")
		paramTag := v.Type().Field(i).Tag.Get("param")
		headerTag := v.Type().Field(i).Tag.Get("header")
		pTag := v.Type().Field(i).Tag.Get("cv")
		defaultTag := v.Type().Field(i).Tag.Get("default")
		rawJsonTag := v.Type().Field(i).Tag.Get("rawJson")

		// 按优先级收集所有可能的值：URL Param > Header > Query Param
		var value string
		var hasValue bool

		// 最低优先级：Query Param
		if paramTag != "" && values.Has(paramTag) {
			value = values.Get(paramTag)
			hasValue = true
		}

		// 中等优先级：Header（可以覆盖Query参数）
		if headerTag != "" {
			headerTag = strings.Split(headerTag, ",")[0]
			headerValue := headers.Get(headerTag) // Get方法内部已处理大小写
			if headerValue != "" {
				value = headerValue
				hasValue = true
			}
		}

		// 最高优先级：URL路径参数（可以覆盖Header和Query参数）
		if urlTag != "" {
			urlTag = strings.Split(urlTag, ",")[0]
			urlValue := chi.URLParam(r, urlTag)
			// URL参数如果存在就使用，即使是空字符串靠, 列表,
			// 注意：chi.URLParam对于不存在的参数返回空字符串，这里无法区分
			// 但通常URL路径参数如果存在就应该有值
			if urlValue != "" {
				value = urlValue
				hasValue = true
			}
		}
		// struct 类型, 判断是否往下层递归
		if pTag != "" && field.Kind() == reflect.Struct && field.CanSet() && field.CanInterface() {
			subErr := parseRequestParamsWithPrefix(r, field.Addr().Interface(), explicitlySetFields, fullFieldName)
			if subErr != nil {
				return subErr
			}
			continue
		}
		// 指针类型
		if pTag != "" && field.Kind() == reflect.Pointer && field.CanSet() && field.CanInterface() {
			if field.Type().Elem().Kind() == reflect.Struct {
				// 如果指针为 nil，则初始化它
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
				// 递归解析嵌入字段
				subErr := parseRequestParamsWithPrefix(r, field.Interface(), explicitlySetFields, fullFieldName)
				if subErr != nil {
					return subErr
				}
				continue
			}
		}
		if hasValue && field.CanSet() {
			// 只有当字段没有被JSON等更高优先级的方式设置时才设置值
			if explicitlySetFields == nil || !explicitlySetFields[fullFieldName] {
				// 记录这个字段被显式设置了
				if explicitlySetFields != nil {
					explicitlySetFields[fullFieldName] = true
				}
				if err := setFieldValue(field, value); err != nil {
					return err
				}
			}
		} else if defaultTag != "" && field.CanSet() && !hasValue {
			// 只有当字段没有被显式设置时才应用默认值
			if explicitlySetFields == nil || !explicitlySetFields[fullFieldName] {
				if err := setFieldValue(field, defaultTag); err != nil {
					return err
				}
			}
		}
		if rawJsonTag != "" && field.CanSet() {
			rawJsonTag = strings.Split(rawJsonTag, ",")[0]
			// 遍历map, 并找到对应的field
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).Name == rawJsonTag {
					// 获取对应的field
					sourceField := v.Field(i)
					// 检查字段是否为字符串或者是指针的字符串
					if sourceField.Kind() == reflect.String {
						// 获取对应的field的值
						var variable interface{}
						if field.Kind() == reflect.Ptr {
							// 目标字段是指针：创建其元素类型的指针 *T
							variable = reflect.New(field.Type().Elem()).Interface()
						} else {
							// 目标字段是值类型：创建 *T
							variable = reflect.New(field.Type()).Interface()
						}
						value := sourceField.String()
						// 按field的类型,解析json并设置值
						if err := json.Unmarshal([]byte(value), variable); err != nil {
							return err
						}
						if field.Kind() == reflect.Ptr {
							field.Set(reflect.ValueOf(variable))
						} else {
							field.Set(reflect.ValueOf(variable).Elem())
						}
					} else if sourceField.Kind() == reflect.Ptr && sourceField.Elem().Kind() == reflect.String {
						// 获取对应的field的值
						var variable interface{}
						if field.Kind() == reflect.Ptr {
							variable = reflect.New(field.Type().Elem()).Interface()
						} else {
							variable = reflect.New(field.Type()).Interface()
						}
						value := sourceField.Elem().String()
						if err := json.Unmarshal([]byte(value), variable); err != nil {
							return err
						}
						if field.Kind() == reflect.Ptr {
							field.Set(reflect.ValueOf(variable))
						} else {
							field.Set(reflect.ValueOf(variable).Elem())
						}
					}
				}
			}
		}
	}
	return nil
}

func checkIfNull(field reflect.Value, fieldType reflect.StructField) bool {
	// 检查字段是否为指针类型
	if field.Kind() == reflect.Ptr {
		// 对于指针类型，只检查指针是否为 nil
		// 如果指针不为nil，说明已经被显式设置过了，不管指向的值是什么
		return field.IsNil()
	} else {
		// 检查非指针类型的字段是否有值
		if field.Kind() == reflect.String && field.String() == "" {
			return true
		} else if field.Kind() == reflect.Int && field.Int() == 0 {
			return true
		} else if field.IsZero() {
			return true
		}
	}
	return false
}

func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Ptr:
		if field.IsNil() {
			// 创建一个新的指针，并设置为默认值
			rv := field.Type().Elem()
			field.Set(reflect.New(rv))
			field = field.Elem()
			// 检查新创建的指针的元素是否仍然是指针类型
			if field.Kind() == reflect.Ptr {
				return fmt.Errorf("cannot set nested pointer field: %v", field.Type())
			}
		} else {
			field = field.Elem()
		}
		return setFieldValue(field, value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	case reflect.Struct:
		if field.Type().PkgPath() == "time" && field.Type().Name() == "Time" {
			parsed, err := parseFlexibleTime(value)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(parsed))
			return nil
		}
		return fmt.Errorf("unsupported struct type: %v", field.Type())
	default:
		// 处理 *time.Time
		if field.Type().PkgPath() == "time" && field.Type().Name() == "Time" {
			parsed, err := parseFlexibleTime(value)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(parsed))
			return nil
		}
		if field.Type().Kind() == reflect.Ptr && field.Type().Elem().PkgPath() == "time" && field.Type().Elem().Name() == "Time" {
			parsed, err := parseFlexibleTime(value)
			if err != nil {
				return err
			}
			field.Set(reflect.New(field.Type().Elem()))
			field.Elem().Set(reflect.ValueOf(parsed))
			return nil
		}
		return fmt.Errorf("unsupported field type: %v with default value", field.Type())
	}
	return nil
}

// parseJSONWithFlexibleTime 使用灵活时间解析解析JSON
func parseJSONWithFlexibleTime(result interface{}, jsonMap map[string]interface{}) error {
	return parseJSONWithFlexibleTimeWithPrefix(result, jsonMap, "")
}

func parseJSONWithFlexibleTimeWithPrefix(result interface{}, jsonMap map[string]interface{}, prefix string) error {
	v := reflect.ValueOf(result).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name

		// 获取json标签
		jsonTag := fieldType.Tag.Get("json")
		jsonFieldName := fieldName
		if jsonTag != "" && jsonTag != "-" {
			jsonFieldName = strings.Split(jsonTag, ",")[0]
		}

		fullFieldName := fieldName
		if prefix != "" {
			fullFieldName = prefix + "." + fieldName
		}

		// 处理时间字段
		if fieldType.Type.PkgPath() == "time" && fieldType.Type.Name() == "Time" {
			if jsonValue, exists := jsonMap[jsonFieldName]; exists {
				if timeStr, ok := jsonValue.(string); ok {
					parsed, err := parseFlexibleTime(timeStr)
					if err == nil {
						field.Set(reflect.ValueOf(parsed))
					}
				}
			}
		} else if fieldType.Type.Kind() == reflect.Ptr && fieldType.Type.Elem().PkgPath() == "time" && fieldType.Type.Elem().Name() == "Time" {
			if jsonValue, exists := jsonMap[jsonFieldName]; exists {
				if timeStr, ok := jsonValue.(string); ok {
					parsed, err := parseFlexibleTime(timeStr)
					if err == nil {
						field.Set(reflect.New(fieldType.Type.Elem()))
						field.Elem().Set(reflect.ValueOf(parsed))
					}
				}
			}
		} else if field.Kind() == reflect.Struct {
			// 递归处理嵌套结构体
			if nestedMap, ok := jsonMap[jsonFieldName].(map[string]interface{}); ok {
				if err := parseJSONWithFlexibleTimeWithPrefix(field.Addr().Interface(), nestedMap, fullFieldName); err != nil {
					return err
				}
			}
		} else if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			// 仅当对应的 JSON 键存在且为对象时，才初始化并递归
			if nestedMap, ok := jsonMap[jsonFieldName].(map[string]interface{}); ok {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
				if err := parseJSONWithFlexibleTimeWithPrefix(field.Interface(), nestedMap, fullFieldName); err != nil {
					return err
				}
			}
		} else {
			// 处理其他类型字段
			if jsonValue, exists := jsonMap[jsonFieldName]; exists {
				if err := setFieldValueFromJSON(field, jsonValue); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// setFieldValueFromJSON 从JSON值设置字段值
func setFieldValueFromJSON(field reflect.Value, value interface{}) error {
	if !field.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	switch field.Kind() {
	case reflect.String:
		if str, ok := value.(string); ok {
			field.SetString(str)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if num, ok := value.(float64); ok {
			field.SetInt(int64(num))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if num, ok := value.(float64); ok {
			field.SetUint(uint64(num))
		}
	case reflect.Float32, reflect.Float64:
		if num, ok := value.(float64); ok {
			field.SetFloat(num)
		}
	case reflect.Bool:
		if b, ok := value.(bool); ok {
			field.SetBool(b)
		}
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldValueFromJSON(field.Elem(), value)
	}

	return nil
}

// parseTimeFieldsFromJSON 从JSON中解析时间字段
func parseTimeFieldsFromJSON(result interface{}, jsonMap map[string]interface{}) error {
	return parseTimeFieldsFromJSONWithPrefix(result, jsonMap, "")
}

func parseTimeFieldsFromJSONWithPrefix(result interface{}, jsonMap map[string]interface{}, prefix string) error {
	v := reflect.ValueOf(result).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name

		// 获取json标签
		jsonTag := fieldType.Tag.Get("json")
		jsonFieldName := fieldName
		if jsonTag != "" && jsonTag != "-" {
			jsonFieldName = strings.Split(jsonTag, ",")[0]
		}

		fullFieldName := fieldName
		if prefix != "" {
			fullFieldName = prefix + "." + fieldName
		}

		// 处理时间字段
		if fieldType.Type.PkgPath() == "time" && fieldType.Type.Name() == "Time" {
			if jsonValue, exists := jsonMap[jsonFieldName]; exists {
				if timeStr, ok := jsonValue.(string); ok {
					parsed, err := parseFlexibleTime(timeStr)
					if err == nil {
						field.Set(reflect.ValueOf(parsed))
					}
				}
			}
		} else if fieldType.Type.Kind() == reflect.Ptr && fieldType.Type.Elem().PkgPath() == "time" && fieldType.Type.Elem().Name() == "Time" {
			if jsonValue, exists := jsonMap[jsonFieldName]; exists {
				if timeStr, ok := jsonValue.(string); ok {
					parsed, err := parseFlexibleTime(timeStr)
					if err == nil {
						field.Set(reflect.New(fieldType.Type.Elem()))
						field.Elem().Set(reflect.ValueOf(parsed))
					}
				}
			}
		} else if field.Kind() == reflect.Struct {
			// 递归处理嵌套结构体
			if nestedMap, ok := jsonMap[jsonFieldName].(map[string]interface{}); ok {
				if err := parseTimeFieldsFromJSONWithPrefix(field.Addr().Interface(), nestedMap, fullFieldName); err != nil {
					return err
				}
			}
		} else if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			// 仅当对应的 JSON 键存在且为对象时，才初始化并递归
			if nestedMap, ok := jsonMap[jsonFieldName].(map[string]interface{}); ok {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
				if err := parseTimeFieldsFromJSONWithPrefix(field.Interface(), nestedMap, fullFieldName); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// parseFlexibleTime 支持多种常见时间格式和时间戳
func parseFlexibleTime(value string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02",
		"2006.01.02 15:04:05",
		"2006.01.02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, value); err == nil {
			return t, nil
		}
	}
	// 支持时间戳（秒/毫秒）
	if ts, err := strconv.ParseInt(value, 10, 64); err == nil {
		if len(value) > 10 {
			// 毫秒
			return time.UnixMilli(ts), nil
		}
		return time.Unix(ts, 0), nil
	}
	return time.Time{}, fmt.Errorf("无法解析时间格式: %s", value)
}
