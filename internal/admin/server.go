package admin

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eolinker/ai-prompt-proxy/internal/config"
	"github.com/eolinker/ai-prompt-proxy/internal/service"
	"github.com/gin-gonic/gin"
)

//go:embed web/*
var webFS embed.FS

// AdminServer 管理API服务器
type AdminServer struct {
	config        *config.Config
	configDir     string
	configService *service.ConfigService
	authService   *service.AuthService
	proxyPort     string // 代理服务端口
	adminPort     string // 管理服务端口
}

// NewAdminServer 创建新的管理API服务器
func NewAdminServer(cfg *config.Config, configDir string) *AdminServer {
	return &AdminServer{
		config:    cfg,
		configDir: configDir,
	}
}

// NewAdminServerWithService 使用配置服务创建新的管理API服务器
func NewAdminServerWithService(configService *service.ConfigService, configDir string, proxyPort, adminPort string) (*AdminServer, error) {
	// 创建认证服务
	authService, err := service.NewAuthService(configService.GetDBManager())
	if err != nil {
		return nil, fmt.Errorf("创建认证服务失败: %w", err)
	}

	return &AdminServer{
		config:        configService.GetConfig(),
		configDir:     configDir,
		configService: configService,
		authService:   authService,
		proxyPort:     proxyPort,
		adminPort:     adminPort,
	}, nil
}

// Start 启动管理API服务器
func (s *AdminServer) Start(port string) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(s.corsMiddleware())

	// 健康检查 - 放在最前面避免路由冲突
	r.GET("/health", s.healthCheck)

	// 设置嵌入式静态文件服务
	s.setupEmbeddedStaticFiles(r)

	// 根路径处理 - 放在最后避免覆盖其他路由
	r.NoRoute(func(c *gin.Context) {
		// 如果是API请求，返回404
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "API接口不存在",
			})
			return
		}
		// 其他请求返回前端页面
		s.serveIndexHTML(c)
	})

	// API路由组
	api := r.Group("/api/v1")
	{
		// 认证相关API（无需认证）
		auth := api.Group("/auth")
		{
			auth.GET("/check-install", s.checkInstall)            // 检查是否首次安装
			auth.GET("/public-key", s.getPublicKey)               // 获取公钥
			auth.POST("/register", s.register)                    // 用户注册（仅首次安装）
			auth.POST("/encrypted-register", s.encryptedRegister) // 加密用户注册
			auth.POST("/login", s.login)                          // 用户登录
			auth.POST("/encrypted-login", s.encryptedLogin)       // 加密用户登录
		}

		// 公开配置API（无需认证）
		publicConfig := api.Group("/config")
		{
			publicConfig.GET("/system", s.getSystemConfig) // 获取系统配置
		}

		// 需要认证的API
		protected := api.Group("")
		protected.Use(s.authMiddleware())
		{
			// 认证相关API
			protected.POST("/auth/logout", s.logout)     // 用户注销
			protected.GET("/auth/profile", s.getProfile) // 获取用户信息

			// 模型相关API
			models := protected.Group("/models")
			{
				models.GET("", s.getModels)          // 获取模型列表
				models.GET("/:id", s.getModel)       // 根据模型ID获取模型信息
				models.PUT("/:id", s.updateModel)    // 根据模型ID配置模型信息
				models.POST("", s.createModel)       // 创建模型配置
				models.DELETE("/:id", s.deleteModel) // 删除模型配置
			}

			// 配置相关API
			config := protected.Group("/config")
			{
				config.POST("/reload", s.reloadConfig) // 重新加载配置
				config.GET("/status", s.getStatus)     // 获取服务状态
			}

			// 用户管理API（需要管理员权限）
			users := protected.Group("/users")
			users.Use(s.adminMiddleware()) // 添加管理员权限检查
			{
				users.GET("", s.getUsers)                         // 获取用户列表
				users.POST("", s.createUser)                      // 创建用户
				users.PUT("/:id", s.updateUser)                   // 更新用户信息
				users.DELETE("/:id", s.deleteUser)                // 删除用户
				users.PUT("/:id/status", s.updateUserStatus)      // 更新用户状态
				users.PUT("/:id/password", s.adminChangePassword) // 管理员修改用户密码
			}

			// 用户个人相关API（所有用户都可以访问）
			user := protected.Group("/user")
			{
				user.PUT("/password", s.changePassword) // 修改自己的密码
			}

			// API Key管理API（所有用户都可以访问自己的API Key）
			apiKeys := protected.Group("/api-keys")
			{
				apiKeys.GET("", s.getAPIKeys)          // 获取当前用户的API Key列表
				apiKeys.POST("", s.createAPIKey)       // 创建API Key
				apiKeys.DELETE("/:id", s.deleteAPIKey) // 删除API Key
			}
		}
	}

	return r.Run(fmt.Sprintf(":%s", port))
}

