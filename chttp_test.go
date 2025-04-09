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

func Test3(t *testing.T) {

	type testStruct struct {
		Platform string `json:"platform" header:"platform" header:"provider" param:"platform" url:"platform" v:"required"`
	}

	// 测试用例1: 成功读取 JSON 请求体
	body := `{"platform": "a"}`
	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	// body
	result1, parserResult, err := Valid[testStruct](req)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if parserResult != ParserResultSuccess {
		t.Errorf("Expected ParserResultNotVerified, got %v", parserResult)
	}
	fmt.Printf(result1.Platform)

	// param
	req2, _ := http.NewRequest("GET", "/test?platform=a", nil)
	req2.Header.Set("Content-Type", "application/json")
	result2, parserResult, err := Valid[testStruct](req2)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if parserResult != ParserResultSuccess {
		t.Errorf("Expected ParserResultNotVerified, got %v", parserResult)
	}
	fmt.Printf(result2.Platform)

	// header
	req3, _ := http.NewRequest("GET", "/test?platform=b", nil)
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("platform", "c")
	result3, parserResult, err := Valid[testStruct](req3)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if parserResult != ParserResultSuccess {
		t.Errorf("Expected ParserResultNotVerified, got %v", parserResult)
	}
	fmt.Printf(result3.Platform)

	// header
	req4, _ := http.NewRequest("GET", "/test", nil)
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set("provider", "c")
	result4, parserResult, err := Valid[testStruct](req4)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if parserResult != ParserResultSuccess {
		t.Errorf("Expected ParserResultNotVerified, got %v", parserResult)
	}
	fmt.Printf(result4.Platform)
}
