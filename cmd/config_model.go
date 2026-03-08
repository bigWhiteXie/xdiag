package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"xdiag/internal/config"
)

func newConfigModelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "配置 LLM 模型参数",
		Long:  "设置 LLM 模型的 API Key、Base URL、协议类型和模型名称",
		Example: `
# 配置 OpenAI 模型
xdiag config model --api-key sk-xxx --model-name gpt-4o

# 配置自定义服务
xdiag config model \
  --api-key xxx \
  --base-url https://custom.ai.com/v1 \
  --protocol openai \
  --model-name custom-model
`,
		RunE: runConfigModel,
	}

	cmd.Flags().String("api-key", "", "LLM API Key (必填)")
	cmd.Flags().String("base-url", "", "LLM Base URL (可选，默认：https://api.openai.com/v1)")
	cmd.Flags().String("protocol", "", "协议类型：openai/anthropic/custom (可选，默认：openai)")
	cmd.Flags().String("model-name", "", "模型名称 (必填)，如 gpt-4o, claude-3-opus")

	cmd.MarkFlagRequired("api-key")
	cmd.MarkFlagRequired("model-name")

	return cmd
}

func runConfigModel(cmd *cobra.Command, args []string) error {
	apiKey, _ := cmd.Flags().GetString("api-key")
	baseURL, _ := cmd.Flags().GetString("base-url")
	protocol, _ := cmd.Flags().GetString("protocol")
	modelName, _ := cmd.Flags().GetString("model-name")

	if apiKey == "" {
		return fmt.Errorf("--api-key 是必填参数")
	}
	if modelName == "" {
		return fmt.Errorf("--model-name 是必填参数")
	}

	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if protocol == "" {
		protocol = "openai"
	}

	if err := config.SaveModelConfig(apiKey, baseURL, protocol, modelName); err != nil {
		return err
	}

	fmt.Println("✅ 配置已保存到 ~/.xdiag/config.yaml")
	return nil
}
