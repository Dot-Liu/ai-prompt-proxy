# AI Prompt Proxy

一个用Golang开发的AI Prompt代理服务，支持在转发AI请求时自动注入自定义Prompt。

## 功能特性

- 📁 **配置文件管理**: 支持从指定文件夹读取YAML配置文件
- 🔄 **请求代理**: 代理转发客户端请求到上游AI服务
- 💉 **Prompt注入**: 根据模型ID自动注入对应的Prompt
- 🎯 **灵活配置**: 支持多种Prompt值类型（string、object、array）
- 📍 **JSON Path**: 支持使用JSON Path指定Prompt插入位置
- 🚀 **多模型支持**: 支持chat、image、audio、video等模型类型

## 配置说明

### 模型配置文件

配置文件使用YAML格式，支持以下字段：

```yaml
models:
  - id: "模型ID"                    # 必须：客户端请求中的模型ID
    name: "模型名称"                 # 必须：模型显示名称
    target_model_id: "目标模型ID"    # 必须：转发到上游服务的实际模型ID
    model_prompt: "模型描述"         # 必须：模型的Prompt描述
    model_type: "chat"              # 必须：模型类型 (chat/image/audio/video)
    prompt_insert_path: "messages.0.content"  # 必须：Prompt插入的JSON路径
    prompt_value:                   # 必须：要注入的Prompt值
      type: "string"                # 值类型：string/object/array
      value: "实际的Prompt内容"
```

### JSON Path 示例

- `messages.0.content` - 插入到messages数组第一个元素的content字段
- `messages.-1` - 在messages数组末尾添加新元素
- `prompt` - 直接设置prompt字段
- `system.instructions` - 设置嵌套对象的字段

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 准备配置文件

在 `configs/` 目录下创建配置文件，参考 `configs/example.yaml`。

### 3. 运行服务

```bash
# 使用默认配置
go run . 

# 指定配置目录和端口
go run . -config=./configs -port=8080
```

### 4. 测试请求

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "gpt-3.5-turbo-custom",
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'
```

## 环境变量

- `UPSTREAM_URL`: 上游AI服务的基础URL（默认：https://api.openai.com）

## Docker 部署

```bash
# 构建镜像
docker build -t ai-prompt-proxy .

# 运行容器
docker run -p 8080:8080 -v $(pwd)/configs:/root/configs ai-prompt-proxy
```

## 开发

```bash
# 安装开发依赖
make deps

# 运行开发服务器
make dev

# 运行测试
make test

# 代码格式化
make fmt
```

## 项目结构