// corsMiddleware CORS中间件
func (s *AdminServer) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ModelResponse 模型响应结构
type ModelResponse struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Target          string           `json:"target"`
	Prompt          string           `json:"prompt"`
	Url             string           `json:"url"`
	Type            config.ModelType `json:"type"`
	PromptPath      string           `json:"prompt_path"`
	PromptValue     interface{}      `json:"prompt_value"`
	PromptValueType config.ValueType `json:"prompt_value_type"`
	CreatedAt       string           `json:"created_at"`
	UpdatedAt       string           `json:"updated_at"`
}

// CreateModelRequest 创建模型请求结构
type CreateModelRequest struct {
	ID              string           `json:"id" binding:"required"`
	Name            string           `json:"name" binding:"required"`
	Target          string           `json:"target" binding:"required"`
	Prompt          string           `json:"prompt"`
	Url             string           `json:"url" binding:"required"`
	Type            config.ModelType `json:"type" binding:"required"`
	PromptPath      string           `json:"prompt_path"`
	PromptValue     interface{}      `json:"prompt_value"`
	PromptValueType config.ValueType `json:"prompt_value_type"`
}

// UpdateModelRequest 更新模型请求结构
type UpdateModelRequest struct {
	Name            string           `json:"name"`
	Target          string           `json:"target"`
	Prompt          string           `json:"prompt"`
	Url             string           `json:"url"`
	Type            config.ModelType `json:"type"`
	PromptPath      string           `json:"prompt_path"`
	PromptValue     interface{}      `json:"prompt_value"`
	PromptValueType config.ValueType `json:"prompt_value_type"`
}

