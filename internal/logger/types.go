package logger

import (
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// FormatterType 格式化器类型
type FormatterType string

const (
	FormatterJSON FormatterType = "json"
	FormatterLine FormatterType = "line"
)

// Period 日志文件轮转周期
type Period string

const (
	PeriodHour Period = "hour"
	PeriodDay  Period = "day"
)

// RequestLogData 请求日志数据结构
type RequestLogData struct {
	// 基础信息
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	UserAgent string    `json:"user_agent"`
	ClientIP  string    `json:"client_ip"`

	// 认证信息
	APIKey string `json:"api_key,omitempty"`
	UserID uint   `json:"user_id,omitempty"`

	// 请求信息
	RequestSize int64             `json:"request_size"`
	RequestBody string            `json:"request_body,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`

	// 代理信息
	ModelID       string `json:"model_id"`
	TargetModel   string `json:"target_model"`
	ProxyURL      string `json:"proxy_url"`
	ProxyScheme   string `json:"proxy_scheme"`
	ProxyHost     string `json:"proxy_host"`
	UpstreamBody  string `json:"upstream_body,omitempty"`  // 发送给上游服务的body

	// 响应信息
	StatusCode   int    `json:"status_code"`
	ResponseSize int64  `json:"response_size"`
	ResponseTime int64  `json:"response_time_ms"` // 毫秒
	ResponseBody string `json:"response_body,omitempty"`  // 响应body

	// 错误信息
	Error string `json:"error,omitempty"`

	// 扩展信息
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// OutputConfig 输出器配置
type OutputConfig struct {
	// 基础配置
	Name        string `json:"name" yaml:"name"`
	Driver      string `json:"driver" yaml:"driver"`
	Description string `json:"description" yaml:"description"`
	Enabled     bool   `json:"enabled" yaml:"enabled"`

	// 文件配置
	File   string `json:"file" yaml:"file"`
	Dir    string `json:"dir" yaml:"dir"`
	Period Period `json:"period" yaml:"period"`
	Expire int    `json:"expire" yaml:"expire"` // 保留天数

	// 格式化配置
	Type      FormatterType   `json:"type" yaml:"type"`
	Formatter FormatterConfig `json:"formatter" yaml:"formatter"`
}

// FormatterConfig 格式化器配置
type FormatterConfig struct {
	Fields map[string][]string `json:"fields" yaml:"fields"`
}

// Logger 日志记录器接口
type Logger interface {
	// LogRequest 记录请求日志
	LogRequest(data *RequestLogData) error

	// Close 关闭日志记录器
	Close() error
}

// Formatter 格式化器接口
type Formatter interface {
	// Format 格式化日志数据
	Format(data *RequestLogData) ([]byte, error)
}

// Output 输出器接口
type Output interface {
	// Write 写入日志数据
	Write(data []byte) error

	// Close 关闭输出器
	Close() error
}
