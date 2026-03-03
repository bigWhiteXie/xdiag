package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "xdiag",
	Short: "xdiag 是一个智能诊断 CLI 工具",
	Long:  `xdiag 是一个基于 AI 的智能诊断工具，帮助用户诊断系统问题。`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// 尝试查找并读取配置文件
	configDir := viper.GetString("config_dir")
	if configDir == "" {
		configDir = "$HOME/.xdiag"
	}
	viper.AddConfigPath(configDir)
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")

	// 尝试读取配置文件，如果不存在则跳过
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不是必须的，所以忽略错误
	}
}