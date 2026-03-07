package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"xdiag/internal/config"
	"xdiag/internal/llm"

	"github.com/cloudwego/eino/schema"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理 xdiag 配置",
	Long:  "用于查看、设置、删除 xdiag 的各项配置",
}

func init() {
	configCmd.AddCommand(configModelCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configUnsetCmd)
	configCmd.AddCommand(configTestCmd)
}

// config model 命令
var configModelCmd = &cobra.Command{
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
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, _ := cmd.Flags().GetString("api-key")
		baseURL, _ := cmd.Flags().GetString("base-url")
		protocol, _ := cmd.Flags().GetString("protocol")
		modelName, _ := cmd.Flags().GetString("model-name")

		// 验证必填参数
		if apiKey == "" {
			return fmt.Errorf("--api-key 是必填参数")
		}
		if modelName == "" {
			return fmt.Errorf("--model-name 是必填参数")
		}

		// 默认值
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		if protocol == "" {
			protocol = "openai"
		}

		// 保存配置
		if err := config.SaveModelConfig(apiKey, baseURL, protocol, modelName); err != nil {
			return err
		}

		fmt.Println("✅ 配置已保存到 ~/.xdiag/config.yaml")
		return nil
	},
}

func init() {
	configModelCmd.Flags().String("api-key", "", "LLM API Key (必填)")
	configModelCmd.Flags().String("base-url", "", "LLM Base URL (可选，默认：https://api.openai.com/v1)")
	configModelCmd.Flags().String("protocol", "", "协议类型：openai/anthropic/custom (可选，默认：openai)")
	configModelCmd.Flags().String("model-name", "", "模型名称 (必填)，如 gpt-4o, claude-3-opus")

	configModelCmd.MarkFlagRequired("api-key")
	configModelCmd.MarkFlagRequired("model-name")
}

// config show 命令
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前配置",
	Long:  "显示当前 LLM 配置信息",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("加载配置失败：%v", err)
		}

		fmt.Println("LLM Configuration:")
		if cfg.LLM.APIKey != "" {
			masked := maskString(cfg.LLM.APIKey)
			fmt.Printf("  API Key: %s\n", masked)
		} else {
			fmt.Println("  API Key: (未设置)")
		}
		fmt.Printf("  Base URL: %s\n", cfg.LLM.BaseURL)
		fmt.Printf("  Provider: %s\n", cfg.LLM.Provider)
		fmt.Printf("  Model Name: %s\n", cfg.LLM.ModelName)

		return nil
	},
}

// config set 命令
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "设置配置项",
	Long:  "设置单个配置项的值",
	Example: `
xdiag config set model_name gpt-4-turbo
xdiag config set base_url https://api.anthropic.com
`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		// 设置配置值
		if err := config.SetConfigValue(key, value); err != nil {
			return err
		}

		fmt.Printf("✅ %s updated to %s\n", key, value)
		return nil
	},
}

// config unset 命令
var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "删除配置项",
	Long:  "删除单个配置项",
	Example: `
xdiag config unset api_key
xdiag config unset base_url
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		// 删除配置值
		if err := config.UnsetConfigValue(key); err != nil {
			return err
		}

		fmt.Printf("✅ %s removed from configuration\n", key)
		return nil
	},
}

// config test 命令
var configTestCmd = &cobra.Command{
	Use:   "test",
	Short: "测试 LLM 配置",
	Long:  "验证 LLM 配置是否有效，测试 API Key 和模型可用性",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("加载配置失败：%v", err)
		}

		if cfg.LLM.APIKey == "" {
			return fmt.Errorf("API Key 未设置")
		}

		fmt.Println("🔍 正在测试 LLM 连接...")

		// 创建客户端
		ctx := context.Background()
		client, err := llm.NewClient(ctx, &cfg.LLM)
		if err != nil {
			return fmt.Errorf("创建 LLM 客户端失败：%v", err)
		}

		// 测试连接 - 发送一个简单的测试请求
		// 根据 Eino 框架接口，我们需要创建 Message 对象
		messages := []*schema.Message{
			{
				Role:    schema.User,
				Content: "Hello, are you there?",
			},
		}

		// 使用 Eino 框架的 Generate 方法
		response, err := client.Generate(ctx, messages)
		if err != nil {
			// 如果连接失败，可能是配置错误，但我们也接受这种情况作为"测试连接失败"
			return fmt.Errorf("测试 LLM 连接失败：%v", err)
		}

		fmt.Printf("✅ 连接成功！API Key 有效，模型可用：%s\n", cfg.LLM.ModelName)
		fmt.Printf("   测试响应：%s\n", response.Content)

		return nil
	},
}

// 辅助函数：掩码敏感信息
func maskString(s string) string {
	if len(s) < 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
