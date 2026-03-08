package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bigWhiteXie/xdiag/internal/config"
)

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "显示当前配置",
		Long:  "显示当前 LLM 配置信息",
		RunE:  runConfigShow,
	}
}

func runConfigShow(cmd *cobra.Command, args []string) error {
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
	fmt.Printf("  Protocol: %s\n", cfg.LLM.Protocol)
	fmt.Printf("  Model Name: %s\n", cfg.LLM.ModelName)
	if cfg.DataDir != "" {
		fmt.Printf("Data Dir: %s\n", cfg.DataDir)
	}
	if cfg.PlaybooksDir != "" {
		fmt.Printf("Playbooks Dir: %s\n", cfg.PlaybooksDir)
	}
	return nil
}

func maskString(s string) string {
	if len(s) < 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
