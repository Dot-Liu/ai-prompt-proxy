package admin

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

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
}

// NewAdminServer 创建新的管理API服务器
func NewAdminServer(cfg *config.Config, configDir string) *AdminServer {
	return &AdminServer{
		config:    cfg,
		configDir: configDir,
	}
}

// NewAdminServerWithService 使用配置服务创建新的管理API服务器
func NewAdminServerWithService(configService *service.ConfigService, configDir string) *AdminServer {
	return &AdminServer{
		config:        configService.GetConfig(),
		configDir:     configDir,
		configService: configService,
	}
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
		// 模型相关API
		models := api.Group("/models")
		{
			models.GET("", s.getModels)          // 获取模型列表
			models.GET("/:id", s.getModel)       // 根据模型ID获取模型信息
			models.PUT("/:id", s.updateModel)    // 根据模型ID配置模型信息
			models.POST("", s.createModel)       // 创建模型配置
			models.DELETE("/:id", s.deleteModel) // 删除模型配置
		}

		// 配置相关API
		config := api.Group("/config")
		{
			config.POST("/reload", s.reloadConfig) // 重新加载配置
			config.GET("/status", s.getStatus)     // 获取服务状态
		}
	}

	return r.Run(":" + port)
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
	model.PromptValue = req.PromptValue  // 允许设置为nil来清空字段
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
