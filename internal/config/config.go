package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ModelType 模型类型
type ModelType string

type ValueType string

const (
	ModelTypeChat  ModelType = "chat"
	ModelTypeImage ModelType = "image"
	ModelTypeAudio ModelType = "audio"
	ModelTypeVideo ModelType = "video"

	ValueTypeString ValueType = "string"
	ValueTypeArray  ValueType = "array"
	ValueTypeObject ValueType = "object"
)

// ModelConfig 模型配置
type ModelConfig struct {
	ID              string      `yaml:"id"`           // 模型ID
	Name            string      `yaml:"name"`         // 模型名称
	Target          string      `yaml:"target"`       // 目标模型ID
	Prompt          string      `yaml:"prompt"`       // Prompt描述
	Url             string      `yaml:"url"`          // 转发的URL
	Type            ModelType   `yaml:"type"`         // 模型类型
	PromptPath      string      `yaml:"prompt_path"`  // Prompt插入位置(JSON Path)
	PromptValue     interface{} `yaml:"prompt_value"` // Prompt值
	PromptValueType ValueType   `yaml:"prompt_type"`  // Prompt值类型
}

func (m *ModelConfig) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("模型ID不能为空")
	}
	if m.Name == "" {
		return fmt.Errorf("模型名称不能为空")
	}
	if m.Target == "" {
		return fmt.Errorf("目标模型ID不能为空")
	}
	if m.Url == "" {
		return fmt.Errorf("转发的URL不能为空")
	}
	u, err := url.Parse(m.Url)
	if err != nil {
		return fmt.Errorf("转发的URL无效: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("转发的URL无效: %s", m.Url)
	}
	if m.Type == "" {
		m.Type = ModelTypeChat
	}

	// 验证模型类型
	switch m.Type {
	case ModelTypeChat, ModelTypeImage, ModelTypeAudio, ModelTypeVideo:
		// 有效类型
	default:
		return fmt.Errorf("无效的模型类型: %s", m.Type)
	}

	if m.PromptPath == "" {
		switch m.Type {
		case ModelTypeChat:
			m.PromptPath = "messages"
			if m.PromptValue == nil {
				m.PromptValue = map[string]interface{}{
					"role":    "system",
					"content": m.PromptValue,
				}
			}
		case ModelTypeImage:
			m.PromptPath = "prompt"
		}
	}

	return nil
}

// Config 全局配置
type Config struct {
	Models map[string]*ModelConfig `yaml:"models"`
	dbPath string                  // 数据库路径
}

// LoadConfig 从指定目录加载配置文件
func LoadConfig(configDir string) (*Config, error) {
	config := &Config{
		Models: make(map[string]*ModelConfig),
	}

	// 读取配置目录下的所有yaml文件
	files, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("读取配置目录失败: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// 只处理yaml和yml文件
		if !strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml") {
			continue
		}

		filePath := filepath.Join(configDir, file.Name())
		if err := loadConfigFile(filePath, config); err != nil {
			return nil, fmt.Errorf("加载配置文件 %s 失败: %w", filePath, err)
		}
	}

	return config, nil
}

// LoadConfigWithDB 从指定目录加载配置文件并支持数据库
func LoadConfigWithDB(configDir string) (*Config, error) {
	// 首先尝试从数据库加载
	dbPath := filepath.Join(configDir, "db")
	config := &Config{
		Models: make(map[string]*ModelConfig),
		dbPath: dbPath,
	}

	// 尝试从数据库加载配置
	if err := config.loadFromDB(); err != nil {
		fmt.Printf("从数据库加载配置失败，将从YAML文件加载: %v\n", err)

		// 如果数据库加载失败，从YAML文件加载
		yamlConfig, err := LoadConfig(configDir)
		if err != nil {
			return nil, err
		}

		// 设置数据库路径
		yamlConfig.SetDBPath(dbPath)

		// 将YAML配置迁移到数据库
		if err := yamlConfig.migrateToDB(); err != nil {
			fmt.Printf("迁移配置到数据库失败: %v\n", err)
		}

		return yamlConfig, nil
	}

	return config, nil
}

// loadConfigFile 加载单个配置文件
func loadConfigFile(filePath string, config *Config) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var fileConfig struct {
		Models []ModelConfig `yaml:"models"`
	}

	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return err
	}

	// 将模型配置添加到全局配置中
	for i := range fileConfig.Models {
		model := &fileConfig.Models[i]
		if err := model.Validate(); err != nil {
			return fmt.Errorf("模型配置验证失败 %s: %w", model.ID, err)
		}
		config.Models[model.ID] = model
	}

	return nil
}

// GetModel 根据模型ID获取模型配置
func (c *Config) GetModel(modelID string) (*ModelConfig, bool) {
	model, exists := c.Models[modelID]
	return model, exists
}

// SetDBPath 设置数据库路径
func (c *Config) SetDBPath(dbPath string) {
	c.dbPath = dbPath
}

// GetDBPath 获取数据库路径
func (c *Config) GetDBPath() string {
	return c.dbPath
}

// AddModel 添加模型配置
func (c *Config) AddModel(model *ModelConfig) {
	if c.Models == nil {
		c.Models = make(map[string]*ModelConfig)
	}
	c.Models[model.ID] = model
}

// RemoveModel 移除模型配置
func (c *Config) RemoveModel(modelID string) bool {
	if _, exists := c.Models[modelID]; exists {
		delete(c.Models, modelID)
		return true
	}
	return false
}

// UpdateModel 更新模型配置
func (c *Config) UpdateModel(model *ModelConfig) bool {
	if _, exists := c.Models[model.ID]; exists {
		c.Models[model.ID] = model
		return true
	}
	return false
}

// loadFromDB 从数据库加载配置
func (c *Config) loadFromDB() error {
	// 这里需要导入db包，但为了避免循环依赖，我们先创建一个简单的实现
	// 实际实现会在后面的重构中完成
	return fmt.Errorf("数据库功能暂未实现")
}

// migrateToDB 将配置迁移到数据库
func (c *Config) migrateToDB() error {
	// 这里需要导入db包，但为了避免循环依赖，我们先创建一个简单的实现
	// 实际实现会在后面的重构中完成
	return fmt.Errorf("数据库迁移功能暂未实现")
}
