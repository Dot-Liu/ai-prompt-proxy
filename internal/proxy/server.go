package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eolinker/ai-prompt-proxy/internal/config"
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
	r.Use(s.apiKeyAuthMiddleware()) // 添加API Key验证中间件

	// 代理所有请求
	r.Any("/*path", s.proxyHandler)

	return r.Run("0.0.0.0:" + port)
}

// apiKeyAuthMiddleware API Key验证中间件
func (s *Server) apiKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取X-Proxy-Key头部
		apiKey := c.GetHeader("X-Proxy-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "缺少API Key，请在请求头中添加X-Proxy-Key",
			})
			c.Abort()
			return
		}

		// 验证API Key
		if s.authService == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "认证服务不可用",
			})
			c.Abort()
			return
		}

		// 从数据库获取API Key信息
		apiKeyInfo, err := s.authService.GetAPIKeyByValue(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的API Key",
			})
			c.Abort()
			return
		}

		// 检查API Key是否启用
		if !apiKeyInfo.IsEnabled {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API Key已被禁用",
			})
			c.Abort()
			return
		}

		// 检查API Key是否过期
		if apiKeyInfo.ExpiresAt != nil && time.Now().After(*apiKeyInfo.ExpiresAt) {
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

// proxyHandler 代理请求处理器
func (s *Server) proxyHandler(c *gin.Context) {
	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	// 解析请求体以获取模型ID
	modelID := extractModelID(body)

	// 查找模型配置
	modelConfig, exists := s.config.GetModel(modelID)

	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("转发请求失败: %v", err)})
		return
	}

	// 如果找到模型配置，注入Prompt并替换模型ID
	modifiedBody, err := injectPrompt(body, modelConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("注入Prompt失败: %v", err)})
		return
	}

	// 修改模型ID为目标模型ID
	modifiedBody, err = replaceModelID(modifiedBody, modelConfig.Target)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("替换模型ID失败: %v", err)})
		return
	}

	// 使用模型配置中的URL
	upstreamURL := modelConfig.Url

	// 转发请求到上游服务
	if err := s.forwardRequest(c, upstreamURL, modifiedBody); err != nil {
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

	// 处理非流式响应
	_, err = io.Copy(c.Writer, resp.Body)
	return err
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

	return nil
}
