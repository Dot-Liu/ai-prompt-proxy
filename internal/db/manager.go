package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	// 确保数据库目录存在
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 设置数据库文件路径
	dbFile := filepath.Join(dbPath, "config.db")

	// 打开数据库连接
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
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
	return m.db.AutoMigrate(&ModelConfigDB{}, &ConfigMetadata{}, &User{}, &APIKey{})
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

// CreateUser 创建用户
func (m *Manager) CreateUser(user *User) error {
	result := m.db.Create(user)
	if result.Error != nil {
		return fmt.Errorf("创建用户失败: %w", result.Error)
	}
	return nil
}

// GetUserByUsername 根据用户名获取用户
func (m *Manager) GetUserByUsername(username string) (*User, error) {
	var user User
	result := m.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在: %s", username)
		}
		return nil, fmt.Errorf("获取用户失败: %w", result.Error)
	}
	return &user, nil
}

// GetUserByID 根据ID获取用户
func (m *Manager) GetUserByID(id uint) (*User, error) {
	var user User
	result := m.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在: %d", id)
		}
		return nil, fmt.Errorf("获取用户失败: %w", result.Error)
	}
	return &user, nil
}

// UpdateUser 更新用户信息
func (m *Manager) UpdateUser(user *User) error {
	result := m.db.Save(user)
	if result.Error != nil {
		return fmt.Errorf("更新用户失败: %w", result.Error)
	}
	return nil
}

// GetUserCount 获取用户总数
func (m *Manager) GetUserCount() (int64, error) {
	var count int64
	result := m.db.Model(&User{}).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("获取用户数量失败: %w", result.Error)
	}
	return count, nil
}

// GetAllUsers 获取所有用户列表
func (m *Manager) GetAllUsers() ([]User, error) {
	var users []User
	result := m.db.Order("created_at DESC").Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("获取用户列表失败: %w", result.Error)
	}
	return users, nil
}

// DeleteUser 删除用户
func (m *Manager) DeleteUser(id uint) error {
	result := m.db.Where("id = ?", id).Delete(&User{})
	if result.Error != nil {
		return fmt.Errorf("删除用户失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不存在: %d", id)
	}
	return nil
}

// UpdateUserStatus 更新用户状态
func (m *Manager) UpdateUserStatus(id uint, isEnabled bool) error {
	result := m.db.Model(&User{}).Where("id = ?", id).Update("is_enabled", isEnabled)
	if result.Error != nil {
		return fmt.Errorf("更新用户状态失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不存在: %d", id)
	}
	return nil
}

// UpdateUserPassword 更新用户密码
func (m *Manager) UpdateUserPassword(id uint, hashedPassword string) error {
	result := m.db.Model(&User{}).Where("id = ?", id).Update("password", hashedPassword)
	if result.Error != nil {
		return fmt.Errorf("更新用户密码失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不存在: %d", id)
	}
	return nil
}

// UpdateUserLastLogin 更新用户最后登录时间
func (m *Manager) UpdateUserLastLogin(id uint) error {
	now := time.Now()
	result := m.db.Model(&User{}).Where("id = ?", id).Update("last_login_at", &now)
	if result.Error != nil {
		return fmt.Errorf("更新用户最后登录时间失败: %w", result.Error)
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

// CreateAPIKey 创建API Key
func (m *Manager) CreateAPIKey(apiKey *APIKey) error {
	result := m.db.Create(apiKey)
	if result.Error != nil {
		return fmt.Errorf("创建API Key失败: %w", result.Error)
	}
	return nil
}

// GetAPIKeysByUserID 根据用户ID获取API Key列表
func (m *Manager) GetAPIKeysByUserID(userID uint) ([]APIKey, error) {
	var apiKeys []APIKey
	result := m.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&apiKeys)
	if result.Error != nil {
		return nil, fmt.Errorf("获取API Key列表失败: %w", result.Error)
	}
	return apiKeys, nil
}

// GetAPIKeyByValue 根据Key值获取API Key
func (m *Manager) GetAPIKeyByValue(keyValue string) (*APIKey, error) {
	var apiKey APIKey
	result := m.db.Where("key_value = ? AND is_enabled = ?", keyValue, true).First(&apiKey)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("API Key不存在或已禁用")
		}
		return nil, fmt.Errorf("获取API Key失败: %w", result.Error)
	}
	return &apiKey, nil
}

// GetAPIKeyByID 根据ID获取API Key
func (m *Manager) GetAPIKeyByID(id uint) (*APIKey, error) {
	var apiKey APIKey
	result := m.db.Where("id = ?", id).First(&apiKey)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("API Key不存在: %d", id)
		}
		return nil, fmt.Errorf("获取API Key失败: %w", result.Error)
	}
	return &apiKey, nil
}

// UpdateAPIKey 更新API Key
func (m *Manager) UpdateAPIKey(apiKey *APIKey) error {
	result := m.db.Save(apiKey)
	if result.Error != nil {
		return fmt.Errorf("更新API Key失败: %w", result.Error)
	}
	return nil
}

// DeleteAPIKey 删除API Key
func (m *Manager) DeleteAPIKey(id uint, userID uint) error {
	result := m.db.Where("id = ? AND user_id = ?", id, userID).Delete(&APIKey{})
	if result.Error != nil {
		return fmt.Errorf("删除API Key失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("API Key不存在或无权限删除: %d", id)
	}
	return nil
}

// UpdateAPIKeyLastUsed 更新API Key最后使用时间
func (m *Manager) UpdateAPIKeyLastUsed(keyValue string) error {
	now := time.Now()
	result := m.db.Model(&APIKey{}).Where("key_value = ?", keyValue).Update("last_used_at", &now)
	if result.Error != nil {
		return fmt.Errorf("更新API Key最后使用时间失败: %w", result.Error)
	}
	return nil
}
