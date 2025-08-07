package proxy

import (
	"time"

	"github.com/eolinker/ai-prompt-proxy/internal/logger"
	"github.com/gin-gonic/gin"
)

func APIAuthMiddleware(c *gin.Context) {

}

func AccessLogMiddleware(c *gin.Context) {
	startTime := time.Now()
	c.Next()
	headers := make(map[string]string, len(c.Request.Header))
	for k, v := range c.Request.Header {
		headers[k] = v[0] // 只记录第一个值
	}
	// 记录访问日志
	logData := logger.RequestLogData{
		RequestID:    c.GetString("request_id"),
		Timestamp:    startTime,
		Method:       c.Request.Method,
		Path:         c.Request.URL.Path,
		UserAgent:    c.Request.UserAgent(),
		ClientIP:     c.GetString("client_ip"),
		APIKey:       c.GetString("api_key"),
		UserID:       c.GetUint("user_id"),
		RequestSize:  c.Request.ContentLength,
		RequestBody:  c.GetString("request_body"), // 原始请求body
		Headers:      headers,
		ModelID:      c.GetString("model_id"),
		TargetModel:  c.GetString("target_model"),
		ProxyURL:     c.GetString("proxy_url"),
		ProxyScheme:  c.GetString("proxy_scheme"),
		ProxyHost:    c.GetString("proxy_host"),
		UpstreamBody: c.GetString("proxy_body"), // 发送给上游的body
		StatusCode:   c.Writer.Status(),
		ResponseSize: int64(c.Writer.Size()),
		ResponseTime: time.Since(startTime).Milliseconds(),
		ResponseBody: c.GetString("response_body"), // 响应body
		Error:        c.GetString("error"),
	}
	go func() {
		logger.GlobalLoggerManager.LogToAll(logData)
	}()
}
