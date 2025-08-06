package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eolinker/ai-prompt-proxy/internal/config"
	"github.com/eolinker/ai-prompt-proxy/internal/db"
)

// ConfigService 配置服务
type ConfigService struct {
	config *config.Config
	db     *db.Manager
}

// NewConfigService 创建配置服务
func NewConfigService(configDir string) (*ConfigService, error) {
	// 创建数据库管理器
	dbPath := filepath.Join(configDir, "db")
	database, err := db.NewManager(dbPath)
	if err != nil {
		return nil, fmt.Errorf("创建数据库管理器失败: %w", err)
	}

	service := &ConfigService{
		config: &config.Config{
			Models: make(map[string]*config.ModelConfig),
		},
		db: database,
	}

	// 加载配置
	if err := service.LoadConfig(configDir); err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	return service, nil
}

// LoadConfig 加载配置
func (s *ConfigService) LoadConfig(configDir string) error {
	// 首先尝试从数据库加载
	dbConfigs, err := s.db.GetAllModelConfigs()
	if err == nil && len(dbConfigs) > 0 {
		s.config.Models = dbConfigs
		fmt.Printf("从数据库加载了 %d 个模型配置\n", len(dbConfigs))
		return nil
	}

	fmt.Printf("数据库中没有配置，从YAML文件加载: %v\n", err)

	// 从YAML文件加载
	yamlConfig, err := config.LoadConfig(configDir)
	if err != nil {
		return fmt.Errorf("从YAML文件加载配置失败: %w", err)
	}

	s.config.Models = yamlConfig.Models

	// 将YAML配置迁移到数据库
	if err := s.MigrateYAMLToDB(); err != nil {
		fmt.Printf("迁移YAML配置到数据库失败: %v\n", err)
	} else {
		fmt.Printf("成功迁移 %d 个模型配置到数据库\n", len(s.config.Models))
	}

	return nil
}

// MigrateYAMLToDB 将YAML配置迁移到数据库
func (s *ConfigService) MigrateYAMLToDB() error {
	for _, model := range s.config.Models {
		if err := s.db.SaveModelConfig(model); err != nil {
			return fmt.Errorf("保存模型配置 %s 到数据库失败: %w", model.ID, err)
		}
	}
	return nil
}

// GetConfig 获取配置
func (s *ConfigService) GetConfig() *config.Config {
	return s.config
}

// GetDBManager 获取数据库管理器
func (s *ConfigService) GetDBManager() *db.Manager {
	return s.db
}

// GetModel 获取模型配置
func (s *ConfigService) GetModel(modelID string) (*config.ModelConfig, bool) {
	return s.config.GetModel(modelID)
}

// GetAllModelsWithTime 获取所有模型配置（包含时间信息）
func (s *ConfigService) GetAllModelsWithTime() ([]db.ModelConfigDB, error) {
	return s.db.GetAllModelConfigsWithTime()
}

// GetModelWithTime 获取单个模型配置（包含时间信息）
func (s *ConfigService) GetModelWithTime(modelID string) (*db.ModelConfigDB, error) {
	return s.db.GetModelConfigWithTime(modelID)
}

// SaveModel 保存模型配置
func (s *ConfigService) SaveModel(model *config.ModelConfig) error {
	// 验证模型配置
	if err := model.Validate(); err != nil {
		return fmt.Errorf("模型配置验证失败: %w", err)
	}

	// 保存到数据库
	if err := s.db.SaveModelConfig(model); err != nil {
		return fmt.Errorf("保存模型配置到数据库失败: %w", err)
	}

	// 更新内存中的配置
	s.config.AddModel(model)

	return nil
}

// UpdateModel 更新模型配置
func (s *ConfigService) UpdateModel(model *config.ModelConfig) error {
	// 验证模型配置
	if err := model.Validate(); err != nil {
		return fmt.Errorf("模型配置验证失败: %w", err)
	}

	// 更新数据库
	if err := s.db.UpdateModelConfig(model); err != nil {
		return fmt.Errorf("更新数据库中的模型配置失败: %w", err)
	}

	// 更新内存中的配置
	s.config.UpdateModel(model)

	return nil
}

// DeleteModel 删除模型配置
func (s *ConfigService) DeleteModel(modelID string) error {
	// 检查模型是否存在
	if _, exists := s.config.GetModel(modelID); !exists {
		return fmt.Errorf("模型 %s 不存在", modelID)
	}

	// 从数据库删除
	if err := s.db.DeleteModelConfig(modelID); err != nil {
		return fmt.Errorf("从数据库删除模型配置失败: %w", err)
	}

	// 从内存中删除
	s.config.RemoveModel(modelID)

	return nil
}

// ReloadConfig 重新加载配置
func (s *ConfigService) ReloadConfig(configDir string) error {
	return s.LoadConfig(configDir)
}

// Close 关闭服务
func (s *ConfigService) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// BackupToYAML 备份配置到YAML文件
func (s *ConfigService) BackupToYAML(backupDir string) error {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 按模型类型分组保存
	modelsByType := make(map[string][]*config.ModelConfig)
	for _, model := range s.config.Models {
		modelType := string(model.Type)
		modelsByType[modelType] = append(modelsByType[modelType], model)
	}

	// 为每种类型创建一个YAML文件
	for modelType, models := range modelsByType {
		filename := fmt.Sprintf("%s-models-backup.yaml", modelType)
		filepath := filepath.Join(backupDir, filename)

		if err := s.saveModelsToYAML(models, filepath); err != nil {
			return fmt.Errorf("保存 %s 模型到文件失败: %w", modelType, err)
		}
	}

	return nil
}

// saveModelsToYAML 保存模型列表到YAML文件
func (s *ConfigService) saveModelsToYAML(models []*config.ModelConfig, filepath string) error {
	// 这里可以实现YAML保存逻辑
	// 为了简化，我们先创建一个占位符实现
	content := "# 模型配置备份文件\n"
	content += "# 生成时间: " + fmt.Sprintf("%v", models) + "\n"
	content += "models:\n"

	for _, model := range models {
		content += fmt.Sprintf("  - id: %s\n", model.ID)
		content += fmt.Sprintf("    name: %s\n", model.Name)
		content += fmt.Sprintf("    target: %s\n", model.Target)
		content += fmt.Sprintf("    prompt: %v\n", model.PromptValue)
		content += fmt.Sprintf("    url: %s\n", model.Url)
		content += fmt.Sprintf("    type: %s\n", model.Type)
		if model.PromptPath != "" {
			content += fmt.Sprintf("    prompt_path: %s\n", model.PromptPath)
		}
		if model.PromptValueType != "" {
			content += fmt.Sprintf("    prompt_type: %s\n", model.PromptValueType)
		}
		content += "\n"
	}

	return os.WriteFile(filepath, []byte(content), 0644)
}
