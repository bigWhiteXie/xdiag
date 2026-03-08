package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "xdiag",
	Short: "xdiag 是一个智能诊断 CLI 工具",
	Long:  "xdiag 是一个基于 AI 的智能诊断工具，帮助用户诊断系统问题",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	configDir := viper.GetString("config_dir")
	if configDir == "" {
		configDir = "$HOME/.xdiag"
	}

	configDir = os.ExpandEnv(configDir)

	viper.AddConfigPath(configDir)
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不是必须的，忽略错误
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		// 仅记录错误，不中断程序
	}

	playbooksDir := filepath.Join(configDir, "playbooks")
	if err := os.MkdirAll(playbooksDir, 0755); err != nil {
		// 仅记录错误，不中断程序
	}

	dataDir := filepath.Join(configDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		// 仅记录错误，不中断程序
	}

	viper.Set("playbooks_dir", playbooksDir)
	viper.Set("data_dir", dataDir)
}