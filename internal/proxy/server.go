package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/eolinker/ai-prompt-proxy/internal/config"
)

// Server 代理服务器
type Server struct {
	config     *config.Config
	httpClient *http.Client
}

// NewServer 创建新的代理服务器
func NewServer(cfg *config.Config) *Server {
	return &Server{
		config:     cfg,
		httpClient: &http.Client{},
	}
}

// Start 启动服务器
func (s *Server) Start(port string) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// 代理所有请求
	r.Any("/*path", s.proxyHandler)

	return r.Run("0.0.0.0:" + port)
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

	var modifiedBody []byte
	var upstreamURL string

	if exists {
		// 如果找到模型配置，注入Prompt并替换模型ID
		modifiedBody, err = injectPrompt(body, modelConfig)
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
		upstreamURL = getUpstreamURL(modelConfig.Url, c.Request.URL.Path)
	} else {
		// 如果没有找到模型配置，直接使用原始请求体
		modifiedBody = body

		// 没有模型配置时，返回错误或使用默认URL
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("未找到模型配置: %s", modelID)})
		return
	}
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