// getModels 获取模型列表
func (s *AdminServer) getModels(c *gin.Context) {
	var models []ModelResponse

	if s.configService != nil {
		// 使用配置服务获取包含时间信息的模型数据
		dbModels, err := s.configService.GetAllModelsWithTime()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": fmt.Sprintf("获取模型列表失败: %v", err),
			})
			return
		}

		for _, dbModel := range dbModels {
			// 转换为配置模型
			model, err := dbModel.ToModelConfig()
			if err != nil {
				continue // 跳过转换失败的模型
			}

			models = append(models, ModelResponse{
				ID:              model.ID,
				Name:            model.Name,
				Target:          model.Target,
				Prompt:          model.Prompt,
				Url:             model.Url,
				Type:            model.Type,
				PromptPath:      model.PromptPath,
				PromptValue:     model.PromptValue,
				PromptValueType: model.PromptValueType,
				CreatedAt:       dbModel.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:       dbModel.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
	} else {
		// 降级方案：从内存配置获取（无时间信息）
		for _, model := range s.config.Models {
			models = append(models, ModelResponse{
				ID:              model.ID,
				Name:            model.Name,
				Target:          model.Target,
				Prompt:          model.Prompt,
				Url:             model.Url,
				Type:            model.Type,
				PromptPath:      model.PromptPath,
				PromptValue:     model.PromptValue,
				PromptValueType: model.PromptValueType,
				CreatedAt:       "",
				UpdatedAt:       "",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"models": models,
			"total":  len(models),
		},
	})
}

// getModel 根据模型ID获取模型信息
func (s *AdminServer) getModel(c *gin.Context) {
	modelID := c.Param("id")

	var response ModelResponse

	if s.configService != nil {
		// 使用配置服务获取包含时间信息的模型数据
		dbModel, err := s.configService.GetModelWithTime(modelID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": fmt.Sprintf("模型 %s 不存在", modelID),
			})
			return
		}

		// 转换为配置模型
		model, err := dbModel.ToModelConfig()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": fmt.Sprintf("模型数据转换失败: %v", err),
			})
			return
		}

		response = ModelResponse{
			ID:              model.ID,
			Name:            model.Name,
			Target:          model.Target,
			Prompt:          model.Prompt,
			Url:             model.Url,
			Type:            model.Type,
			PromptPath:      model.PromptPath,
			PromptValue:     model.PromptValue,
			PromptValueType: model.PromptValueType,
			CreatedAt:       dbModel.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:       dbModel.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	} else {
		// 降级方案：从内存配置获取（无时间信息）
		model, exists := s.config.GetModel(modelID)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": fmt.Sprintf("模型 %s 不存在", modelID),
			})
			return
		}

		response = ModelResponse{
			ID:              model.ID,
			Name:            model.Name,
			Target:          model.Target,
			Prompt:          model.Prompt,
			Url:             model.Url,
			Type:            model.Type,
			PromptPath:      model.PromptPath,
			PromptValue:     model.PromptValue,
			PromptValueType: model.PromptValueType,
			CreatedAt:       "",
			UpdatedAt:       "",
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    response,
	})
}

// createModel 创建模型配置
func (s *AdminServer) createModel(c *gin.Context) {
	var req CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	// 检查模型ID是否已存在
	if _, exists := s.config.GetModel(req.ID); exists {
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": fmt.Sprintf("模型 %s 已存在", req.ID),
		})
		return
	}

	// 创建新的模型配置
	newModel := &config.ModelConfig{
		ID:              req.ID,
		Name:            req.Name,
		Target:          req.Target,
		Prompt:          req.Prompt,
		Url:             req.Url,
		Type:            req.Type,
		PromptPath:      req.PromptPath,
		PromptValue:     req.PromptValue,
		PromptValueType: req.PromptValueType,
	}

	// 保存模型配置
	var err error
	if s.configService != nil {
		// 使用配置服务保存
		err = s.configService.SaveModel(newModel)
	} else {
		// 验证模型配置
		if err := newModel.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": fmt.Sprintf("模型配置验证失败: %v", err),
			})
			return
		}

		// 添加到内存配置
		s.config.Models[req.ID] = newModel

		// 保存到文件
		err = s.saveModelToFile(newModel)
		if err != nil {
			// 如果保存失败，从内存中移除
			delete(s.config.Models, req.ID)
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("保存模型配置失败: %v", err),
		})
		return
	}

	// 构建响应数据
	var response ModelResponse
	if s.configService != nil {
		// 从数据库获取包含时间信息的模型数据
		dbModel, err := s.configService.GetModelWithTime(newModel.ID)
		if err == nil {
			response = ModelResponse{
				ID:              newModel.ID,
				Name:            newModel.Name,
				Target:          newModel.Target,
				Prompt:          newModel.Prompt,
				Url:             newModel.Url,
				Type:            newModel.Type,
				PromptPath:      newModel.PromptPath,
				PromptValue:     newModel.PromptValue,
				PromptValueType: newModel.PromptValueType,
				CreatedAt:       dbModel.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:       dbModel.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
		} else {
			// 降级方案：无时间信息
			response = ModelResponse{
				ID:              newModel.ID,
				Name:            newModel.Name,
				Target:          newModel.Target,
				Prompt:          newModel.Prompt,
				Url:             newModel.Url,
				Type:            newModel.Type,
				PromptPath:      newModel.PromptPath,
				PromptValue:     newModel.PromptValue,
				PromptValueType: newModel.PromptValueType,
				CreatedAt:       "",
				UpdatedAt:       "",
			}
		}
	} else {
		// 无配置服务时的响应（无时间信息）
		response = ModelResponse{
			ID:              newModel.ID,
			Name:            newModel.Name,
			Target:          newModel.Target,
			Prompt:          newModel.Prompt,
			Url:             newModel.Url,
			Type:            newModel.Type,
			PromptPath:      newModel.PromptPath,
			PromptValue:     newModel.PromptValue,
			PromptValueType: newModel.PromptValueType,
			CreatedAt:       "",
			UpdatedAt:       "",
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "模型创建成功",
		"data":    response,
	})
}

