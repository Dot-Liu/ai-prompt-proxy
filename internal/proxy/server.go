package proxy

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eolinker/ai-prompt-proxy/internal/config"
	"github.com/eolinker/ai-prompt-proxy/internal/logger"
	"github.com/eolinker/ai-prompt-proxy/internal/service"
)

// Server 代理服务器
type Server struct {
	config      *config.Config
	httpClient  *http.Client
	authService *service.AuthService
}

// NewServer 创建新的代理服务器
func NewServer(cfg *config.Config, authService *service.AuthService) *Server {
	return &Server{
		config:      cfg,
		httpClient:  &http.Client{},
		authService: authService,
	}
}

// Start 启动服务器
func (s *Server) Start(port string) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(AccessLogMiddleware)
	r.Use(s.apiKeyAuthMiddleware()) // 添加API Key验证中间件

	// 代理所有请求
	r.Any("/*path", s.proxyHandler)

	return r.Run(":" + port)
}

// apiKeyAuthMiddleware API Key验证中间件
func (s *Server) apiKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := generateRequestID()
		c.Set("request_id", requestID)

		// 获取客户端IP
		clientIP := c.ClientIP()
		if clientIP == "" {
			clientIP = c.Request.RemoteAddr
		}

		forwardedFor := c.Request.Header.Get("X-Forwarded-For")
		if forwardedFor != "" {
			// 如果有X-Forwarded-For头部，取第一个IP作为客户端IP
			clientIP = strings.Split(forwardedFor, ",")[0]
		} else {
			if clientIP == "::1" {
				clientIP = "127.0.0.1"
			} else {
				// 如果是IPv6地址，转换为IPv4地址
				if strings.HasPrefix(clientIP, "[") && strings.HasSuffix(clientIP, "]") {
					clientIP = strings.Trim(clientIP, "[]")
				}

				if ip := net.ParseIP(clientIP); ip != nil {
					clientIP = ip.String()
				}
				// 如果无法解析IP，使用RemoteAddr
				if clientIP == "" {
					clientIP = c.Request.RemoteAddr
				}
			}
		}
		realIP := c.GetHeader("x-real-ip") // 尝试获取真实IP头部
		if realIP != "" {
			clientIP = realIP
		}

		c.Set("client_ip", clientIP)

		body, _ := io.ReadAll(c.Request.Body)
		c.Set("request_body", string(body)) // 保存原始请求体到上下文)

		// 尝试获取X-Proxy-Key头部
		apiKey := c.GetHeader("X-Proxy-Key")

		c.Set("api_key", apiKey)
		// 如果两种认证方式都没有提供有效凭据
		if apiKey == "" {
			c.Set("error", "缺少认证信息")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "缺少认证信息，请在请求头中添加X-Proxy-Key或Authorization Bearer token",
			})
			c.Abort()
			return
		}

		// 验证API Key
		if s.authService == nil {
			c.Set("error", "认证服务不可用")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "认证服务不可用",
			})
			c.Abort()
			return
		}

		// 从数据库获取API Key信息
		apiKeyInfo, err := s.authService.GetAPIKeyByValue(apiKey)
		if err != nil {
			c.Set("error", "无效的API Key")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的API Key",
			})
			c.Abort()
			return
		}
		c.Set("user_id", apiKeyInfo.UserID)

		// 检查API Key是否启用
		if !apiKeyInfo.IsEnabled {

			c.Set("error", "API Key已被禁用")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API Key已被禁用",
			})
			c.Abort()
			return
		}

		// 检查API Key是否过期
		if apiKeyInfo.ExpiresAt != nil && time.Now().After(*apiKeyInfo.ExpiresAt) {
			// 记录API Key过期日志
			c.Set("error", "API Key已过期")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API Key已过期",
			})
			c.Abort()
			return
		}

		// 更新API Key最后使用时间（异步执行，不影响请求性能）
		go func() {
			if err := s.authService.UpdateAPIKeyLastUsed(apiKey); err != nil {
				// 记录错误但不影响请求
				fmt.Printf("更新API Key最后使用时间失败: %v\n", err)
			}
		}()

		// 将API Key信息存储到上下文中，供后续使用
		c.Set("api_key_info", apiKeyInfo)
		c.Set("user_id", apiKeyInfo.UserID)
		// 继续处理请求
		c.Next()
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// logRequest 记录请求日志
func (s *Server) logRequest(data logger.RequestLogData) {
	// 异步记录日志，避免影响请求性能
	go func() {
		logger.GlobalLoggerManager.LogToAll(data)
	}()
}

