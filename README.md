# xdiag 智能诊断 CLI 工具

xdiag 是一个基于 Go 语言和 Eino 框架开发的非交互式、跨节点智能诊断 CLI 工具。

## 功能特性

- **配置管理**：支持多种 LLM 服务提供商（OpenAI、Anthropic、SiliconFlow 等）
- **智能诊断**：基于 AI Agent 的自动化诊断流程
- **目标管理**：统一管理待诊断的目标节点（主机、数据库、中间件等）
- **剧本驱动**：可扩展的诊断方案（Playbook）

## 安装

```bash
go install xdiag
```

## 使用方法

### 配置 LLM 模型

```bash
# 配置 OpenAI 模型
xdiag config model --api-key sk-xxxxxx --model-name gpt-4o

# 查看当前配置
xdiag config show

# 修改配置项
xdiag config set model_name gpt-4-turbo

# 测试 LLM 连接
xdiag config test
```

## 集成 Eino 框架说明

当前版本使用模拟接口实现，当 Eino 框架可用时，请按以下步骤替换实现：

1. 在 `go.mod` 中添加 Eino 框架依赖
2. 在 `internal/llm/client_factory.go` 中，将创建客户端的代码替换为：

```go
import (
    "context"
    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/components/model"
)

// createOpenAIClient 创建 OpenAI 客户端
func (f *ClientFactory) createOpenAIClient(ctx context.Context, config *ClientConfig) (ToolCallingChatModel, error) {
    return openai.NewChatModel(ctx, &openai.ChatModelConfig{
        BaseURL: config.BaseURL,
        Model:   config.ModelName,
        APIKey:  config.APIKey,
    })
}
```

类似地，为其他提供商实现相应的客户端创建逻辑。

## 开发

```bash
# 运行项目
go run main.go

# 运行测试
go test ./...
```

## 许可证

MIT