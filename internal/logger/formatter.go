package logger

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// JSONFormatter JSON格式化器
type JSONFormatter struct {
	config FormatterConfig
}

// NewJSONFormatter 创建JSON格式化器
func NewJSONFormatter(config FormatterConfig) *JSONFormatter {
	return &JSONFormatter{
		config: config,
	}
}

// Format 格式化为JSON格式
func (f *JSONFormatter) Format(data *RequestLogData) ([]byte, error) {
	result := make(map[string]interface{})
	
	// 处理fields配置
	if fields, exists := f.config.Fields["fields"]; exists {
		for _, field := range fields {
			f.processField(field, data, result)
		}
	}
	
	// 处理自定义引用项
	for key, fields := range f.config.Fields {
		if key != "fields" {
			customData := make(map[string]interface{})
			for _, field := range fields {
				f.processField(field, data, customData)
			}
			result[key] = customData
		}
	}
	
	return json.Marshal(result)
}

// LineFormatter Line格式化器
type LineFormatter struct {
	config FormatterConfig
}

// NewLineFormatter 创建Line格式化器
func NewLineFormatter(config FormatterConfig) *LineFormatter {
	return &LineFormatter{
		config: config,
	}
}

// Format 格式化为Line格式
func (f *LineFormatter) Format(data *RequestLogData) ([]byte, error) {
	var parts []string
	
	// 处理fields配置
	if fields, exists := f.config.Fields["fields"]; exists {
		for _, field := range fields {
			value := f.extractFieldValue(field, data)
			parts = append(parts, fmt.Sprintf("%v", value))
		}
	}
	
	result := strings.Join(parts, "\t") + "\n"
	return []byte(result), nil
}

// processField 处理字段配置
func (f *JSONFormatter) processField(field string, data *RequestLogData, result map[string]interface{}) {
	// 解析字段格式: ($|@){pattern}[#] as {name}
	field = strings.TrimSpace(field)
	
	var fieldName, alias string
	var isArray bool
	
	// 检查是否有别名
	if strings.Contains(field, " as ") {
		parts := strings.Split(field, " as ")
		if len(parts) == 2 {
			field = strings.TrimSpace(parts[0])
			alias = strings.TrimSpace(parts[1])
		}
	}
	
	// 检查是否为数组引用
	if strings.HasSuffix(field, "#") {
		isArray = true
		field = strings.TrimSuffix(field, "#")
	}
	
	// 提取字段值
	value := f.extractFieldValue(field, data)
	
	// 确定最终的字段名
	if alias != "" {
		fieldName = alias
	} else if strings.HasPrefix(field, "$") {
		fieldName = strings.TrimPrefix(field, "$")
	} else if strings.HasPrefix(field, "@") {
		fieldName = strings.TrimPrefix(field, "@")
	} else {
		fieldName = field
	}
	
	// 处理数组类型
	if isArray {
		if arr, ok := value.([]interface{}); ok {
			result[fieldName] = arr
		} else {
			result[fieldName] = []interface{}{value}
		}
	} else {
		result[fieldName] = value
	}
}

// extractFieldValue 提取字段值
func (f *JSONFormatter) extractFieldValue(field string, data *RequestLogData) interface{} {
	// 处理系统变量 ($pattern)
	if strings.HasPrefix(field, "$") {
		return f.getSystemValue(strings.TrimPrefix(field, "$"), data)
	}
	
	// 处理引用 (@pattern)
	if strings.HasPrefix(field, "@") {
		refKey := strings.TrimPrefix(field, "@")
		if refFields, exists := f.config.Fields[refKey]; exists {
			refResult := make(map[string]interface{})
			for _, refField := range refFields {
				f.processField(refField, data, refResult)
			}
			return refResult
		}
		return nil
	}
	
	// 处理常量
	return field
}

// extractFieldValue Line格式化器的字段值提取
func (f *LineFormatter) extractFieldValue(field string, data *RequestLogData) interface{} {
	// 移除别名部分，只保留字段名
	if strings.Contains(field, " as ") {
		parts := strings.Split(field, " as ")
		field = strings.TrimSpace(parts[0])
	}
	
	// 移除数组标记
	field = strings.TrimSuffix(field, "#")
	
	// 处理系统变量
	if strings.HasPrefix(field, "$") {
		return f.getSystemValue(strings.TrimPrefix(field, "$"), data)
	}
	
	// 处理引用
	if strings.HasPrefix(field, "@") {
		refKey := strings.TrimPrefix(field, "@")
		if refFields, exists := f.config.Fields[refKey]; exists {
			var values []string
			for _, refField := range refFields {
				value := f.extractFieldValue(refField, data)
				values = append(values, fmt.Sprintf("%v", value))
			}
			return strings.Join(values, " ")
		}
		return ""
	}
	
	// 处理常量
	return field
}

// getSystemValue 获取系统变量值
func (f *JSONFormatter) getSystemValue(pattern string, data *RequestLogData) interface{} {
	return getSystemValue(pattern, data)
}

// getSystemValue Line格式化器的系统变量获取
func (f *LineFormatter) getSystemValue(pattern string, data *RequestLogData) interface{} {
	return getSystemValue(pattern, data)
}

// getSystemValue 通用的系统变量获取函数
func getSystemValue(pattern string, data *RequestLogData) interface{} {
	switch pattern {
	// 基础信息
	case "request_id":
		return data.RequestID
	case "timestamp", "time_iso8601":
		return data.Timestamp.Format(time.RFC3339)
	case "time_local":
		return data.Timestamp.Format("2006-01-02 15:04:05")
	case "msec":
		return data.Timestamp.UnixMilli()
	case "method", "request_method":
		return data.Method
	case "path", "request_uri":
		return data.Path
	case "user_agent":
		return data.UserAgent
	case "client_ip", "remote_addr":
		return data.ClientIP
		
	// 认证信息
	case "api_key":
		return data.APIKey
	case "user_id":
		return data.UserID
		
	// 请求信息
	case "request_size", "request_length":
		return data.RequestSize
	case "request_body":
		return data.RequestBody
		
	// 代理信息
	case "model_id":
		return data.ModelID
	case "target_model":
		return data.TargetModel
	case "proxy_uri", "proxy_url":
		return data.ProxyURL
	case "proxy_scheme":
		return data.ProxyScheme
	case "proxy_host", "proxy_addr":
		return data.ProxyHost
	case "upstream_body":
		return data.UpstreamBody
		
	// 响应信息
	case "status", "status_code":
		return data.StatusCode
	case "response_size", "response_length":
		return data.ResponseSize
	case "response_time":
		return data.ResponseTime
	case "response_body":
		return data.ResponseBody
		
	// 错误信息
	case "error":
		return data.Error
		
	default:
		// 尝试从Extra中获取
		if data.Extra != nil {
			if value, exists := data.Extra[pattern]; exists {
				return value
			}
		}
		
		// 使用反射尝试获取字段值
		v := reflect.ValueOf(data).Elem()
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" {
				tagName := strings.Split(jsonTag, ",")[0]
				if tagName == pattern {
					return v.Field(i).Interface()
				}
			}
		}
		
		return ""
	}
}