package chttp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type TestStructWithBoolDefault struct {
	EnableFeature  bool `param:"enable_feature" default:"true" json:"enable_feature"`
	DisableFeature bool `param:"disable_feature" default:"false" json:"disable_feature"`
}

type TestStructWithBoolPointerDefault struct {
	EnableFeature  *bool `param:"enable_feature" default:"true" json:"enable_feature"`
	DisableFeature *bool `param:"disable_feature" default:"false" json:"disable_feature"`
}

func TestBoolDefaultValueOverride(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		expectedEnable bool
		expectedDisable bool
		description    string
	}{
		{
			name: "user_passes_false_default_true",
			queryParams: map[string]string{
				"enable_feature": "false",
			},
			expectedEnable: false, // 用户传入false，应该是false而不是默认的true
			expectedDisable: false, // 没有传入值，使用默认值false
			description: "当用户传入false，默认值是true时，应该使用用户传入的false",
		},
		{
			name: "user_passes_true_default_false",
			queryParams: map[string]string{
				"disable_feature": "true",
			},
			expectedEnable: true,  // 没有传入值，使用默认值true
			expectedDisable: true, // 用户传入true，应该是true
			description: "当用户传入true，默认值是false时，应该使用用户传入的true",
		},
		{
			name: "no_values_passed_use_defaults",
			queryParams: map[string]string{},
			expectedEnable: true,  // 使用默认值true
			expectedDisable: false, // 使用默认值false
			description: "当用户没有传入任何值时，应该使用默认值",
		},
		{
			name: "both_values_passed",
			queryParams: map[string]string{
				"enable_feature": "false",
				"disable_feature": "true",
			},
			expectedEnable: false, // 用户传入false
			expectedDisable: true, // 用户传入true
			description: "当用户传入两个值时，都应该使用用户传入的值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的HTTP请求
			req := &http.Request{
				Method: http.MethodGet,
				URL:    &url.URL{},
			}
			
			// 设置查询参数
			values := url.Values{}
			for key, value := range tt.queryParams {
				values.Set(key, value)
			}
			req.URL.RawQuery = values.Encode()

			// 解析请求
			result, validation, err := ParseWithValidation[TestStructWithBoolDefault](req)
			
			// 检查解析是否成功
			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}
			
			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证结果
			if result.EnableFeature != tt.expectedEnable {
				t.Errorf("%s: EnableFeature expected %v, got %v", 
					tt.description, tt.expectedEnable, result.EnableFeature)
			}
			
			if result.DisableFeature != tt.expectedDisable {
				t.Errorf("%s: DisableFeature expected %v, got %v", 
					tt.description, tt.expectedDisable, result.DisableFeature)
			}
		})
	}
}

func TestBoolDefaultValueWithValidFunction(t *testing.T) {
	// 测试使用Valid函数的场景
	req := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{},
	}
	
	// 用户传入false，但默认值是true
	values := url.Values{}
	values.Set("enable_feature", "false")
	req.URL.RawQuery = values.Encode()

	result, parserResult, err := Valid[TestStructWithBoolDefault](req)
	
	if err != nil {
		t.Fatalf("Valid function failed: %v", err)
	}
	
	if parserResult != ParserResultSuccess {
		t.Fatalf("Parser result expected success, got %v", parserResult)
	}
	
	// 用户传入false，应该是false而不是默认的true
	if result.EnableFeature != false {
		t.Errorf("EnableFeature should be false (user input), but got %v", result.EnableFeature)
	}
	
	// 没有传入disable_feature，应该使用默认值false
	if result.DisableFeature != false {
		t.Errorf("DisableFeature should be false (default), but got %v", result.DisableFeature)
	}
}

func TestBoolDefaultValueWithJSONBody(t *testing.T) {
	tests := []struct {
		name           string
		jsonBody       string
		expectedEnable bool
		expectedDisable bool
		description    string
	}{
		{
			name: "json_body_false_default_true",
			jsonBody: `{"enable_feature": false}`,
			expectedEnable: false, // JSON中传入false，应该是false而不是默认的true
			expectedDisable: false, // 没有传入值，使用默认值false
			description: "当JSON body中传入false，默认值是true时，应该使用用户传入的false",
		},
		{
			name: "json_body_true_default_false",
			jsonBody: `{"disable_feature": true}`,
			expectedEnable: true,  // 没有传入值，使用默认值true
			expectedDisable: true, // JSON中传入true，应该是true
			description: "当JSON body中传入true，默认值是false时，应该使用用户传入的true",
		},
		{
			name: "json_body_both_values",
			jsonBody: `{"enable_feature": false, "disable_feature": true}`,
			expectedEnable: false, // JSON中传入false
			expectedDisable: true, // JSON中传入true
			description: "当JSON body中传入两个布尔值时，都应该使用传入的值",
		},
		{
			name: "empty_json_body_use_defaults",
			jsonBody: `{}`,
			expectedEnable: true,  // 使用默认值true
			expectedDisable: false, // 使用默认值false
			description: "当JSON body为空时，应该使用默认值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的HTTP POST请求，带JSON body
			req := &http.Request{
				Method: http.MethodPost,
				URL:    &url.URL{},
				Body:   io.NopCloser(bytes.NewBuffer([]byte(tt.jsonBody))),
				Header: make(http.Header),
			}
			req.Header.Set("Content-Type", "application/json")

			// 解析请求
			result, validation, err := ParseWithValidation[TestStructWithBoolDefault](req)
			
			// 检查解析是否成功
			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}
			
			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证结果
			if result.EnableFeature != tt.expectedEnable {
				t.Errorf("%s: EnableFeature expected %v, got %v", 
					tt.description, tt.expectedEnable, result.EnableFeature)
			}
			
			if result.DisableFeature != tt.expectedDisable {
				t.Errorf("%s: DisableFeature expected %v, got %v", 
					tt.description, tt.expectedDisable, result.DisableFeature)
			}
		})
	}
}

