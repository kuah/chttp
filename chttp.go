package chttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
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
	switch r.Method {
	case http.MethodGet:
		err := parseRequestParams(r, &result)
		if err != nil {
			return result, &ParamValidation{Valid: &vCompleted, ValidMessage: &validationMsg}, errors.New("Invalid request params")
		}
	default:
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
					if err := json.NewDecoder(bytes.NewBuffer(body)).Decode(&result); err != nil {
						return result, nil, errors.Wrap(err, "body is not json")
					}
				}
			}
		}
		err := parseRequestParams(r, &result)
		if err != nil {
			return result, nil, errors.Wrap(err, "Invalid request params")
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

// bool validation
// string validation failed message

// parseRequestParamsWithValidation
// error error
func parseRequestParams(r *http.Request, arg interface{}) error {
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
		pTag := v.Type().Field(i).Tag.Get("cv")
		defaultTag := v.Type().Field(i).Tag.Get("default")
		rawJsonTag := v.Type().Field(i).Tag.Get("rawJson")

		var value string
		if paramTag != "" {
			value = values.Get(paramTag)
		}
		if value == "" && urlTag != "" {
			urlTag = strings.Split(urlTag, ",")[0]
			value = chi.URLParam(r, urlTag)
		}
		if value == "" && headerTag != "" {
			headerTag = strings.Split(headerTag, ",")[0]
			value = headers.Get(headerTag)
		}
		// struct 类型, 判断是否往下层递归
		if pTag != "" && field.Kind() == reflect.Struct && field.CanSet() && field.CanInterface() {
			subErr := parseRequestParams(r, field.Addr().Interface())
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
				subErr := parseRequestParams(r, field.Interface())
				if subErr != nil {
					return subErr
				}
				continue
			}
		}
		if value != "" && field.CanSet() {
			if err := setFieldValue(field, value); err != nil {
				return err
			}
		} else if defaultTag != "" && field.CanSet() && checkIfNull(field, fieldType) {
			if err := setFieldValue(field, defaultTag); err != nil {
				return err
			}
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
							return err
						}
						field.Set(reflect.ValueOf(variable).Elem())
					} else if sourceField.Kind() == reflect.Ptr && sourceField.Elem().Kind() == reflect.String {
						// 获取对应的field的值
						variable := reflect.New(field.Type()).Interface()
						value := sourceField.Elem().String()
						if err := json.Unmarshal([]byte(value), variable); err != nil {
							return err
						}
						field.Set(reflect.ValueOf(variable).Elem())
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
		// 检查指针是否为 nil
		if field.IsNil() {
			return true
		} else {
			// 获取指针指向的值
			elem := field.Elem()
			if elem.Kind() == reflect.String && elem.String() == "" {
				return true
			} else if elem.Kind() == reflect.Int && elem.Int() == 0 {
				return true
			} else if elem.IsZero() {
				return true
			}
		}
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
	default:
		return fmt.Errorf("unsupported field type: %v with default value", field.Type())
	}
	return nil
}
