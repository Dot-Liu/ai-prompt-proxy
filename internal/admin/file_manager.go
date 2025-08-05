package admin

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	
	"github.com/eolinker/ai-prompt-proxy/internal/config"
	"gopkg.in/yaml.v3"
)

// saveModelToFile 保存单个模型到文件
func (s *AdminServer) saveModelToFile(model *config.ModelConfig) error {
	// 创建文件名（基于模型ID）
	filename := fmt.Sprintf("%s.yaml", model.ID)
	filePath := filepath.Join(s.configDir, filename)
	
	// 创建文件配置结构
	fileConfig := struct {
		Models []config.ModelConfig `yaml:"models"`
	}{
		Models: []config.ModelConfig{*model},
	}
	
	// 序列化为YAML
	data, err := yaml.Marshal(fileConfig)
	if err != nil {
		return fmt.Errorf("序列化模型配置失败: %w", err)
	}
	
	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	
	return nil
}

// saveAllModelsToFile 保存所有模型到文件
func (s *AdminServer) saveAllModelsToFile() error {
	// 按模型类型分组保存
	modelGroups := make(map[config.ModelType][]config.ModelConfig)
	
	for _, model := range s.config.Models {
		modelGroups[model.Type] = append(modelGroups[model.Type], *model)
	}
	
	// 为每个模型类型创建文件
	for modelType, models := range modelGroups {
		filename := fmt.Sprintf("%s-models.yaml", modelType)
		filePath := filepath.Join(s.configDir, filename)
		
		fileConfig := struct {
			Models []config.ModelConfig `yaml:"models"`
		}{
			Models: models,
		}
		
		data, err := yaml.Marshal(fileConfig)
		if err != nil {
			return fmt.Errorf("序列化%s模型配置失败: %w", modelType, err)
		}
		
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return fmt.Errorf("写入%s配置文件失败: %w", modelType, err)
		}
	}
	
	return nil
}

// backupConfig 备份配置文件
func (s *AdminServer) backupConfig() error {
	backupDir := filepath.Join(s.configDir, "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}
	
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("config_backup_%s.yaml", timestamp))
	
	// 创建完整的配置备份
	allModels := make([]config.ModelConfig, 0, len(s.config.Models))
	for _, model := range s.config.Models {
		allModels = append(allModels, *model)
	}
	
	fileConfig := struct {
		Models []config.ModelConfig `yaml:"models"`
	}{
		Models: allModels,
	}
	
	data, err := yaml.Marshal(fileConfig)
	if err != nil {
		return fmt.Errorf("序列化备份配置失败: %w", err)
	}
	
	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return fmt.Errorf("写入备份文件失败: %w", err)
	}
	
	return nil
}