func TestBoolPointerDefaultValueWithJSONBody(t *testing.T) {
	tests := []struct {
		name            string
		jsonBody        string
		expectedEnable  *bool
		expectedDisable *bool
		description     string
	}{
		{
			name:            "json_body_false_default_true_pointer",
			jsonBody:        `{"enable_feature": false}`,
			expectedEnable:  &[]bool{false}[0], // JSON中传入false，应该是false而不是默认的true
			expectedDisable: &[]bool{false}[0], // 没有传入值，使用默认值false
			description:     "当JSON body中传入false，默认值是true时，*bool应该使用用户传入的false",
		},
		{
			name:            "json_body_true_default_false_pointer",
			jsonBody:        `{"disable_feature": true}`,
			expectedEnable:  &[]bool{true}[0],  // 没有传入值，使用默认值true
			expectedDisable: &[]bool{true}[0],  // JSON中传入true，应该是true
			description:     "当JSON body中传入true，默认值是false时，*bool应该使用用户传入的true",
		},
		{
			name:            "empty_json_body_use_defaults_pointer",
			jsonBody:        `{}`,
			expectedEnable:  &[]bool{true}[0],  // 使用默认值true
			expectedDisable: &[]bool{false}[0], // 使用默认值false
			description:     "当JSON body为空时，*bool应该使用默认值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的HTTP POST请求，带JSON body
			req := &http.Request{
				Method: http.MethodPost,
				URL:    &url.URL{},
				Body:   io.NopCloser(bytes.NewBuffer([]byte(tt.jsonBody))),
				Header: make(http.Header),
			}
			req.Header.Set("Content-Type", "application/json")

			// 解析请求
			result, validation, err := ParseWithValidation[TestStructWithBoolPointerDefault](req)
			
			// 检查解析是否成功
			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}
			
			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证结果
			if tt.expectedEnable == nil {
				if result.EnableFeature != nil {
					t.Errorf("%s: EnableFeature expected nil, got %v", 
						tt.description, *result.EnableFeature)
				}
			} else {
				if result.EnableFeature == nil || *result.EnableFeature != *tt.expectedEnable {
					t.Errorf("%s: EnableFeature expected %v, got %v", 
						tt.description, *tt.expectedEnable, 
						func() interface{} {
							if result.EnableFeature == nil { return nil }
							return *result.EnableFeature
						}())
				}
			}
			
			if tt.expectedDisable == nil {
				if result.DisableFeature != nil {
					t.Errorf("%s: DisableFeature expected nil, got %v", 
						tt.description, *result.DisableFeature)
				}
			} else {
				if result.DisableFeature == nil || *result.DisableFeature != *tt.expectedDisable {
					t.Errorf("%s: DisableFeature expected %v, got %v", 
						tt.description, *tt.expectedDisable,
						func() interface{} {
							if result.DisableFeature == nil { return nil }
							return *result.DisableFeature
						}())
				}
			}
		})
	}
}

// 添加更全面的测试结构体，支持header、URL路径参数
type TestStructWithBoolDefaultAllSources struct {
	EnableFeature     bool `param:"enable_feature" default:"true" json:"enable_feature" header:"X-Enable-Feature"`
	DisableFeature    bool `param:"disable_feature" default:"false" json:"disable_feature" header:"X-Disable-Feature"`
	UrlParamFeature   bool `url:"url_param_feature" default:"true"`
	HeaderOnlyFeature bool `header:"X-Header-Only" default:"false"`
}

// 测试Header中的bool默认值
func TestBoolDefaultValueWithHeader(t *testing.T) {
	tests := []struct {
		name            string
		headers         map[string]string
		expectedEnable  bool
		expectedDisable bool
		expectedUrlParam bool
		expectedHeaderOnly bool
		description     string
	}{
		{
			name: "header_false_default_true",
			headers: map[string]string{
				"X-Enable-Feature": "false",
			},
			expectedEnable:     false, // Header中传入false，应该是false而不是默认的true
			expectedDisable:    false, // 没有传入值，使用默认值false
			expectedUrlParam:   true,  // 使用默认值true
			expectedHeaderOnly: false, // 使用默认值false
			description:        "当Header中传入false，默认值是true时，应该使用用户传入的false",
		},
		{
			name: "header_true_default_false",
			headers: map[string]string{
				"X-Disable-Feature": "true",
				"X-Header-Only":     "true",
			},
			expectedEnable:     true,  // 使用默认值true
			expectedDisable:    true,  // Header中传入true，应该是true
			expectedUrlParam:   true,  // 使用默认值true
			expectedHeaderOnly: true,  // Header中传入true，应该是true
			description:        "当Header中传入true，默认值是false时，应该使用用户传入的true",
		},
		{
			name:               "no_headers_use_defaults",
			headers:            map[string]string{},
			expectedEnable:     true,  // 使用默认值true
			expectedDisable:    false, // 使用默认值false
			expectedUrlParam:   true,  // 使用默认值true
			expectedHeaderOnly: false, // 使用默认值false
			description:        "当没有传入任何Header值时，应该使用默认值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的HTTP POST请求
			req := &http.Request{
				Method: http.MethodPost,
				URL:    &url.URL{},
				Header: make(http.Header),
			}
			
			// 设置headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// 解析请求
			result, validation, err := ParseWithValidation[TestStructWithBoolDefaultAllSources](req)
			
			// 检查解析是否成功
			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}
			
			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证结果
			if result.EnableFeature != tt.expectedEnable {
				t.Errorf("%s: EnableFeature expected %v, got %v", 
					tt.description, tt.expectedEnable, result.EnableFeature)
			}
			
			if result.DisableFeature != tt.expectedDisable {
				t.Errorf("%s: DisableFeature expected %v, got %v", 
					tt.description, tt.expectedDisable, result.DisableFeature)
			}
			
			if result.HeaderOnlyFeature != tt.expectedHeaderOnly {
				t.Errorf("%s: HeaderOnlyFeature expected %v, got %v", 
					tt.description, tt.expectedHeaderOnly, result.HeaderOnlyFeature)
			}
		})
	}
}

