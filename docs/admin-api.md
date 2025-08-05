# 管理API文档

管理API服务器提供模型配置的CRUD操作接口。

## 基础信息

- **基础URL**: `http://localhost:8081/api/v1`
- **认证**: 暂无（可根据需要添加）
- **响应格式**: JSON

## 通用响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

## API接口

### 1. 获取模型列表

**GET** `/models`

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "models": [
      {
        "id": "gpt-3.5-turbo-custom",
        "name": "GPT-3.5 Turbo 自定义",
        "target_model_id": "gpt-3.5-turbo",
        "model_prompt": "你是一个专业的AI助手",
        "model_type": "chat",
        "prompt_insert_path": "messages.0.content",
        "prompt_value": {
          "type": "string",
          "value": "你是一个专业的AI助手"
        }
      }
    ],
    "total": 1
  }
}
```

### 2. 根据模型ID获取模型信息

**GET** `/models/{id}`

**路径参数**:
- `id`: 模型ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "gpt-3.5-turbo-custom",
    "name": "GPT-3.5 Turbo 自定义",
    "target_model_id": "gpt-3.5-turbo",
    "model_prompt": "你是一个专业的AI助手",
    "model_type": "chat",
    "prompt_insert_path": "messages.0.content",
    "prompt_value": {
      "type": "string",
      "value": "你是一个专业的AI助手"
    }
  }
}
```

### 3. 创建模型配置

**POST** `/models`

**请求体**:
```json
{
  "id": "new-model-id",
  "name": "新模型名称",
  "target_model_id": "gpt-4",
  "model_prompt": "模型描述",
  "model_type": "chat",
  "prompt_insert_path": "messages.-1",
  "prompt_value": {
    "type": "object",
    "value": {
      "role": "system",
      "content": "你是一个专业助手"
    }
  }
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "模型创建成功",
  "data": {
    "id": "new-model-id",
    "name": "新模型名称",
    // ... 其他字段
  }
}
```

### 4. 更新模型配置

**PUT** `/models/{id}`

**路径参数**:
- `id`: 模型ID

**请求体** (所有字段都是可选的):
```json
{
  "name": "更新的模型名称",
  "target_model_id": "gpt-4-turbo",
  "model_prompt": "更新的描述",
  "model_type": "chat",
  "prompt_insert_path": "messages.0.content",
  "prompt_value": {
    "type": "string",
    "value": "更新的Prompt"
  }
}
```

### 5. 删除模型配置

**DELETE** `/models/{id}`

**路径参数**:
- `id`: 模型ID

**响应示例**:
```json
{
  "code": 0,
  "message": "模型删除成功"
}
```

### 6. 重新加载配置

**POST** `/config/reload`

**响应示例**:
```json
{
  "code": 0,
  "message": "配置重新加载成功",
  "data": {
    "total_models": 5
  }
}
```

### 7. 获取服务状态

**GET** `/config/status`

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "running",
    "total_models": 5,
    "config_dir": "./configs"
  }
}
```

### 8. 健康检查

**GET** `/health`

**响应示例**:
```json
{
  "status": "ok",
  "timestamp": {}
}
```

## 错误码说明

- `0`: 成功
- `400`: 请求参数错误
- `404`: 资源不存在
- `409`: 资源冲突（如模型ID已存在）
- `500`: 服务器内部错误

## 使用示例

### 创建一个新的聊天模型配置

```bash
curl -X POST http://localhost:8081/api/v1/models \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my-custom-gpt",
    "name": "我的自定义GPT",
    "target_model_id": "gpt-3.5-turbo",
    "model_prompt": "专业编程助手",
    "model_type": "chat",
    "prompt_insert_path": "messages.-1",
    "prompt_value": {
      "type": "object",
      "value": {
        "role": "system",
        "content": "你是一个专业的编程助手，擅长解决各种编程问题。"
      }
    }
  }'
```

### 获取所有模型列表

```bash
curl http://localhost:8081/api/v1/models
```

### 更新模型配置

```bash
curl -X PUT http://localhost:8081/api/v1/models/my-custom-gpt \
  -H "Content-Type: application/json" \
  -d '{
    "name": "更新后的模型名称",
    "model_prompt": "更新后的描述"
  }'
```