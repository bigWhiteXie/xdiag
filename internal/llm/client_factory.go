package llm

import (
	"context"
	"fmt"
	"net/http"

	eino_openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// ModelProvider 定义了模型提供商类型
type ModelProvider string

const (
	OpenAIProvider      ModelProvider = "openai"
	AnthropicProvider   ModelProvider = "anthropic"
	CustomProvider      ModelProvider = "custom"
	SiliconFlowProvider ModelProvider = "siliconflow"
)

// ClientFactory 是大模型客户端工厂
type ClientFactory struct{}

// ClientConfig 包含创建客户端所需的配置
type ClientConfig struct {
	APIKey     string `mapstructure:"api_key"`
	BaseURL    string `mapstructure:"base_url"`
	ModelName  string `mapstructure:"model_name"`
	Protocol   string `mapstructure:"protocol"`
	MaxRetries int    `mapstructure:"max_retries"`
}

// NewClient 根据配置创建相应的大模型客户端
func NewClient(ctx context.Context, config *ClientConfig) (model.ToolCallingChatModel, error) {
	switch ModelProvider(config.Protocol) {
	case OpenAIProvider, SiliconFlowProvider, CustomProvider:
		return createOpenAIClient(ctx, config)
	case AnthropicProvider:
		return createAnthropicClient(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Protocol)
	}
}

// createOpenAIClient 创建 OpenAI 客户端或其他兼容 OpenAI API 的服务
func createOpenAIClient(ctx context.Context, config *ClientConfig) (model.ToolCallingChatModel, error) {
	// 创建自定义 HTTP 客户端，将 API key 添加到 Authorization 头
	httpClient := &http.Client{
		Transport: &authTransport{
			apiKey: config.APIKey,
			base:   http.DefaultTransport,
		},
	}

	// 使用 Eino 框架创建 OpenAI 客户端
	client, err := eino_openai.NewChatModel(ctx, &eino_openai.ChatModelConfig{
		BaseURL:    config.BaseURL,
		Model:      config.ModelName,
		APIKey:     config.APIKey,
		HTTPClient: httpClient,
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

// authTransport 是一个自定义的 HTTP Transport，用于在请求头中添加 Authorization
type authTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 克隆请求以避免修改原始请求
	req = req.Clone(req.Context())
	// 将 API key 添加到 Authorization 头
	req.Header.Set("Authorization", t.apiKey)
	return t.base.RoundTrip(req)
}

// createAnthropicClient 创建 Anthropic 客户端
func createAnthropicClient(ctx context.Context, config *ClientConfig) (model.ToolCallingChatModel, error) {
	// 注意：这里我们暂时返回错误，因为 Anthropic 客户端需要额外的组件
	// 在实际应用中，您需要添加 Anthropic 相关的 Eino 组件
	return nil, fmt.Errorf("Anthropic provider not yet implemented")
}