// 测试混合数据源的bool默认值优先级
func TestBoolDefaultValueMixedSources(t *testing.T) {
	tests := []struct {
		name            string
		queryParams     map[string]string
		headers         map[string]string
		jsonBody        string
		expectedEnable  bool
		expectedDisable bool
		expectedUrlParam bool
		description     string
	}{
		{
			name: "json_overrides_header_param_no_url",
			queryParams: map[string]string{
				"enable_feature": "true", // param设置为true
			},
			headers: map[string]string{
				"X-Enable-Feature": "false", // header设置为false
			},
			jsonBody:         `{"enable_feature": false}`, // JSON设置为false，应该优先
			expectedEnable:   false, // JSON优先，应该是false
			expectedDisable:  false, // 使用默认值false
			expectedUrlParam: true,  // 使用默认值true
			description:      "JSON应该覆盖header和query param的值",
		},
		{
			name: "header_overrides_param_no_json_no_url",
			queryParams: map[string]string{
				"enable_feature": "true", // param设置为true
			},
			headers: map[string]string{
				"X-Enable-Feature": "false", // header设置为false，应该优先
			},
			jsonBody:         `{}`, // 空JSON
			expectedEnable:   false, // header优先，应该是false
			expectedDisable:  false, // 使用默认值false
			expectedUrlParam: true,  // 使用默认值true
			description:      "当没有JSON时，header应该覆盖query param的值",
		},
		{
			name: "param_only_with_defaults", 
			queryParams: map[string]string{
				"disable_feature": "true", // 只设置disable_feature
			},
			headers:          map[string]string{},
			jsonBody:         `{}`,
			expectedEnable:   true, // 使用默认值true
			expectedDisable:  true, // param传入true
			expectedUrlParam: true, // 使用默认值true
			description:      "只有query param值时，其他字段使用默认值",
		},
		{
			name:             "all_defaults_when_empty",
			queryParams:      map[string]string{},
			headers:          map[string]string{},
			jsonBody:         `{}`,
			expectedEnable:   true,  // 使用默认值true
			expectedDisable:  false, // 使用默认值false
			expectedUrlParam: true,  // 使用默认值true
			description:      "当所有数据源都为空时，应该使用默认值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的HTTP POST请求
			req := &http.Request{
				Method: http.MethodPost,
				URL:    &url.URL{},
				Body:   io.NopCloser(bytes.NewBuffer([]byte(tt.jsonBody))),
				Header: make(http.Header),
			}
			req.Header.Set("Content-Type", "application/json")
			
			// 设置查询参数
			values := url.Values{}
			for key, value := range tt.queryParams {
				values.Set(key, value)
			}
			req.URL.RawQuery = values.Encode()
			
			// 设置headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// 解析请求
			result, validation, err := ParseWithValidation[TestStructWithBoolDefaultAllSources](req)
			
			// 检查解析是否成功
			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}
			
			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证结果
			if result.EnableFeature != tt.expectedEnable {
				t.Errorf("%s: EnableFeature expected %v, got %v", 
					tt.description, tt.expectedEnable, result.EnableFeature)
			}
			
			if result.DisableFeature != tt.expectedDisable {
				t.Errorf("%s: DisableFeature expected %v, got %v", 
					tt.description, tt.expectedDisable, result.DisableFeature)
			}
			
			if result.UrlParamFeature != tt.expectedUrlParam {
				t.Errorf("%s: UrlParamFeature expected %v, got %v", 
					tt.description, tt.expectedUrlParam, result.UrlParamFeature)
			}
		})
	}
}

// 测试URL路径参数的bool默认值
func TestBoolDefaultValueWithURLParam(t *testing.T) {
	// 注意：这个测试需要使用chi的URLParam，所以需要模拟chi的context
	// 这里我们用一个简化的测试，主要测试解析逻辑
	tests := []struct {
		name           string
		urlParams      map[string]string // 模拟URLParam
		expectedResult bool
		description    string
	}{
		{
			name: "url_param_false_default_true",
			urlParams: map[string]string{
				"url_param_feature": "false",
			},
			expectedResult: false, // URL参数传入false，应该是false而不是默认的true
			description:    "URL路径参数传入false时，应该使用用户传入的false而不是默认值true",
		},
		{
			name:           "no_url_param_use_default",
			urlParams:      map[string]string{},
			expectedResult: true, // 使用默认值true
			description:    "没有URL路径参数时，应该使用默认值true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 由于URL路径参数需要chi的context，这里我们主要测试逻辑
			// 在实际应用中，chi会设置URLParam到context中
			
			// 创建GET请求来测试URL参数解析
			req := &http.Request{
				Method: http.MethodGet,
				URL:    &url.URL{},
			}
			
			// 模拟chi.URLParam的行为，通过query参数来测试
			// 在实际使用中，这会是路径参数如 /api/users/{url_param_feature}
			values := url.Values{}
			for key, value := range tt.urlParams {
				values.Set(key, value)
			}
			req.URL.RawQuery = values.Encode()

			// 解析请求 - 使用简单的结构体来测试URL参数
			type URLParamTestStruct struct {
				UrlParamFeature bool `param:"url_param_feature" default:"true"`
			}
			
			result, validation, err := ParseWithValidation[URLParamTestStruct](req)
			
			// 检查解析是否成功
			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}
			
			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证结果
			if result.UrlParamFeature != tt.expectedResult {
				t.Errorf("%s: UrlParamFeature expected %v, got %v", 
					tt.description, tt.expectedResult, result.UrlParamFeature)
			}
		})
	}
}

