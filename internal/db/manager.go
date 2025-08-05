package db

import (
	"fmt"
	"path/filepath"

	"github.com/eolinker/ai-prompt-proxy/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Manager 数据库管理器
type Manager struct {
	db *gorm.DB
}

// NewManager 创建数据库管理器
func NewManager(dbPath string) (*Manager, error) {
	// 确保数据库文件路径存在
	dbFile := filepath.Join(dbPath, "config.db")

	// 连接SQLite数据库
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 静默模式，减少日志输出
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	manager := &Manager{db: db}

	// 自动迁移数据库表
	if err := manager.migrate(); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return manager, nil
}

// migrate 执行数据库迁移
func (m *Manager) migrate() error {
	return m.db.AutoMigrate(&ModelConfigDB{}, &ConfigMetadata{})
}

// SaveModelConfig 保存模型配置
func (m *Manager) SaveModelConfig(cfg *config.ModelConfig) error {
	dbModel := &ModelConfigDB{}
	if err := dbModel.FromModelConfig(cfg); err != nil {
		return fmt.Errorf("转换模型配置失败: %w", err)
	}

	// 使用UPSERT操作（如果存在则更新，否则创建）
	result := m.db.Save(dbModel)
	if result.Error != nil {
		return fmt.Errorf("保存模型配置失败: %w", result.Error)
	}

	return nil
}

// GetModelConfig 获取模型配置
func (m *Manager) GetModelConfig(id string) (*config.ModelConfig, error) {
	var dbModel ModelConfigDB
	result := m.db.Where("id = ?", id).First(&dbModel)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("模型配置不存在: %s", id)
		}
		return nil, fmt.Errorf("获取模型配置失败: %w", result.Error)
	}

	return dbModel.ToModelConfig()
}

// GetModelConfigWithTime 获取模型配置（包含时间信息）
func (m *Manager) GetModelConfigWithTime(id string) (*ModelConfigDB, error) {
	var dbModel ModelConfigDB
	result := m.db.Where("id = ?", id).First(&dbModel)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("模型配置不存在: %s", id)
		}
		return nil, fmt.Errorf("获取模型配置失败: %w", result.Error)
	}

	return &dbModel, nil
}

// GetAllModelConfigs 获取所有模型配置
func (m *Manager) GetAllModelConfigs() (map[string]*config.ModelConfig, error) {
	var dbModels []ModelConfigDB
	result := m.db.Find(&dbModels)
	if result.Error != nil {
		return nil, fmt.Errorf("获取所有模型配置失败: %w", result.Error)
	}

	configs := make(map[string]*config.ModelConfig)
	for _, dbModel := range dbModels {
		cfg, err := dbModel.ToModelConfig()
		if err != nil {
			return nil, fmt.Errorf("转换模型配置失败 %s: %w", dbModel.ID, err)
		}
		configs[cfg.ID] = cfg
	}

	return configs, nil
}

// GetAllModelConfigsWithTime 获取所有模型配置（包含时间信息）
func (m *Manager) GetAllModelConfigsWithTime() ([]ModelConfigDB, error) {
	var dbModels []ModelConfigDB
	result := m.db.Order("updated_at DESC").Find(&dbModels)
	if result.Error != nil {
		return nil, fmt.Errorf("获取所有模型配置失败: %w", result.Error)
	}

	return dbModels, nil
}

// DeleteModelConfig 删除模型配置
func (m *Manager) DeleteModelConfig(id string) error {
	result := m.db.Where("id = ?", id).Delete(&ModelConfigDB{})
	if result.Error != nil {
		return fmt.Errorf("删除模型配置失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("模型配置不存在: %s", id)
	}
	return nil
}

// UpdateModelConfig 更新模型配置
func (m *Manager) UpdateModelConfig(cfg *config.ModelConfig) error {
	// 先检查模型是否存在
	var existing ModelConfigDB
	result := m.db.Where("id = ?", cfg.ID).First(&existing)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return fmt.Errorf("模型配置不存在: %s", cfg.ID)
		}
		return fmt.Errorf("查询模型配置失败: %w", result.Error)
	}

	// 更新配置
	dbModel := &ModelConfigDB{}
	if err := dbModel.FromModelConfig(cfg); err != nil {
		return fmt.Errorf("转换模型配置失败: %w", err)
	}

	// 使用Select明确指定要更新的字段，包括可能为空的字段
	result = m.db.Model(&existing).Select("name", "target", "prompt", "url", "type", "prompt_path", "prompt_value", "prompt_value_type").Updates(dbModel)
	if result.Error != nil {
		return fmt.Errorf("更新模型配置失败: %w", result.Error)
	}

	return nil
}

// GetMetadata 获取配置元数据
func (m *Manager) GetMetadata(key string) (string, error) {
	var metadata ConfigMetadata
	result := m.db.Where("key = ?", key).First(&metadata)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("元数据不存在: %s", key)
		}
		return "", fmt.Errorf("获取元数据失败: %w", result.Error)
	}
	return metadata.Value, nil
}

// SetMetadata 设置配置元数据
func (m *Manager) SetMetadata(key, value string) error {
	metadata := ConfigMetadata{
		Key:   key,
		Value: value,
	}
	result := m.db.Save(&metadata)
	if result.Error != nil {
		return fmt.Errorf("设置元数据失败: %w", result.Error)
	}
	return nil
}

// Close 关闭数据库连接
func (m *Manager) Close() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB 获取原始数据库连接（用于高级操作）
func (m *Manager) GetDB() *gorm.DB {
	return m.db
}
