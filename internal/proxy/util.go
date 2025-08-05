package proxy

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/eolinker/ai-prompt-proxy/internal/config"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func injectPrompt(body []byte, cfg *config.ModelConfig) ([]byte, error) {
	bodyStr := string(body)
	val := cfg.PromptValue
	valType := cfg.PromptValueType
	if val == nil {
		switch cfg.Type {
		case config.ModelTypeChat:
			val = map[string]interface{}{
				"role":    "system",
				"content": cfg.Prompt,
			}
			valType = config.ValueTypeArray
		case config.ModelTypeImage, config.ModelTypeAudio:
			val = cfg.Prompt
			valType = config.ValueTypeString
		default:
			return nil, fmt.Errorf("unsupported model type: %s", cfg.Type)
		}
	}
	typ := reflect.TypeOf(val)
	switch typ.Kind() {
	case reflect.Map:
		result := gjson.Get(bodyStr, cfg.PromptPath)
		switch {
		case result.IsArray():
			// 将cfg.PromptValue添加到数组首位
			arr := result.Array()
			vs := make([]interface{}, 0, len(arr)+1)
			vs = append(vs, val)
			for _, v := range arr {
				vs = append(vs, v.Value())
			}
			data, err := sjson.Set(bodyStr, cfg.PromptPath, vs)
			if err != nil {
				return nil, err
			}
			return []byte(data), nil
		case result.Type == gjson.Null:
			switch valType {
			case "", config.ValueTypeArray:
				vs := make([]interface{}, 0, +1)
				vs = append(vs, val)
				// 如果路径不存在，则直接设置为数组
				data, err := sjson.Set(bodyStr, cfg.PromptPath, vs)
				if err != nil {
					return nil, err
				}
				return []byte(data), nil
			case config.ValueTypeObject:
				// 如果路径不存在，则直接设置为对象
				data, err := sjson.Set(bodyStr, cfg.PromptPath, val)
				if err != nil {
					return nil, err
				}
				return []byte(data), nil
			case config.ValueTypeString:
				// 如果路径不存在，则直接设置为字符串
				v, err := json.Marshal(val)
				data, err := sjson.Set(bodyStr, cfg.PromptPath, v)
				if err != nil {
					return nil, err
				}
				return []byte(data), nil
			default:
				return nil, fmt.Errorf("unsupported prompt value type: %s", valType)
			}
		default:
			return nil, fmt.Errorf("prompt path %s is not an array", cfg.PromptPath)
		}
	case reflect.String:
		result := gjson.Get(bodyStr, cfg.PromptPath)
		switch result.Type {
		case gjson.String:
			// 将cfg.PromptValue添加到字符串前
			data, err := sjson.Set(bodyStr, cfg.PromptPath, fmt.Sprintf("%s\n%s", val.(string), result.String()))
			if err != nil {
				return nil, err
			}
			return []byte(data), nil
		case gjson.Null:
			// 如果路径不存在，则直接设置
			data, err := sjson.Set(bodyStr, cfg.PromptPath, val.(string))
			if err != nil {
				return nil, err
			}
			return []byte(data), nil
		default:
			return nil, fmt.Errorf("prompt path %s is not a string", cfg.PromptPath)
		}
	default:
		return nil, fmt.Errorf("unsupported prompt value type: %s", typ.Kind())
	}
}

func replaceModelID(body []byte, target string) ([]byte, error) {
	bodyStr := string(body)
	result, err := sjson.Set(bodyStr, "model", target)
	if err != nil {
		return nil, err
	}
	return []byte(result), nil
}

func extractModelID(body []byte) string {
	// 尝试从JSON中提取model字段
	result := gjson.GetBytes(body, "model")
	if result.Exists() {
		return result.String()
	}
	return ""
}

func getUpstreamURL(prefix string, path string) string {
	// 这里可以根据需要配置不同的上游服务
	// 示例：OpenAI API
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(prefix, "/"), strings.TrimPrefix(path, "/"))

}