// 测试不同类型的bool默认值
func TestBoolDefaultValueTypes(t *testing.T) {
	type TestStruct struct {
		BoolField        bool   `param:"bool_field" default:"true"`
		BoolPtrField     *bool  `param:"bool_ptr_field" default:"false"`
		StringField      string `param:"string_field" default:"default_string"`
		IntField         int    `param:"int_field" default:"42"`
		FloatField       float64 `param:"float_field" default:"3.14"`
	}
	
	tests := []struct {
		name        string
		queryParams map[string]string
		expected    TestStruct
		description string
	}{
		{
			name: "mixed_types_with_bool_false",
			queryParams: map[string]string{
				"bool_field": "false", // 传入false，应该不被默认值覆盖
				"int_field":  "100",   // 传入100，应该不被默认值覆盖
			},
			expected: TestStruct{
				BoolField:    false,            // 用户传入false
				BoolPtrField: &[]bool{false}[0], // 使用默认值false
				StringField:  "default_string",  // 使用默认值
				IntField:     100,               // 用户传入100
				FloatField:   3.14,              // 使用默认值
			},
			description: "混合类型中bool值为false时不应被默认值覆盖",
		},
		{
			name: "all_defaults",
			queryParams: map[string]string{},
			expected: TestStruct{
				BoolField:    true,              // 使用默认值true
				BoolPtrField: &[]bool{false}[0], // 使用默认值false
				StringField:  "default_string",  // 使用默认值
				IntField:     42,                // 使用默认值
				FloatField:   3.14,              // 使用默认值
			},
			description: "所有字段都应该使用默认值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Method: http.MethodGet,
				URL:    &url.URL{},
			}
			
			// 设置查询参数
			values := url.Values{}
			for key, value := range tt.queryParams {
				values.Set(key, value)
			}
			req.URL.RawQuery = values.Encode()

			result, validation, err := ParseWithValidation[TestStruct](req)
			
			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}
			
			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证结果
			if result.BoolField != tt.expected.BoolField {
				t.Errorf("%s: BoolField expected %v, got %v", 
					tt.description, tt.expected.BoolField, result.BoolField)
			}
			
			if (result.BoolPtrField == nil) != (tt.expected.BoolPtrField == nil) {
				t.Errorf("%s: BoolPtrField nil mismatch", tt.description)
			} else if result.BoolPtrField != nil && tt.expected.BoolPtrField != nil {
				if *result.BoolPtrField != *tt.expected.BoolPtrField {
					t.Errorf("%s: BoolPtrField expected %v, got %v", 
						tt.description, *tt.expected.BoolPtrField, *result.BoolPtrField)
				}
			}
			
			if result.StringField != tt.expected.StringField {
				t.Errorf("%s: StringField expected %v, got %v", 
					tt.description, tt.expected.StringField, result.StringField)
			}
			
			if result.IntField != tt.expected.IntField {
				t.Errorf("%s: IntField expected %v, got %v", 
					tt.description, tt.expected.IntField, result.IntField)
			}
			
			if result.FloatField != tt.expected.FloatField {
				t.Errorf("%s: FloatField expected %v, got %v", 
					tt.description, tt.expected.FloatField, result.FloatField)
			}
		})
	}
}