// updateModel 根据模型ID配置模型信息
func (s *AdminServer) updateModel(c *gin.Context) {
	modelID := c.Param("id")

	model, exists := s.config.GetModel(modelID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": fmt.Sprintf("模型 %s 不存在", modelID),
		})
		return
	}

	var req UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	// 备份原始配置
	originalModel := *model

	// 更新字段（只更新非空字段，prompt相关字段可以为空）
	if req.Name != "" {
		model.Name = req.Name
	}
	if req.Target != "" {
		model.Target = req.Target
	}
	// Prompt相关字段允许为空，直接更新
	model.Prompt = req.Prompt
	model.PromptPath = req.PromptPath
	model.PromptValueType = req.PromptValueType
	model.PromptValue = req.PromptValue // 允许设置为nil来清空字段
	if req.Url != "" {
		model.Url = req.Url
	}
	if req.Type != "" {
		model.Type = req.Type
	}

	// 保存更新后的配置
	var err error
	if s.configService != nil {
		// 使用配置服务更新
		err = s.configService.UpdateModel(model)
	} else {
		// 验证更新后的配置
		if err := model.Validate(); err != nil {
			// 恢复原始配置
			*model = originalModel
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": fmt.Sprintf("模型配置验证失败: %v", err),
			})
			return
		}

		// 保存到文件
		err = s.saveModelToFile(model)
	}

	if err != nil {
		// 恢复原始配置
		*model = originalModel
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("保存模型配置失败: %v", err),
		})
		return
	}

	// 构建响应数据
	var response ModelResponse
	if s.configService != nil {
		// 从数据库获取包含时间信息的模型数据
		dbModel, err := s.configService.GetModelWithTime(model.ID)
		if err == nil {
			response = ModelResponse{
				ID:              model.ID,
				Name:            model.Name,
				Target:          model.Target,
				Prompt:          model.Prompt,
				Url:             model.Url,
				Type:            model.Type,
				PromptPath:      model.PromptPath,
				PromptValue:     model.PromptValue,
				PromptValueType: model.PromptValueType,
				CreatedAt:       dbModel.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:       dbModel.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
		} else {
			// 降级方案：无时间信息
			response = ModelResponse{
				ID:              model.ID,
				Name:            model.Name,
				Target:          model.Target,
				Prompt:          model.Prompt,
				Url:             model.Url,
				Type:            model.Type,
				PromptPath:      model.PromptPath,
				PromptValue:     model.PromptValue,
				PromptValueType: model.PromptValueType,
				CreatedAt:       "",
				UpdatedAt:       "",
			}
		}
	} else {
		// 无配置服务时的响应（无时间信息）
		response = ModelResponse{
			ID:              model.ID,
			Name:            model.Name,
			Target:          model.Target,
			Prompt:          model.Prompt,
			Url:             model.Url,
			Type:            model.Type,
			PromptPath:      model.PromptPath,
			PromptValue:     model.PromptValue,
			PromptValueType: model.PromptValueType,
			CreatedAt:       "",
			UpdatedAt:       "",
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "模型更新成功",
		"data":    response,
	})
}

