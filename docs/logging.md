# 日志功能说明

## 概述

AI Prompt Proxy 现在支持完整的请求日志记录功能，可以记录所有通过代理服务器的请求详细信息。

## 功能特性

### 1. 日志记录器管理
- 支持多个日志记录器实例
- 全局日志管理器统一管理
- 支持动态添加、删除、启用/禁用日志记录器

### 2. 格式化器
- **JSON格式化器**: 输出结构化的JSON日志
- **行格式化器**: 输出自定义格式的文本日志
- 支持字段映射和别名
- 支持系统变量和引用

### 3. 文件输出
- 自动日志文件轮转（按小时或按天）
- 自动清理过期日志文件
- 支持自定义日志目录和文件名
- 线程安全的文件写入

### 4. 记录的信息

每个请求日志包含以下信息：

```json
{
  "request_id": "唯一请求ID",
  "timestamp": "请求时间戳",
  "method": "HTTP方法",
  "path": "请求路径",
  "user_agent": "用户代理",
  "client_ip": "客户端IP",
  "api_key": "API密钥",
  "user_id": "用户ID",
  "request_size": "请求大小(字节)",
  "model_id": "原始模型ID",
  "target_model": "目标模型ID",
  "proxy_url": "代理URL",
  "proxy_scheme": "代理协议",
  "proxy_host": "代理主机",
  "status_code": "HTTP状态码",
  "response_size": "响应大小(字节)",
  "response_time_ms": "响应时间(毫秒)",
  "error": "错误信息(如有)"
}
```

## 配置

### 默认配置

程序启动时会自动创建一个默认的日志记录器：

```go
defaultConfig := logger.OutputConfig{
    Name:        "default",
    Driver:      "file",
    Description: "默认访问日志",
    Enabled:     true,
    Type:        logger.FormatterJSON,
    File:        "access.log",
    Dir:         "./logs",
    Period:      logger.PeriodDay,
    Expire:      30, // 保留30天
    Formatter: logger.FormatterConfig{
        Fields: map[string][]string{
            "default": {
                "request_id", "timestamp", "method", "path", "user_agent", 
                "client_ip", "api_key", "user_id", "request_size", "model_id", 
                "target_model", "proxy_url", "proxy_scheme", "proxy_host", 
                "status_code", "response_size", "response_time", "error",
            },
        },
    },
}
```

### 自定义配置

可以通过代码添加自定义的日志记录器：

```go
// 添加自定义日志记录器
config := logger.OutputConfig{
    Name:        "custom",
    Driver:      "file", 
    Description: "自定义日志",
    Enabled:     true,
    Type:        logger.FormatterLine,
    File:        "custom.log",
    Dir:         "./logs",
    Period:      logger.PeriodHour,
    Expire:      7, // 保留7天
    Formatter: logger.FormatterConfig{
        Fields: map[string][]string{
            "default": {"timestamp", "method", "path", "status_code"},
        },
    },
}

err := logger.GlobalLoggerManager.AddLogger("custom", config)
```

## 日志文件

### 文件位置
- 默认日志目录: `./logs`
- 默认日志文件: `access.log.log`

### 文件轮转
- 按天轮转: `access.log.2025-08-07.log`
- 按小时轮转: `access.log.2025-08-07-14.log`

### 自动清理
- 根据配置的 `Expire` 天数自动删除过期日志文件
- 每小时检查一次过期文件

## API接口

### 日志管理器方法

```go
// 添加日志记录器
func (m *LoggerManager) AddLogger(name string, config OutputConfig) error

// 获取日志记录器
func (m *LoggerManager) GetLogger(name string) (*RequestLogger, bool)

// 移除日志记录器
func (m *LoggerManager) RemoveLogger(name string) error

// 列出所有日志记录器
func (m *LoggerManager) ListLoggers() []string

// 向所有启用的日志记录器记录日志
func (m *LoggerManager) LogToAll(data RequestLogData)

// 关闭所有日志记录器
func (m *LoggerManager) Close() error
```

### 日志记录器方法

```go
// 记录请求日志
func (l *RequestLogger) LogRequest(data RequestLogData) error

// 启用/禁用日志记录
func (l *RequestLogger) Enable()
func (l *RequestLogger) Disable()
func (l *RequestLogger) IsEnabled() bool

// 获取日志文件列表
func (l *RequestLogger) GetLogFiles() ([]LogFileInfo, error)

// 读取日志文件内容
func (l *RequestLogger) ReadLogFile(filename string, offset int64, limit int64) ([]byte, error)

// 关闭日志记录器
func (l *RequestLogger) Close() error
```

## 性能考虑

1. **异步记录**: 所有日志记录都是异步进行的，不会阻塞请求处理
2. **缓冲写入**: 使用缓冲写入提高文件I/O性能
3. **内存管理**: 及时关闭文件句柄，避免内存泄漏
4. **错误处理**: 日志记录失败不会影响正常的请求处理

## 使用示例

### 在代理处理器中记录日志

```go
// 记录请求日志
logData := logger.RequestLogData{
    RequestID:    requestID,
    Timestamp:    time.Now(),
    Method:       c.Request.Method,
    Path:         c.Request.URL.Path,
    ClientIP:     c.ClientIP(),
    APIKey:       apiKey,
    ModelID:      modelID,
    StatusCode:   200,
    ResponseTime: time.Since(startTime).Milliseconds(),
}

// 异步记录到所有启用的日志记录器
logger.GlobalLoggerManager.LogToAll(logData)
```

## 故障排除

### 常见问题

1. **日志文件未创建**
   - 检查日志目录权限
   - 确认日志记录器已启用

2. **日志记录失败**
   - 检查磁盘空间
   - 查看控制台错误信息

3. **性能问题**
   - 减少记录的字段数量
   - 增加日志轮转频率
   - 减少日志保留天数

### 调试信息

程序启动时会输出日志初始化信息：
```
2025/08/07 14:04:28 默认日志记录器初始化成功
```

日志记录失败时会在控制台输出错误信息：
```
记录日志失败: 写入文件失败: permission denied
```