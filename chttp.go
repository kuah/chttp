package chttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
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
	switch r.Method {
	case http.MethodGet:
		validated, validateMsg, err := parseRequestParamsWithValidation(r, &result)
		if err != nil {
			return result, nil, errors.New("Invalid request params")
		}
		vCompleted = validated
		validationMsg = validateMsg
	default:
		if contentType := r.Header.Get("Content-Type"); !strings.Contains(contentType, "multipart/form-data") {
			if r.Body != nil {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					return result, nil, errors.Wrap(err, "Read body error")
				}
				// 重置请求体
				r.Body = io.NopCloser(bytes.NewBuffer(body))
				if len(body) > 0 {
					if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
						return result, nil, errors.Wrap(err, "body is not json")
					}
				}
			}
		}
		validated, validateMsg, err := parseRequestParamsWithValidation(r, &result)
		if err != nil {
			return result, nil, errors.Wrap(err, "Invalid request params")
		}
		vCompleted = validated
		validationMsg = validateMsg
	}
	return result, &ParamValidation{Valid: &vCompleted, ValidMessage: &validationMsg}, nil
}

// parseRequestParamsWithValidation
// bool validation
// string validation failed message
// error error
func parseRequestParamsWithValidation(r *http.Request, arg interface{}) (bool, string, error) {
	values := r.URL.Query()
	headers := r.Header
	v := reflect.ValueOf(arg).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		urlTag := v.Type().Field(i).Tag.Get("url")
		paramTag := v.Type().Field(i).Tag.Get("param")
		headerTag := v.Type().Field(i).Tag.Get("header")
		validationTag := v.Type().Field(i).Tag.Get("v")
		pTag := v.Type().Field(i).Tag.Get("cv")
		defaultTag := v.Type().Field(i).Tag.Get("default")
		fieldName := v.Type().Field(i).Name
		rawJsonTag := v.Type().Field(i).Tag.Get("rawJson")

		var value string
		if paramTag != "" {
			value = values.Get(paramTag)
		}
		if value == "" && urlTag != "" {
			urlTag = strings.Split(urlTag, ",")[0]
			vs := r.URL.Query()
			value = vs.Get(urlTag)
		}
		if value == "" && headerTag != "" {
			headerTag = strings.Split(headerTag, ",")[0]
			value = headers.Get(headerTag)
		}
		// struct 类型, 判断是否往下层递归
		if pTag != "" && field.Kind() == reflect.Struct && field.CanSet() && field.CanInterface() {
			subValid, subField, subErr := parseRequestParamsWithValidation(r, field.Addr().Interface())
			if !subValid {
				return false, subField, subErr
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
				subValid, subField, subErr := parseRequestParamsWithValidation(r, field.Interface())
				if !subValid {
					return false, subField, subErr
				}
				continue
			}
		}
		if value != "" && field.CanSet() {
			if err := setFieldValue(field, value); err != nil {
				return false, "", err
			}
		} else if defaultTag != "" && field.CanSet() && checkIfNull(field, fieldType) {
			if err := setFieldValue(field, defaultTag); err != nil {
				return false, "", err
			}
		} else if strings.Contains(validationTag, "required") && field.CanSet() && checkIfNull(field, fieldType) {
			valPos := ""
			switch {
			case headerTag != "":
				valPos = "header"
			case urlTag != "":
				valPos = "url"
			case paramTag != "":
				valPos = "param"
			}
			// 返回field的名字
			return false, fmt.Sprintf("%s %s is required", valPos, fieldName), nil
		}

		if rawJsonTag != "" && field.CanSet() && checkIfNull(field, fieldType) {
			rawJsonTag = strings.Split(rawJsonTag, ",")[0]
			// 遍历map, 并找到对应的field
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).Name == rawJsonTag {
					// 获取对应的field
					sourceField := v.Field(i)
					// 检查字段是否为字符串或者是指针的字符串
					if sourceField.Kind() == reflect.String {
						// 获取对应的field的值
						variable := reflect.New(field.Type()).Interface()
						value := sourceField.String()
						// 按field的类型,解析json并设置值
						if err := json.Unmarshal([]byte(value), variable); err != nil {
							return false, "", err
						}
						field.Set(reflect.ValueOf(variable).Elem())
					} else if sourceField.Kind() == reflect.Ptr && sourceField.Elem().Kind() == reflect.String {
						// 获取对应的field的值
						variable := reflect.New(field.Type()).Interface()
						value := sourceField.Elem().String()
						if err := json.Unmarshal([]byte(value), variable); err != nil {
							return false, "", err
						}
						field.Set(reflect.ValueOf(variable).Elem())
					}
				}
			}
		}
	}
	return true, "", nil
}

func checkIfNull(field reflect.Value, fieldType reflect.StructField) bool {
	// 检查字段是否为指针类型
	if field.Kind() == reflect.Ptr {
		// 检查指针是否为 nil
		if field.IsNil() {
			fmt.Printf("Field %s is required but is nil.\n", fieldType.Name)
			return true
		} else {
			// 获取指针指向的值
			elem := field.Elem()
			if elem.Kind() == reflect.String && elem.String() == "" {
				fmt.Printf("Field %s is required but has no value.\n", fieldType.Name)
				return true
			} else if elem.Kind() == reflect.Int && elem.Int() == 0 {
				fmt.Printf("Field %s is required but has no value.\n", fieldType.Name)
				return true
			} else if elem.IsZero() {
				fmt.Printf("Field %s is required but has no value.\n", fieldType.Name)
				return true
			}
		}
	} else {
		// 检查非指针类型的字段是否有值
		if field.Kind() == reflect.String && field.String() == "" {
			fmt.Printf("Field %s is required but has no value.\n", fieldType.Name)
			return true
		} else if field.Kind() == reflect.Int && field.Int() == 0 {
			fmt.Printf("Field %s is required but has no value.\n", fieldType.Name)
			return true
		} else if field.IsZero() {
			fmt.Printf("Field %s is required but has no value.\n", fieldType.Name)
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
	default:
		return fmt.Errorf("unsupported field type: %v with default value", field.Type())
	}
	return nil
}