// deleteModel 删除模型配置
func (s *AdminServer) deleteModel(c *gin.Context) {
	modelID := c.Param("id")

	_, exists := s.config.GetModel(modelID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": fmt.Sprintf("模型 %s 不存在", modelID),
		})
		return
	}

	// 删除模型配置
	var err error
	if s.configService != nil {
		// 使用配置服务删除
		err = s.configService.DeleteModel(modelID)
	} else {
		// 从内存中删除
		delete(s.config.Models, modelID)

		// 从文件中删除（重新保存所有配置）
		err = s.saveAllModelsToFile()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("删除模型配置失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "模型删除成功",
	})
}

// reloadConfig 重新加载配置
func (s *AdminServer) reloadConfig(c *gin.Context) {
	var err error
	if s.configService != nil {
		// 使用配置服务重新加载
		err = s.configService.ReloadConfig(s.configDir)
		if err == nil {
			// 更新本地配置引用
			s.config = s.configService.GetConfig()
		}
	} else {
		// 从文件重新加载
		newConfig, loadErr := config.LoadConfig(s.configDir)
		if loadErr != nil {
			err = loadErr
		} else {
			s.config.Models = newConfig.Models
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("重新加载配置失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "配置重新加载成功",
		"data": gin.H{
			"total_models": len(s.config.Models),
		},
	})
}

// getStatus 获取服务状态
func (s *AdminServer) getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"status":       "running",
			"total_models": len(s.config.Models),
			"config_dir":   s.configDir,
		},
	})
}

// getSystemConfig 获取系统配置
func (s *AdminServer) getSystemConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"proxy_port": s.proxyPort,
			"admin_port": s.adminPort,
		},
	})
}

// healthCheck 健康检查
func (s *AdminServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"timestamp": gin.H{
			"unix": gin.H{},
		},
	})
}

// setupEmbeddedStaticFiles 设置嵌入式静态文件服务
func (s *AdminServer) setupEmbeddedStaticFiles(r *gin.Engine) {
	// 创建子文件系统，去掉 "web" 前缀
	webSubFS, err := fs.Sub(webFS, "web")
	if err != nil {
		// 如果嵌入文件不可用，尝试使用外部文件
		r.Static("/static", "./web")
		r.StaticFile("/app.js", "./web/app.js")
		r.StaticFile("/admin", "./web/index.html")
		return
	}

	// 使用嵌入的文件系统
	r.StaticFS("/static", http.FS(webSubFS))

	// 单独处理 app.js
	r.GET("/app.js", func(c *gin.Context) {
		data, err := webFS.ReadFile("web/app.js")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Content-Type", "application/javascript")
		c.Data(http.StatusOK, "application/javascript", data)
	})

	// 单独处理 admin 路径
	r.GET("/admin", s.serveIndexHTML)
}

// serveIndexHTML 提供嵌入的 index.html
func (s *AdminServer) serveIndexHTML(c *gin.Context) {
	data, err := webFS.ReadFile("web/index.html")
	if err != nil {
		// 如果嵌入文件不可用，尝试使用外部文件
		c.File("./web/index.html")
		return
	}
	c.Header("Content-Type", "text/html")
	c.Data(http.StatusOK, "text/html", data)
}

// authMiddleware 认证中间件
func (s *AdminServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "未提供认证token",
			})
			c.Abort()
			return
		}

		// 检查Bearer前缀
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "认证token格式错误",
			})
			c.Abort()
			return
		}

		// 提取token
		tokenString := authHeader[7:]

		// 验证token
		claims, err := s.authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "认证token无效",
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("is_admin", claims.IsAdmin)

		c.Next()
	}
}

// checkInstall 检查是否首次安装
func (s *AdminServer) checkInstall(c *gin.Context) {
	isFirstInstall, err := s.authService.IsFirstInstall()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("检查安装状态失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"is_first_install": isFirstInstall,
		},
	})
}

// register 用户注册（仅首次安装）
func (s *AdminServer) register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	// 注册用户
	response, err := s.authService.Register(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "注册成功",
		"data":    response,
	})
}

