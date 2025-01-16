package chttp

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

// TestValid 测试 Valid 函数
func TestValid(t *testing.T) {
	// 测试用例1: 成功验证
	req, _ := http.NewRequest("GET", "/test?param1=value1", nil)
	type testStruct struct {
		Param1 string `param:"param1" v:"required"`
	}
	result, parserResult, err := Valid[testStruct](req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if parserResult != ParserResultSuccess {
		t.Errorf("Expected ParserResultSuccess, got %v", parserResult)
	}
	if result.Param1 != "value1" {
		t.Errorf("Expected Param1 to be 'value1', got %v", result.Param1)
	}

	// 测试用例2: 参数缺失
	req, _ = http.NewRequest("GET", "/test", nil)
	_, parserResult, err = Valid[testStruct](req)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if parserResult != ParserResultNotVerified {
		t.Errorf("Expected ParserResultNotVerified, got %v", parserResult)
	}

	type user struct {
		Phone string `param:"phone" v:"required,number"`
	}

	// 测试用例 验证param是否手机号
	req, _ = http.NewRequest("GET", "/test?phone=12345678901", nil)
	_, parserResult, err = Valid[user](req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if parserResult == ParserResultNotVerified {
		t.Errorf("Expected ParserResultNotVerified, got %v", parserResult)
	}

	// 测试用例 验证param是否手机号
	req1, _ := http.NewRequest("GET", "/test?phone=1234das5678901", nil)
	a, parserResult, err := Valid[user](req1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
		println(fmt.Sprintf("%v", a))
	}
	if parserResult != ParserResultNotVerified {
		t.Errorf("Expected ParserResultNotVerified, got %v", parserResult)
	}
}

// TestReadRequestBody 测试 ReadRequestBody 函数
func TestReadRequestBody(t *testing.T) {
	// 测试用例1: 成功读取 JSON 请求体
	body := `{"param1": "value1"}`
	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	type testStruct struct {
		Param1 string `json:"param1"`
	}
	result, err := ReadRequestBody[testStruct](req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result.Param1 != "value1" {
		t.Errorf("Expected Param1 to be 'value1', got %v", result.Param1)
	}

	// 测试用例2: 无效的 JSON 请求体
	body = `{"param1": "value1"`
	req, _ = http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	_, err = ReadRequestBody[testStruct](req)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
