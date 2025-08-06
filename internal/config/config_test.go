package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// 创建临时配置目录
	tempDir := t.TempDir()

	// 创建测试配置文件
	configContent := `models:
  - id: "test-model"
    name: "测试模型"
    target: "gpt-3.5-turbo"
    prompt: "测试Prompt"
    url: "https://api.openai.com/v1/chat/completions"
    type: "chat"
    prompt_path: "messages.0.content"
    prompt_value: "测试值"
    prompt_type: "string"`

	configFile := filepath.Join(tempDir, "test.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	// 加载配置
	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if len(config.Models) != 1 {
		t.Errorf("期望1个模型，实际得到%d个", len(config.Models))
	}

	model, exists := config.GetModel("test-model")
	if !exists {
		t.Error("未找到测试模型")
	}

	if model.Name != "测试模型" {
		t.Errorf("模型名称不匹配，期望'测试模型'，实际'%s'", model.Name)
	}

	if model.Type != ModelTypeChat {
		t.Errorf("模型类型不匹配，期望'%s'，实际'%s'", ModelTypeChat, model.Type)
	}
}

func TestValidateModelConfig(t *testing.T) {
	tests := []struct {
		name    string
		model   ModelConfig
		wantErr bool
	}{
		{
			name: "有效配置",
			model: ModelConfig{
				ID:              "test",
				Name:            "测试",
				Target:          "gpt-3.5-turbo",
				Prompt:          "测试",
				Url:             "https://api.openai.com/v1/chat/completions",
				Type:            ModelTypeChat,
				PromptPath:      "messages.0.content",
				PromptValue:     "测试",
				PromptValueType: ValueTypeString,
			},
			wantErr: false,
		},
		{
			name: "缺少ID",
			model: ModelConfig{
				Name:            "测试",
				Target:          "gpt-3.5-turbo",
				Prompt:          "测试",
				Url:             "https://api.openai.com/v1/chat/completions",
				Type:            ModelTypeChat,
				PromptPath:      "messages.0.content",
				PromptValue:     "测试",
				PromptValueType: ValueTypeString,
			},
			wantErr: true,
		},
		{
			name: "无效模型类型",
			model: ModelConfig{
				ID:              "test",
				Name:            "测试",
				Target:          "gpt-3.5-turbo",
				Prompt:          "测试",
				Url:             "https://api.openai.com/v1/chat/completions",
				Type:            "invalid",
				PromptPath:      "messages.0.content",
				PromptValue:     "测试",
				PromptValueType: ValueTypeString,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
