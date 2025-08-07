package logger

import (
	"fmt"
	"sync"
	"time"
)

// RequestLogger 请求日志记录器
type RequestLogger struct {
	formatter Formatter
	output    Output
	config    OutputConfig
	mutex     sync.RWMutex
	enabled   bool
}

// NewRequestLogger 创建请求日志记录器
func NewRequestLogger(config OutputConfig) (*RequestLogger, error) {
	// 创建格式化器
	var formatter Formatter

	switch config.Type {
	case FormatterJSON:
		formatter = NewJSONFormatter(config.Formatter)
	case FormatterLine:
		formatter = NewLineFormatter(config.Formatter)
	default:
		return nil, fmt.Errorf("不支持的格式化器类型: %s", config.Type)
	}

	// 创建输出器
	output, err := NewFileOutput(config)
	if err != nil {
		return nil, fmt.Errorf("创建文件输出器失败: %w", err)
	}

	logger := &RequestLogger{
		formatter: formatter,
		output:    output,
		config:    config,
		enabled:   true,
	}

	return logger, nil
}

// LogRequest 记录请求日志
func (l *RequestLogger) LogRequest(data RequestLogData) error {
	l.mutex.RLock()
	enabled := l.enabled
	l.mutex.RUnlock()

	if !enabled {
		return nil
	}

	// 设置时间戳
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	// 格式化数据
	formatted, err := l.formatter.Format(&data)
	if err != nil {
		return fmt.Errorf("格式化日志数据失败: %w", err)
	}

	// 添加换行符
	formatted = append(formatted, '\n')

	// 输出到文件
	if err := l.output.Write(formatted); err != nil {
		return fmt.Errorf("写入日志失败: %w", err)
	}

	return nil
}

// Enable 启用日志记录
func (l *RequestLogger) Enable() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.enabled = true
}

// Disable 禁用日志记录
func (l *RequestLogger) Disable() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.enabled = false
}

// IsEnabled 检查是否启用
func (l *RequestLogger) IsEnabled() bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.enabled
}

// Close 关闭日志记录器
func (l *RequestLogger) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.enabled = false

	if l.output != nil {
		return l.output.Close()
	}

	return nil
}

// GetConfig 获取配置
func (l *RequestLogger) GetConfig() OutputConfig {
	return l.config
}

// GetLogFiles 获取日志文件列表
func (l *RequestLogger) GetLogFiles() ([]LogFileInfo, error) {
	if fileOutput, ok := l.output.(*FileOutput); ok {
		return fileOutput.GetLogFiles()
	}
	return nil, fmt.Errorf("输出器不支持文件列表功能")
}

// ReadLogFile 读取日志文件内容
func (l *RequestLogger) ReadLogFile(filename string, offset int64, limit int64) ([]byte, error) {
	if fileOutput, ok := l.output.(*FileOutput); ok {
		return fileOutput.ReadLogFile(filename, offset, limit)
	}
	return nil, fmt.Errorf("输出器不支持文件读取功能")
}

// LoggerManager 日志管理器
type LoggerManager struct {
	loggers map[string]*RequestLogger
	mutex   sync.RWMutex
}

// NewLoggerManager 创建日志管理器
func NewLoggerManager() *LoggerManager {
	return &LoggerManager{
		loggers: make(map[string]*RequestLogger),
	}
}

// AddLogger 添加日志记录器
func (m *LoggerManager) AddLogger(name string, config OutputConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 如果已存在，先关闭旧的
	if oldLogger, exists := m.loggers[name]; exists {
		oldLogger.Close()
	}

	// 创建新的日志记录器
	logger, err := NewRequestLogger(config)
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %w", err)
	}

	m.loggers[name] = logger
	return nil
}

// GetLogger 获取日志记录器
func (m *LoggerManager) GetLogger(name string) (*RequestLogger, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	logger, exists := m.loggers[name]
	return logger, exists
}

// RemoveLogger 移除日志记录器
func (m *LoggerManager) RemoveLogger(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if logger, exists := m.loggers[name]; exists {
		if err := logger.Close(); err != nil {
			return fmt.Errorf("关闭日志记录器失败: %w", err)
		}
		delete(m.loggers, name)
	}

	return nil
}

// ListLoggers 列出所有日志记录器
func (m *LoggerManager) ListLoggers() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var names []string
	for name := range m.loggers {
		names = append(names, name)
	}

	return names
}

// LogToAll 向所有启用的日志记录器记录日志
func (m *LoggerManager) LogToAll(data RequestLogData) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, logger := range m.loggers {
		if logger.IsEnabled() {
			// 异步记录，避免阻塞
			go func(l *RequestLogger) {
				if err := l.LogRequest(data); err != nil {
					fmt.Printf("记录日志失败: %v\n", err)
				}
			}(logger)
		}
	}
}

// Close 关闭所有日志记录器
func (m *LoggerManager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var lastErr error
	for name, logger := range m.loggers {
		if err := logger.Close(); err != nil {
			lastErr = err
			fmt.Printf("关闭日志记录器 %s 失败: %v\n", name, err)
		}
	}

	// 清空映射
	m.loggers = make(map[string]*RequestLogger)

	return lastErr
}

// 全局日志管理器实例
var GlobalLoggerManager = NewLoggerManager()
