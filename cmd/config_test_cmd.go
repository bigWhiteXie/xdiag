package cmd

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/spf13/cobra"

	"github.com/bigWhiteXie/xdiag/internal/config"
	"github.com/bigWhiteXie/xdiag/internal/llm"
)

func newConfigTestLLMCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "测试 LLM 配置",
		Long:  "验证 LLM 配置是否有效，测试 API Key 和模型可用性",
		RunE:  runConfigTestLLM,
	}
}

func runConfigTestLLM(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败：%v", err)
	}

	if cfg.LLM.APIKey == "" {
		return fmt.Errorf("API Key 未设置")
	}

	fmt.Println("🔍 正在测试 LLM 连接...")

	ctx := context.Background()
	client, err := llm.NewClient(ctx, &cfg.LLM)
	if err != nil {
		return fmt.Errorf("创建 LLM 客户端失败：%v", err)
	}

	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: "Hello, are you there?",
		},
	}

	response, err := client.Generate(ctx, messages)
	if err != nil {
		return fmt.Errorf("测试 LLM 连接失败：%v", err)
	}

	fmt.Printf("✅ 连接成功！API Key 有效，模型可用：%s\n", cfg.LLM.ModelName)
	fmt.Printf("   测试响应：%s\n", response.Content)

	return nil
}
