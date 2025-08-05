package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eolinker/ai-prompt-proxy/internal/config"
)

// SQLiteDB SQLite数据库管理器
type SQLiteDB struct {
	db   *sql.DB
	path string
}

// NewSQLiteDB 创建SQLite数据库管理器
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	// 确保目录存在
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	dbFile := filepath.Join(dbPath, "config.db")

	// 由于依赖问题，我们先创建一个文件存储的实现
	// 后续可以在网络正常时替换为真正的SQLite
	return &SQLiteDB{
		path: dbFile,
	}, nil
}

// Init 初始化数据库
func (s *SQLiteDB) Init() error {
	// 创建配置存储目录
	configDir := filepath.Dir(s.path)
	modelsDir := filepath.Join(configDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return fmt.Errorf("创建模型配置目录失败: %w", err)
	}
	return nil
}

// SaveModelConfig 保存模型配置
func (s *SQLiteDB) SaveModelConfig(cfg *config.ModelConfig) error {
	modelsDir := filepath.Join(filepath.Dir(s.path), "models")
	modelFile := filepath.Join(modelsDir, cfg.ID+".json")

	// 创建数据库模型
	dbModel := &ModelRecord{
		ID:              cfg.ID,
		Name:            cfg.Name,
		Target:          cfg.Target,
		Url:             cfg.Url,
		Type:            string(cfg.Type),
		PromptPath:      cfg.PromptPath,
		PromptValueType: string(cfg.PromptValueType),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// 序列化PromptValue
	if cfg.PromptValue != nil {
		promptValueBytes, err := json.Marshal(cfg.PromptValue)
		if err != nil {
			return fmt.Errorf("序列化PromptValue失败: %w", err)
		}
		dbModel.PromptValue = string(promptValueBytes)
	}

	// 保存到文件
	data, err := json.MarshalIndent(dbModel, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化模型配置失败: %w", err)
	}

	if err := os.WriteFile(modelFile, data, 0644); err != nil {
		return fmt.Errorf("保存模型配置文件失败: %w", err)
	}

	return nil
}

// GetModelConfig 获取模型配置
func (s *SQLiteDB) GetModelConfig(id string) (*config.ModelConfig, error) {
	modelsDir := filepath.Join(filepath.Dir(s.path), "models")
	modelFile := filepath.Join(modelsDir, id+".json")

	data, err := os.ReadFile(modelFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("模型配置不存在: %s", id)
		}
		return nil, fmt.Errorf("读取模型配置文件失败: %w", err)
	}

	var dbModel ModelRecord
	if err := json.Unmarshal(data, &dbModel); err != nil {
		return nil, fmt.Errorf("解析模型配置失败: %w", err)
	}

	return dbModel.ToModelConfig()
}

// GetAllModelConfigs 获取所有模型配置
func (s *SQLiteDB) GetAllModelConfigs() (map[string]*config.ModelConfig, error) {
	modelsDir := filepath.Join(filepath.Dir(s.path), "models")

	// 检查目录是否存在
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		return make(map[string]*config.ModelConfig), nil
	}

	files, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, fmt.Errorf("读取模型配置目录失败: %w", err)
	}

	configs := make(map[string]*config.ModelConfig)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		modelID := strings.TrimSuffix(file.Name(), ".json")
		cfg, err := s.GetModelConfig(modelID)
		if err != nil {
			// 记录错误但继续处理其他文件
			fmt.Printf("警告: 加载模型配置 %s 失败: %v\n", modelID, err)
			continue
		}
		configs[cfg.ID] = cfg
	}

	return configs, nil
}

// DeleteModelConfig 删除模型配置
func (s *SQLiteDB) DeleteModelConfig(id string) error {
	modelsDir := filepath.Join(filepath.Dir(s.path), "models")
	modelFile := filepath.Join(modelsDir, id+".json")

	if err := os.Remove(modelFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("模型配置不存在: %s", id)
		}
		return fmt.Errorf("删除模型配置文件失败: %w", err)
	}

	return nil
}

// UpdateModelConfig 更新模型配置
func (s *SQLiteDB) UpdateModelConfig(cfg *config.ModelConfig) error {
	// 先检查模型是否存在
	_, err := s.GetModelConfig(cfg.ID)
	if err != nil {
		return err
	}

	// 更新配置（实际上就是重新保存）
	return s.SaveModelConfig(cfg)
}

// Close 关闭数据库连接
func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// ModelRecord 数据库记录结构
type ModelRecord struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Target          string    `json:"target"`
	Prompt          string    `json:"prompt"`
	Url             string    `json:"url"`
	Type            string    `json:"type"`
	PromptPath      string    `json:"prompt_path"`
	PromptValue     string    `json:"prompt_value"` // JSON字符串
	PromptValueType string    `json:"prompt_value_type"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ToModelConfig 转换为配置模型
func (m *ModelRecord) ToModelConfig() (*config.ModelConfig, error) {
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
		Url:             m.Url,
		Type:            config.ModelType(m.Type),
		PromptPath:      m.PromptPath,
		PromptValue:     promptValue,
		PromptValueType: config.ValueType(m.PromptValueType),
	}, nil
}