// 测试String类型的默认值
func TestStringDefaultValue(t *testing.T) {
	type TestStruct struct {
		StringField      string  `param:"string_field" default:"default_value" json:"string_field" header:"X-String"`
		EmptyStringField string  `param:"empty_string_field" default:"should_use_default" json:"empty_string_field"`
		StringPtrField   *string `param:"string_ptr_field" default:"ptr_default" json:"string_ptr_field"`
	}

	tests := []struct {
		name        string
		method      string
		queryParams map[string]string
		headers     map[string]string
		jsonBody    string
		expected    TestStruct
		description string
	}{
		{
			name:   "empty_string_should_use_user_input",
			method: "GET",
			queryParams: map[string]string{
				"string_field": "", // 传入空字符串，应该使用用户传入的空字符串
			},
			headers:  map[string]string{},
			jsonBody: "",
			expected: TestStruct{
				StringField:      "",                   // 用户传入空字符串
				EmptyStringField: "should_use_default", // 没有传入，使用默认值
				StringPtrField:   &[]string{"ptr_default"}[0], // 使用默认值
			},
			description: "空字符串应该被当作有效的用户输入",
		},
		{
			name:   "non_empty_string_overrides_default",
			method: "GET",
			queryParams: map[string]string{
				"string_field": "user_value",
			},
			headers:  map[string]string{},
			jsonBody: "",
			expected: TestStruct{
				StringField:      "user_value",         // 用户传入值
				EmptyStringField: "should_use_default", // 使用默认值
				StringPtrField:   &[]string{"ptr_default"}[0], // 使用默认值
			},
			description: "非空字符串应该覆盖默认值",
		},
		{
			name:   "header_overrides_param",
			method: "POST",
			queryParams: map[string]string{
				"string_field": "param_value",
			},
			headers: map[string]string{
				"X-String": "header_value", // Header应该覆盖param
			},
			jsonBody: `{}`,
			expected: TestStruct{
				StringField:      "header_value",       // Header优先
				EmptyStringField: "should_use_default", // 使用默认值
				StringPtrField:   &[]string{"ptr_default"}[0], // 使用默认值
			},
			description: "Header应该覆盖Query参数",
		},
		{
			name:   "json_overrides_all",
			method: "POST",
			queryParams: map[string]string{
				"string_field": "param_value",
			},
			headers: map[string]string{
				"X-String": "header_value",
			},
			jsonBody: `{"string_field": "json_value", "string_ptr_field": "json_ptr_value"}`,
			expected: TestStruct{
				StringField:      "json_value",         // JSON最高优先级
				EmptyStringField: "should_use_default", // 使用默认值
				StringPtrField:   &[]string{"json_ptr_value"}[0], // JSON设置的值
			},
			description: "JSON应该覆盖所有其他数据源",
		},
		{
			name:   "json_empty_string_should_use_user_input",
			method: "POST",
			queryParams: map[string]string{
				"string_field": "param_value",
			},
			headers: map[string]string{
				"X-String": "header_value",
			},
			jsonBody: `{"string_field": "", "string_ptr_field": ""}`, // JSON中的空字符串应该使用用户传入的空字符串
			expected: TestStruct{
				StringField:      "",                   // JSON中的空字符串，使用用户输入
				EmptyStringField: "should_use_default", // 使用默认值
				StringPtrField:   &[]string{""}[0],     // JSON中的空字符串，使用用户输入
			},
			description: "JSON中的空字符串应该被当作有效的用户输入",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Method: tt.method,
				URL:    &url.URL{},
				Header: make(http.Header),
			}

			// 设置查询参数
			if len(tt.queryParams) > 0 {
				values := url.Values{}
				for key, value := range tt.queryParams {
					values.Set(key, value)
				}
				req.URL.RawQuery = values.Encode()
			}

			// 设置headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// 设置JSON body
			if tt.jsonBody != "" {
				req.Body = io.NopCloser(bytes.NewBuffer([]byte(tt.jsonBody)))
				req.Header.Set("Content-Type", "application/json")
			}

			result, validation, err := ParseWithValidation[TestStruct](req)

			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}

			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证字符串字段
			if result.StringField != tt.expected.StringField {
				t.Errorf("%s: StringField expected %q, got %q",
					tt.description, tt.expected.StringField, result.StringField)
			}

			if result.EmptyStringField != tt.expected.EmptyStringField {
				t.Errorf("%s: EmptyStringField expected %q, got %q",
					tt.description, tt.expected.EmptyStringField, result.EmptyStringField)
			}

			// 验证指针字段
			if tt.expected.StringPtrField == nil {
				if result.StringPtrField != nil {
					t.Errorf("%s: StringPtrField expected nil, got %q",
						tt.description, *result.StringPtrField)
				}
			} else {
				if result.StringPtrField == nil {
					t.Errorf("%s: StringPtrField expected %q, got nil",
						tt.description, *tt.expected.StringPtrField)
				} else if *result.StringPtrField != *tt.expected.StringPtrField {
					t.Errorf("%s: StringPtrField expected %q, got %q",
						tt.description, *tt.expected.StringPtrField, *result.StringPtrField)
				}
			}
		})
	}
}