// APIKeyResponse API Key响应结构
type APIKeyResponse struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	KeyValue   string `json:"key_value,omitempty"` // 只在创建时返回完整key
	KeyPreview string `json:"key_preview"`         // 显示用的预览（前几位+***）
	IsEnabled  bool   `json:"is_enabled"`
	LastUsedAt string `json:"last_used_at"`
	ExpiresAt  string `json:"expires_at"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// CreateAPIKeyRequest 创建API Key请求结构
type CreateAPIKeyRequest struct {
	Name      string `json:"name" binding:"required"`
	KeyValue  string `json:"key_value"` // 可选，如果不提供则自动生成
	ExpiresAt string `json:"expires_at"` // 可选的过期时间
}

// getAPIKeys 获取当前用户的API Key列表
func (s *AdminServer) getAPIKeys(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户信息不存在",
		})
		return
	}

	if s.authService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "认证服务不可用",
		})
		return
	}

	apiKeys, err := s.authService.GetAPIKeysByUserID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("获取API Key列表失败: %v", err),
		})
		return
	}

	var response []APIKeyResponse
	for _, apiKey := range apiKeys {
		// 生成key预览（显示前8位+***）
		keyPreview := ""
		if len(apiKey.KeyValue) > 8 {
			keyPreview = apiKey.KeyValue[:8] + "***"
		} else {
			keyPreview = apiKey.KeyValue + "***"
		}

		lastUsedAt := ""
		if apiKey.LastUsedAt != nil {
			lastUsedAt = apiKey.LastUsedAt.Format("2006-01-02T15:04:05Z07:00")
		}

		expiresAt := ""
		if apiKey.ExpiresAt != nil {
			expiresAt = apiKey.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
		}

		response = append(response, APIKeyResponse{
			ID:         apiKey.ID,
			Name:       apiKey.Name,
			KeyPreview: keyPreview,
			IsEnabled:  apiKey.IsEnabled,
			LastUsedAt: lastUsedAt,
			ExpiresAt:  expiresAt,
			CreatedAt:  apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:  apiKey.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"api_keys": response,
			"total":    len(response),
		},
	})
}

// createAPIKey 创建API Key
func (s *AdminServer) createAPIKey(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户信息不存在",
		})
		return
	}

	if s.authService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "认证服务不可用",
		})
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	// 如果没有提供KeyValue，则自动生成
	keyValue := req.KeyValue
	if keyValue == "" {
		keyValue = s.generateAPIKey()
	}

	// 创建API Key
	apiKey, err := s.authService.CreateAPIKey(userID.(uint), req.Name, keyValue, req.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("创建API Key失败: %v", err),
		})
		return
	}

	// 返回创建的API Key（包含完整key值）
	lastUsedAt := ""
	if apiKey.LastUsedAt != nil {
		lastUsedAt = apiKey.LastUsedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	expiresAtStr := ""
	if apiKey.ExpiresAt != nil {
		expiresAtStr = apiKey.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
	}

	response := APIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		KeyValue:  apiKey.KeyValue, // 创建时返回完整key
		IsEnabled: apiKey.IsEnabled,
		LastUsedAt: lastUsedAt,
		ExpiresAt: expiresAtStr,
		CreatedAt: apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: apiKey.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "API Key创建成功",
		"data":    response,
	})
}

// deleteAPIKey 删除API Key
func (s *AdminServer) deleteAPIKey(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户信息不存在",
		})
		return
	}

	if s.authService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "认证服务不可用",
		})
		return
	}

	idStr := c.Param("id")
	id, err := parseUint(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的API Key ID",
		})
		return
	}

	err = s.authService.DeleteAPIKey(uint(id), userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "API Key删除成功",
	})
}

// adminMiddleware 管理员权限中间件
func (s *AdminServer) adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "需要管理员权限",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// getUsers 获取用户列表
func (s *AdminServer) getUsers(c *gin.Context) {
	response, err := s.authService.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("获取用户列表失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    response,
	})
}

// createUser 创建用户
func (s *AdminServer) createUser(c *gin.Context) {
	var req service.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	// 获取创建者ID
	creatorID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户信息不存在",
		})
		return
	}

	response, err := s.authService.CreateUser(&req, creatorID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "用户创建成功",
		"data":    response,
	})
}

// updateUser 更新用户信息
func (s *AdminServer) updateUser(c *gin.Context) {
	userID := c.Param("id")
	id, err := parseUint(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户ID格式错误",
		})
		return
	}

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	err = s.authService.UpdateUser(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "用户信息更新成功",
	})
}

// deleteUser 删除用户
func (s *AdminServer) deleteUser(c *gin.Context) {
	userID := c.Param("id")
	id, err := parseUint(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户ID格式错误",
		})
		return
	}

	// 不允许删除自己
	currentUserID, exists := c.Get("user_id")
	if exists && currentUserID.(uint) == uint(id) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不能删除自己",
		})
		return
	}

	err = s.authService.DeleteUser(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "用户删除成功",
	})
}

// updateUserStatus 更新用户状态
func (s *AdminServer) updateUserStatus(c *gin.Context) {
	userID := c.Param("id")
	id, err := parseUint(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户ID格式错误",
		})
		return
	}

	var req struct {
		IsEnabled bool `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	// 不允许禁用自己
	currentUserID, exists := c.Get("user_id")
	if exists && currentUserID.(uint) == uint(id) && !req.IsEnabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不能禁用自己",
		})
		return
	}

	err = s.authService.UpdateUserStatus(uint(id), req.IsEnabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "用户状态更新成功",
	})
}

