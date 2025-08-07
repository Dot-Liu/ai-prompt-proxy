package db

import (
	"encoding/json"
	"time"

	"github.com/eolinker/ai-prompt-proxy/internal/config"
)

// ModelConfigDB 数据库中的模型配置表
type ModelConfigDB struct {
	ID              string    `gorm:"primaryKey;column:id" json:"id"`
	Name            string    `gorm:"column:name;not null" json:"name"`
	Target          string    `gorm:"column:target;not null" json:"target"`
	Prompt          string    `gorm:"column:prompt" json:"prompt"`
	Url             string    `gorm:"column:url;not null" json:"url"`
	Type            string    `gorm:"column:type;not null" json:"type"`
	PromptPath      string    `gorm:"column:prompt_path" json:"prompt_path"`
	PromptValue     string    `gorm:"column:prompt_value;type:text" json:"prompt_value"` // JSON字符串
	PromptValueType string    `gorm:"column:prompt_value_type" json:"prompt_value_type"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (ModelConfigDB) TableName() string {
	return "model_configs"
}

// ToModelConfig 转换为配置模型
func (m *ModelConfigDB) ToModelConfig() (*config.ModelConfig, error) {
	var promptValue interface{}
	if m.PromptValue != "" {
		if err := json.Unmarshal([]byte(m.PromptValue), &promptValue); err != nil {
			// 如果JSON解析失败，当作字符串处理
			promptValue = m.PromptValue
		}
	}

	return &config.ModelConfig{
		ID:              m.ID,
		Name:            m.Name,
		Target:          m.Target,
		Prompt:          m.Prompt,
		Url:             m.Url,
		Type:            config.ModelType(m.Type),
		PromptPath:      m.PromptPath,
		PromptValue:     promptValue,
		PromptValueType: config.ValueType(m.PromptValueType),
	}, nil
}

// FromModelConfig 从配置模型创建数据库模型
func (m *ModelConfigDB) FromModelConfig(cfg *config.ModelConfig) error {
	m.ID = cfg.ID
	m.Name = cfg.Name
	m.Target = cfg.Target
	m.Prompt = cfg.Prompt
	m.Url = cfg.Url
	m.Type = string(cfg.Type)
	m.PromptPath = cfg.PromptPath
	m.PromptValueType = string(cfg.PromptValueType)

	// 将PromptValue序列化为JSON字符串
	if cfg.PromptValue != nil {
		promptValueBytes, err := json.Marshal(cfg.PromptValue)
		if err != nil {
			return err
		}
		m.PromptValue = string(promptValueBytes)
	} else {
		// 如果PromptValue为nil，明确设置为空字符串以清空数据库字段
		m.PromptValue = ""
	}

	return nil
}

// ConfigMetadata 配置元数据表
type ConfigMetadata struct {
	Key       string    `gorm:"primaryKey;column:key" json:"key"`
	Value     string    `gorm:"column:value;type:text" json:"value"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (ConfigMetadata) TableName() string {
	return "config_metadata"
}

// User 用户表
type User struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Username    string     `gorm:"column:username;uniqueIndex;not null" json:"username"`
	Password    string     `gorm:"column:password;not null" json:"-"` // 不在JSON中显示密码
	IsAdmin     bool       `gorm:"column:is_admin;default:false" json:"is_admin"`
	IsEnabled   bool       `gorm:"column:is_enabled;default:true" json:"is_enabled"` // 用户是否启用
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"last_login_at"`        // 最后登录时间
	CreatedBy   uint       `gorm:"column:created_by;default:0" json:"created_by"`    // 创建者ID，0表示系统创建
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// APIKey API密钥表
type APIKey struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint      `gorm:"column:user_id;not null;index" json:"user_id"`           // 所属用户ID
	Name        string    `gorm:"column:name;not null" json:"name"`                       // API Key名称/描述
	KeyValue    string    `gorm:"column:key_value;uniqueIndex;not null" json:"key_value"` // API Key值
	IsEnabled   bool      `gorm:"column:is_enabled;default:true" json:"is_enabled"`       // 是否启用
	LastUsedAt  *time.Time `gorm:"column:last_used_at" json:"last_used_at"`               // 最后使用时间
	ExpiresAt   *time.Time `gorm:"column:expires_at" json:"expires_at"`                   // 过期时间，null表示永不过期
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	
	// 关联用户
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (APIKey) TableName() string {
	return "api_keys"
}
