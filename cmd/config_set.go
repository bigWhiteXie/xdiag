package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bigWhiteXie/xdiag/internal/config"
)

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "设置配置项",
		Long:  "设置单个配置项的值",
		Example: `
xdiag config set model_name gpt-4-turbo
xdiag config set base_url https://api.anthropic.com
xdiag config set api_key xxxxx
xdiag config set provider openai
xdiag config set data_dir /root/.github.com/bigWhiteXie/xdiag/data
xdiag config set book_path /root/.github.com/bigWhiteXie/xdiag/playbooks


`,
		Args: cobra.ExactArgs(2),
		RunE: runConfigSet,
	}
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	if err := config.SetConfigValue(key, value); err != nil {
		return err
	}

	fmt.Printf("✅ %s updated to %s\n", key, value)
	return nil
}