// proxyHandler 代理请求处理器
func (s *Server) proxyHandler(c *gin.Context) {
	bodyStr := c.GetString("request_body")
	body := []byte(bodyStr)
	// 解析请求体以获取模型ID
	modelID := extractModelID(body)
	c.Set("model_id", modelID)
	// 查找模型配置
	modelConfig, exists := s.config.GetModel(modelID)
	if !exists {
		c.Set("error", fmt.Sprintf("模型配置未找到: %s", modelID))
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("模型配置未找到: %s", modelID)})
		return
	}
	c.Set("target_model", modelConfig.Target)
	// 如果找到模型配置，注入Prompt并替换模型ID
	modifiedBody, err := injectPrompt(body, modelConfig)
	if err != nil {
		// 记录注入失败的错误日志
		c.Set("error", fmt.Sprintf("注入Prompt失败: %v", err))

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("注入Prompt失败: %v", err)})
		return
	}

	// 修改模型ID为目标模型ID
	modifiedBody, err = replaceModelID(modifiedBody, modelConfig.Target)
	if err != nil {
		c.Set("error", fmt.Sprintf("替换模型ID失败: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("替换模型ID失败: %v", err)})
		return
	}
	c.Set("modified_body", string(modifiedBody))

	// 解析上游URL
	upstreamURL := modelConfig.Url
	parseURL, err := url.Parse(upstreamURL)
	if err != nil {
		// 记录URL解析失败的错误日志
		c.Set("error", fmt.Sprintf("解析上游URL失败: %v, URL: %s", err, upstreamURL))
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("解析上游URL失败: %v", err)})
		return
	}
	c.Set("proxy_url", upstreamURL)
	c.Set("proxy_scheme", parseURL.Scheme)
	c.Set("proxy_host", parseURL.Host)
	c.Set("proxy_port", parseURL.Port())
	c.Set("proxy_path", parseURL.Path)
	c.Set("proxy_body", string(modifiedBody))

	// 转发请求到上游服务
	if err := s.forwardRequest(c, upstreamURL, modifiedBody); err != nil {
		// forwardRequestWithLogging 内部已经处理了日志记录
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("转发请求失败: %v", err)})
		return
	}
}

// forwardRequest 转发请求到上游服务
func (s *Server) forwardRequest(c *gin.Context, upstreamURL string, body []byte) error {
	// 创建新的请求
	req, err := http.NewRequest(c.Request.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	// 复制原始请求的头部
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// 更新Content-Length
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// 设置状态码
	c.Status(resp.StatusCode)

	// 检查是否为流式响应
	if s.isStreamingResponse(resp) {
		return s.handleStreamingResponse(c, resp)
	}
	bodyBuilder := strings.Builder{}
	reader := bufio.NewReader(resp.Body)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		bodyBuilder.Write(line)
		if _, err = c.Writer.Write(line); err != nil {
			break
		}
	}
	c.Set("response_body", bodyBuilder.String())
	if err != nil {
		c.Set("error", err.Error())
		return err
	}
	return nil
}

// isStreamingResponse 检查是否为流式响应
func (s *Server) isStreamingResponse(resp *http.Response) bool {
	contentType := resp.Header.Get("Content-Type")
	return strings.Contains(contentType, "text/event-stream") ||
		strings.Contains(contentType, "application/x-ndjson") ||
		strings.Contains(contentType, "text/plain")
}

// handleStreamingResponse 处理流式响应
func (s *Server) handleStreamingResponse(c *gin.Context, resp *http.Response) error {
	// 设置流式响应的必要头部
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// 确保响应立即发送
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}

	// 创建缓冲读取器
	reader := bufio.NewReader(resp.Body)
	bodyBuilder := &strings.Builder{}
	for {
		// 逐行读取响应
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("读取流式响应失败: %w", err)
		}

		// 写入响应数据
		if _, err := c.Writer.Write(line); err != nil {
			return fmt.Errorf("写入流式响应失败: %w", err)
		}
		bodyBuilder.Write(line)

		// 立即刷新缓冲区
		if flusher, ok := c.Writer.(http.Flusher); ok {
			flusher.Flush()
		}

		// 检查客户端是否断开连接
		select {
		case <-c.Request.Context().Done():
			return c.Request.Context().Err()
		default:
			// 继续处理
		}
	}

	c.Set("response_body", bodyBuilder.String())
	return nil
}

// handleStreamingResponseWithLogging 处理流式响应并返回响应大小
func (s *Server) handleStreamingResponseWithLogging(c *gin.Context, resp *http.Response) (int64, error) {
	// 设置流式响应的必要头部
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// 确保响应立即发送
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}

	// 创建缓冲读取器
	reader := bufio.NewReader(resp.Body)
	var totalSize int64

	for {
		// 逐行读取响应
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return totalSize, fmt.Errorf("读取流式响应失败: %w", err)
		}

		// 写入响应数据
		n, err := c.Writer.Write(line)
		if err != nil {
			return totalSize, fmt.Errorf("写入流式响应失败: %w", err)
		}
		totalSize += int64(n)

		// 立即刷新缓冲区
		if flusher, ok := c.Writer.(http.Flusher); ok {
			flusher.Flush()
		}

		// 检查客户端是否断开连接
		select {
		case <-c.Request.Context().Done():
			return totalSize, c.Request.Context().Err()
		default:
			// 继续处理
		}
	}

	return totalSize, nil
}

// copyResponseWithSize 复制响应并返回响应大小
func (s *Server) copyResponseWithSize(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}