// 测试Int类型的默认值
func TestIntDefaultValue(t *testing.T) {
	type TestStruct struct {
		IntField     int   `param:"int_field" default:"42" json:"int_field" header:"X-Int"`
		ZeroIntField int   `param:"zero_int_field" default:"100" json:"zero_int_field"`
		IntPtrField  *int  `param:"int_ptr_field" default:"99" json:"int_ptr_field"`
		Int64Field   int64 `param:"int64_field" default:"12345" json:"int64_field"`
	}

	tests := []struct {
		name        string
		method      string
		queryParams map[string]string
		headers     map[string]string
		jsonBody    string
		expected    TestStruct
		description string
	}{
		{
			name:   "zero_int_should_use_default",
			method: "GET",
			queryParams: map[string]string{
				"int_field": "0", // 传入0，应该使用用户传入的0而不是默认值
			},
			headers:  map[string]string{},
			jsonBody: "",
			expected: TestStruct{
				IntField:     0,   // 用户传入0
				ZeroIntField: 100, // 使用默认值
				IntPtrField:  &[]int{99}[0], // 使用默认值
				Int64Field:   12345, // 使用默认值
			},
			description: "传入0应该使用用户的0值，而不是默认值",
		},
		{
			name:   "positive_int_overrides_default",
			method: "GET",
			queryParams: map[string]string{
				"int_field":      "123",
				"zero_int_field": "0", // 传入0应该覆盖默认值100
			},
			headers:  map[string]string{},
			jsonBody: "",
			expected: TestStruct{
				IntField:     123, // 用户传入值
				ZeroIntField: 0,   // 用户传入0
				IntPtrField:  &[]int{99}[0], // 使用默认值
				Int64Field:   12345, // 使用默认值
			},
			description: "正整数和0都应该覆盖默认值",
		},
		{
			name:   "negative_int_overrides_default",
			method: "GET",
			queryParams: map[string]string{
				"int_field": "-50", // 负数也应该覆盖默认值
			},
			headers:  map[string]string{},
			jsonBody: "",
			expected: TestStruct{
				IntField:     -50, // 用户传入负数
				ZeroIntField: 100, // 使用默认值
				IntPtrField:  &[]int{99}[0], // 使用默认值
				Int64Field:   12345, // 使用默认值
			},
			description: "负整数应该覆盖默认值",
		},
		{
			name:   "json_int_values",
			method: "POST",
			queryParams: map[string]string{
				"int_field": "999", // 会被JSON覆盖
			},
			headers: map[string]string{
				"X-Int": "888", // 会被JSON覆盖
			},
			jsonBody: `{"int_field": 0, "int_ptr_field": -1, "int64_field": 0}`,
			expected: TestStruct{
				IntField:     0,    // JSON传入0
				ZeroIntField: 100,  // 使用默认值
				IntPtrField:  &[]int{-1}[0], // JSON传入-1
				Int64Field:   0,    // JSON传入0
			},
			description: "JSON中的整数值（包括0和负数）应该覆盖默认值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Method: tt.method,
				URL:    &url.URL{},
				Header: make(http.Header),
			}

			// 设置查询参数
			if len(tt.queryParams) > 0 {
				values := url.Values{}
				for key, value := range tt.queryParams {
					values.Set(key, value)
				}
				req.URL.RawQuery = values.Encode()
			}

			// 设置headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// 设置JSON body
			if tt.jsonBody != "" {
				req.Body = io.NopCloser(bytes.NewBuffer([]byte(tt.jsonBody)))
				req.Header.Set("Content-Type", "application/json")
			}

			result, validation, err := ParseWithValidation[TestStruct](req)

			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}

			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证结果
			if result.IntField != tt.expected.IntField {
				t.Errorf("%s: IntField expected %d, got %d",
					tt.description, tt.expected.IntField, result.IntField)
			}

			if result.ZeroIntField != tt.expected.ZeroIntField {
				t.Errorf("%s: ZeroIntField expected %d, got %d",
					tt.description, tt.expected.ZeroIntField, result.ZeroIntField)
			}

			if tt.expected.IntPtrField == nil {
				if result.IntPtrField != nil {
					t.Errorf("%s: IntPtrField expected nil, got %d",
						tt.description, *result.IntPtrField)
				}
			} else {
				if result.IntPtrField == nil {
					t.Errorf("%s: IntPtrField expected %d, got nil",
						tt.description, *tt.expected.IntPtrField)
				} else if *result.IntPtrField != *tt.expected.IntPtrField {
					t.Errorf("%s: IntPtrField expected %d, got %d",
						tt.description, *tt.expected.IntPtrField, *result.IntPtrField)
				}
			}

			if result.Int64Field != tt.expected.Int64Field {
				t.Errorf("%s: Int64Field expected %d, got %d",
					tt.description, tt.expected.Int64Field, result.Int64Field)
			}
		})
	}
}

// 测试Float类型的默认值
func TestFloatDefaultValue(t *testing.T) {
	type TestStruct struct {
		FloatField    float64  `param:"float_field" default:"3.14" json:"float_field" header:"X-Float"`
		ZeroFloatField float64 `param:"zero_float_field" default:"2.71" json:"zero_float_field"`
		FloatPtrField *float64 `param:"float_ptr_field" default:"1.41" json:"float_ptr_field"`
		Float32Field  float32  `param:"float32_field" default:"9.99" json:"float32_field"`
	}

	tests := []struct {
		name        string
		method      string
		queryParams map[string]string
		headers     map[string]string
		jsonBody    string
		expected    TestStruct
		description string
	}{
		{
			name:   "zero_float_should_use_user_value",
			method: "GET",
			queryParams: map[string]string{
				"float_field": "0.0", // 传入0.0，应该使用用户值
			},
			headers:  map[string]string{},
			jsonBody: "",
			expected: TestStruct{
				FloatField:     0.0, // 用户传入0.0
				ZeroFloatField: 2.71, // 使用默认值
				FloatPtrField:  &[]float64{1.41}[0], // 使用默认值
				Float32Field:   9.99, // 使用默认值
			},
			description: "传入0.0应该使用用户的值而不是默认值",
		},
		{
			name:   "negative_float_overrides_default",
			method: "GET",
			queryParams: map[string]string{
				"float_field":      "-1.5",
				"zero_float_field": "0", // 传入0应该覆盖默认值
			},
			headers:  map[string]string{},
			jsonBody: "",
			expected: TestStruct{
				FloatField:     -1.5, // 用户传入负数
				ZeroFloatField: 0.0,  // 用户传入0
				FloatPtrField:  &[]float64{1.41}[0], // 使用默认值
				Float32Field:   9.99, // 使用默认值
			},
			description: "负浮点数和0都应该覆盖默认值",
		},
		{
			name:   "json_float_values",
			method: "POST",
			queryParams: map[string]string{
				"float_field": "999.9", // 会被JSON覆盖
			},
			headers: map[string]string{
				"X-Float": "888.8", // 会被JSON覆盖
			},
			jsonBody: `{"float_field": 0.0, "float_ptr_field": -3.14, "float32_field": 0}`,
			expected: TestStruct{
				FloatField:     0.0,  // JSON传入0.0
				ZeroFloatField: 2.71, // 使用默认值
				FloatPtrField:  &[]float64{-3.14}[0], // JSON传入-3.14
				Float32Field:   0.0,  // JSON传入0
			},
			description: "JSON中的浮点数值（包括0.0和负数）应该覆盖默认值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Method: tt.method,
				URL:    &url.URL{},
				Header: make(http.Header),
			}

			// 设置查询参数
			if len(tt.queryParams) > 0 {
				values := url.Values{}
				for key, value := range tt.queryParams {
					values.Set(key, value)
				}
				req.URL.RawQuery = values.Encode()
			}

			// 设置headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// 设置JSON body
			if tt.jsonBody != "" {
				req.Body = io.NopCloser(bytes.NewBuffer([]byte(tt.jsonBody)))
				req.Header.Set("Content-Type", "application/json")
			}

			result, validation, err := ParseWithValidation[TestStruct](req)

			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}

			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证结果
			if result.FloatField != tt.expected.FloatField {
				t.Errorf("%s: FloatField expected %f, got %f",
					tt.description, tt.expected.FloatField, result.FloatField)
			}

			if result.ZeroFloatField != tt.expected.ZeroFloatField {
				t.Errorf("%s: ZeroFloatField expected %f, got %f",
					tt.description, tt.expected.ZeroFloatField, result.ZeroFloatField)
			}

			if tt.expected.FloatPtrField == nil {
				if result.FloatPtrField != nil {
					t.Errorf("%s: FloatPtrField expected nil, got %f",
						tt.description, *result.FloatPtrField)
				}
			} else {
				if result.FloatPtrField == nil {
					t.Errorf("%s: FloatPtrField expected %f, got nil",
						tt.description, *tt.expected.FloatPtrField)
				} else if *result.FloatPtrField != *tt.expected.FloatPtrField {
					t.Errorf("%s: FloatPtrField expected %f, got %f",
						tt.description, *tt.expected.FloatPtrField, *result.FloatPtrField)
				}
			}

			if result.Float32Field != tt.expected.Float32Field {
				t.Errorf("%s: Float32Field expected %f, got %f",
					tt.description, tt.expected.Float32Field, result.Float32Field)
			}
		})
	}
}

