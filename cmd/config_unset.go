package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"xdiag/internal/config"
)

func newConfigUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unset <key>",
		Short: "删除配置项",
		Long:  "删除单个配置项",
		Example: `
xdiag config unset api_key
xdiag config unset base_url
`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigUnset,
	}
}

func runConfigUnset(cmd *cobra.Command, args []string) error {
	key := args[0]

	if err := config.UnsetConfigValue(key); err != nil {
		return err
	}

	fmt.Printf("✅ %s removed from configuration\n", key)
	return nil
}
