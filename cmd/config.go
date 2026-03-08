package cmd

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理 xdiag 配置",
	Long:  "用于查看、设置、删除 xdiag 的各项配置",
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(newConfigModelCmd())
	configCmd.AddCommand(newConfigShowCmd())
	configCmd.AddCommand(newConfigSetCmd())
	configCmd.AddCommand(newConfigUnsetCmd())
	configCmd.AddCommand(newConfigTestLLMCmd())
}