// 测试所有类型的混合默认值场景
func TestMixedTypesDefaultValue(t *testing.T) {
	type TestStruct struct {
		BoolField    bool     `param:"bool_field" default:"true" json:"bool_field" header:"X-Bool"`
		StringField  string   `param:"string_field" default:"default_str" json:"string_field" header:"X-String"`
		IntField     int      `param:"int_field" default:"42" json:"int_field" header:"X-Int"`
		FloatField   float64  `param:"float_field" default:"3.14" json:"float_field" header:"X-Float"`
		BoolPtrField *bool    `param:"bool_ptr_field" default:"false" json:"bool_ptr_field"`
		IntPtrField  *int     `param:"int_ptr_field" default:"100" json:"int_ptr_field"`
	}

	tests := []struct {
		name        string
		method      string
		queryParams map[string]string
		headers     map[string]string
		jsonBody    string
		expected    TestStruct
		description string
	}{
		{
			name:   "all_zero_values_from_user",
			method: "GET",
			queryParams: map[string]string{
				"bool_field":   "false", // 用户传入false
				"string_field": "",      // 用户传入空字符串
				"int_field":    "0",     // 用户传入0
				"float_field":  "0.0",   // 用户传入0.0
			},
			headers:  map[string]string{},
			jsonBody: "",
			expected: TestStruct{
				BoolField:    false,                      // 用户传入false，不应该被默认值true覆盖
				StringField:  "",                         // 用户传入空字符串，应该使用用户输入
				IntField:     0,                          // 用户传入0，不应该被默认值42覆盖
				FloatField:   0.0,                        // 用户传入0.0，不应该被默认值3.14覆盖
				BoolPtrField: &[]bool{false}[0],          // 使用默认值
				IntPtrField:  &[]int{100}[0],             // 使用默认值
			},
			description: "用户传入的零值（false, 0, 0.0, 空字符串）不应该被默认值覆盖",
		},
		{
			name:   "mixed_sources_priority",
			method: "POST",
			queryParams: map[string]string{
				"bool_field":   "true",   // param
				"string_field": "param_string", 
				"int_field":    "999",
				"float_field":  "999.9",
			},
			headers: map[string]string{
				"X-Bool":   "false",        // header应该覆盖param
				"X-String": "header_string",
				"X-Int":    "888",
				"X-Float":  "888.8",
			},
			jsonBody: `{"bool_field": true, "int_field": 0}`, // JSON应该覆盖header和param
			expected: TestStruct{
				BoolField:    true,                       // JSON值
				StringField:  "header_string",            // Header值（JSON中没有）
				IntField:     0,                          // JSON值（0不应该被默认值覆盖）
				FloatField:   888.8,                      // Header值（JSON中没有）
				BoolPtrField: &[]bool{false}[0],          // 使用默认值
				IntPtrField:  &[]int{100}[0],             // 使用默认值
			},
			description: "数据源优先级：JSON > Header > Param，零值不应该被默认值覆盖",
		},
		{
			name:   "partial_json_with_defaults",
			method: "POST",
			queryParams: map[string]string{},
			headers:     map[string]string{},
			jsonBody:    `{"bool_ptr_field": true, "int_ptr_field": 0}`, // 指针字段的JSON值
			expected: TestStruct{
				BoolField:    true,                       // 使用默认值true
				StringField:  "default_str",              // 使用默认值
				IntField:     42,                         // 使用默认值
				FloatField:   3.14,                       // 使用默认值
				BoolPtrField: &[]bool{true}[0],           // JSON传入true
				IntPtrField:  &[]int{0}[0],               // JSON传入0，不应该被默认值覆盖
			},
			description: "部分字段由JSON设置，其他字段使用默认值，JSON中的0值不应该被默认值覆盖",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Method: tt.method,
				URL:    &url.URL{},
				Header: make(http.Header),
			}

			// 设置查询参数
			if len(tt.queryParams) > 0 {
				values := url.Values{}
				for key, value := range tt.queryParams {
					values.Set(key, value)
				}
				req.URL.RawQuery = values.Encode()
			}

			// 设置headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// 设置JSON body
			if tt.jsonBody != "" {
				req.Body = io.NopCloser(bytes.NewBuffer([]byte(tt.jsonBody)))
				req.Header.Set("Content-Type", "application/json")
			}

			result, validation, err := ParseWithValidation[TestStruct](req)

			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}

			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证所有字段
			if result.BoolField != tt.expected.BoolField {
				t.Errorf("%s: BoolField expected %v, got %v",
					tt.description, tt.expected.BoolField, result.BoolField)
			}

			if result.StringField != tt.expected.StringField {
				t.Errorf("%s: StringField expected %q, got %q",
					tt.description, tt.expected.StringField, result.StringField)
			}

			if result.IntField != tt.expected.IntField {
				t.Errorf("%s: IntField expected %d, got %d",
					tt.description, tt.expected.IntField, result.IntField)
			}

			if result.FloatField != tt.expected.FloatField {
				t.Errorf("%s: FloatField expected %f, got %f",
					tt.description, tt.expected.FloatField, result.FloatField)
			}

			// 验证指针字段
			if tt.expected.BoolPtrField == nil {
				if result.BoolPtrField != nil {
					t.Errorf("%s: BoolPtrField expected nil, got %v",
						tt.description, *result.BoolPtrField)
				}
			} else {
				if result.BoolPtrField == nil {
					t.Errorf("%s: BoolPtrField expected %v, got nil",
						tt.description, *tt.expected.BoolPtrField)
				} else if *result.BoolPtrField != *tt.expected.BoolPtrField {
					t.Errorf("%s: BoolPtrField expected %v, got %v",
						tt.description, *tt.expected.BoolPtrField, *result.BoolPtrField)
				}
			}

			if tt.expected.IntPtrField == nil {
				if result.IntPtrField != nil {
					t.Errorf("%s: IntPtrField expected nil, got %d",
						tt.description, *result.IntPtrField)
				}
			} else {
				if result.IntPtrField == nil {
					t.Errorf("%s: IntPtrField expected %d, got nil",
						tt.description, *tt.expected.IntPtrField)
				} else if *result.IntPtrField != *tt.expected.IntPtrField {
					t.Errorf("%s: IntPtrField expected %d, got %d",
						tt.description, *tt.expected.IntPtrField, *result.IntPtrField)
				}
			}
		})
	}
}