// adminChangePassword 管理员修改用户密码
func (s *AdminServer) adminChangePassword(c *gin.Context) {
	userID := c.Param("id")
	id, err := parseUint(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户ID格式错误",
		})
		return
	}

	var req service.AdminChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	err = s.authService.AdminChangePassword(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "密码修改成功",
	})
}

// changePassword 用户修改自己的密码
func (s *AdminServer) changePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户信息不存在",
		})
		return
	}

	var req service.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	err := s.authService.ChangePassword(userID.(uint), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "密码修改成功",
	})
}

// parseUint 解析字符串为uint
func parseUint(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 32)
}

// generateAPIKey 生成API Key
func (s *AdminServer) generateAPIKey() string {
	// 生成32字节的随机数据
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		// 如果随机数生成失败，使用时间戳作为后备方案
		return fmt.Sprintf("ak_%d", time.Now().UnixNano())
	}
	// 转换为十六进制字符串并添加前缀
	return "ak_" + hex.EncodeToString(bytes)
}

// login 用户登录
func (s *AdminServer) login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	// 用户登录
	response, err := s.authService.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "登录成功",
		"data":    response,
	})
}

// logout 用户注销
func (s *AdminServer) logout(c *gin.Context) {
	// 简单的注销响应，客户端需要删除本地token
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "注销成功",
	})
}

// getProfile 获取用户信息
func (s *AdminServer) getProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户信息不存在",
		})
		return
	}

	// 从数据库获取用户信息
	user, err := s.authService.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("获取用户信息失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    user,
	})
}

// getPublicKey 获取RSA公钥
func (s *AdminServer) getPublicKey(c *gin.Context) {
	response, err := s.authService.GetPublicKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("获取公钥失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    response,
	})
}

// encryptedLogin 加密用户登录
func (s *AdminServer) encryptedLogin(c *gin.Context) {
	var req service.EncryptedLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	// 加密登录
	response, err := s.authService.EncryptedLogin(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "登录成功",
		"data":    response,
	})
}

// encryptedRegister 加密用户注册（仅首次安装）
func (s *AdminServer) encryptedRegister(c *gin.Context) {
	var req service.EncryptedRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	// 加密注册
	response, err := s.authService.EncryptedRegister(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "注册成功",
		"data":    response,
	})
}
