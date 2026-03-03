package llm

import (
	"context"
	"fmt"

	eino_openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// ToolCallingChatModel 是 Eino 框架中 ToolCallingChatModel 的别名
type ToolCallingChatModel = model.ToolCallingChatModel

// ModelProvider 定义了模型提供商类型
type ModelProvider string

const (
	OpenAIProvider     ModelProvider = "openai"
	AnthropicProvider  ModelProvider = "anthropic"
	CustomProvider     ModelProvider = "custom"
	SiliconFlowProvider ModelProvider = "siliconflow"
)

// ClientFactory 是大模型客户端工厂
type ClientFactory struct{}

// ClientConfig 包含创建客户端所需的配置
type ClientConfig struct {
	Provider  string
	APIKey    string
	BaseURL   string
	ModelName string
}

// NewClient 根据配置创建相应的大模型客户端
func (f *ClientFactory) NewClient(ctx context.Context, config *ClientConfig) (ToolCallingChatModel, error) {
	switch ModelProvider(config.Provider) {
	case OpenAIProvider, SiliconFlowProvider, CustomProvider:
		return f.createOpenAIClient(ctx, config)
	case AnthropicProvider:
		return f.createAnthropicClient(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// createOpenAIClient 创建 OpenAI 客户端或其他兼容 OpenAI API 的服务
func (f *ClientFactory) createOpenAIClient(ctx context.Context, config *ClientConfig) (ToolCallingChatModel, error) {
	// 使用 Eino 框架创建 OpenAI 客户端
	client, err := eino_openai.NewChatModel(ctx, &eino_openai.ChatModelConfig{
		BaseURL: config.BaseURL,
		Model:   config.ModelName,
		APIKey:  config.APIKey,
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

// createAnthropicClient 创建 Anthropic 客户端
func (f *ClientFactory) createAnthropicClient(ctx context.Context, config *ClientConfig) (ToolCallingChatModel, error) {
	// 注意：这里我们暂时返回错误，因为 Anthropic 客户端需要额外的组件
	// 在实际应用中，您需要添加 Anthropic 相关的 Eino 组件
	return nil, fmt.Errorf("Anthropic provider not yet implemented")
}