// 注意：在实际使用chi router的场景中，URL路径参数将拥有最高优先级
// URL参数优先级：URL路径参数 > JSON > Header > Query参数 > 默认值
// 测试当前优先级顺序：JSON > Header > Query > Default
func TestCurrentPriorityOrder(t *testing.T) {
	type TestStruct struct {
		Feature bool `param:"feature" default:"true" json:"feature" header:"X-Feature"`
	}
	
	tests := []struct {
		name        string
		jsonBody    string
		queryParam  string
		headerValue string
		expected    bool
		description string
	}{
		{
			name:        "json_highest_priority",
			jsonBody:    `{"feature": false}`,
			queryParam:  "true",
			headerValue: "true", 
			expected:    false,
			description: "JSON应该拥有最高优先级，覆盖header和query参数",
		},
		{
			name:        "header_overrides_query_and_default",
			jsonBody:    `{}`,
			queryParam:  "false",
			headerValue: "true",
			expected:    true,
			description: "当没有JSON时，Header应该覆盖query参数和默认值",
		},
		{
			name:        "query_overrides_default",
			jsonBody:    `{}`,
			queryParam:  "false",
			headerValue: "",
			expected:    false,
			description: "当没有JSON和Header时，Query参数应该覆盖默认值",
		},
		{
			name:        "use_default_when_no_input",
			jsonBody:    `{}`,
			queryParam:  "",
			headerValue: "",
			expected:    true,
			description: "当没有任何输入时，应该使用默认值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的HTTP POST请求
			req := &http.Request{
				Method: http.MethodPost,
				URL:    &url.URL{},
				Body:   io.NopCloser(bytes.NewBuffer([]byte(tt.jsonBody))),
				Header: make(http.Header),
			}
			req.Header.Set("Content-Type", "application/json")
			
			// 设置查询参数
			if tt.queryParam != "" {
				values := url.Values{}
				values.Set("feature", tt.queryParam)
				req.URL.RawQuery = values.Encode()
			}
			
			// 设置header
			if tt.headerValue != "" {
				req.Header.Set("X-Feature", tt.headerValue)
			}

			// 解析请求
			result, validation, err := ParseWithValidation[TestStruct](req)
			if err != nil {
				t.Fatalf("ParseWithValidation failed: %v", err)
			}
			if validation.Valid == nil || !*validation.Valid {
				t.Fatalf("Validation failed: %v", validation.ValidMessage)
			}

			// 验证结果
			if result.Feature != tt.expected {
				t.Errorf("%s: Feature expected %v, got %v", 
					tt.description, tt.expected, result.Feature)
			}
		})
	}
}
