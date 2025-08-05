package proxy

import (
	"testing"

	"github.com/eolinker/ai-prompt-proxy/internal/config"
)

func TestInjectPromptMessages(t *testing.T) {
	body := "{\"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}"
	cfg := &config.ModelConfig{
		Name:        "test-model",
		Target:      "test-target",
		Url:         "http://127.0.0.1:8080",
		PromptPath:  "messages",
		PromptValue: map[string]interface{}{"role": "system", "content": "This is a test prompt."},
	}
	result, err := injectPrompt([]byte(body), cfg)
	if err != nil {
		t.Fatalf("injectPrompt failed: %v", err)
	}
	t.Log("Result:", string(result))
}

func TestInjectPromptMessageNil(t *testing.T) {
	body := "{\"stream\":true}"
	cfg := &config.ModelConfig{
		Name:        "test-model",
		Target:      "test-target",
		Url:         "http://127.0.0.1:8080",
		PromptPath:  "messages",
		PromptValue: map[string]interface{}{"role": "system", "content": "This is a test prompt."},
	}
	result, err := injectPrompt([]byte(body), cfg)
	if err != nil {
		t.Fatalf("injectPrompt failed: %v", err)
	}
	t.Log("Result:", string(result))
}
