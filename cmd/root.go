package cmd

import (
	"os"

	"github.com/bigWhiteXie/xdiag/internal/config"
	"github.com/bigWhiteXie/xdiag/internal/svc"

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
	configDir := config.GetConfigDir()

	viper.AddConfigPath(configDir)
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不是必须的，忽略错误
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		// 仅记录错误，不中断程序
	}

	// 初始化日志系统
	logLevel := viper.GetString("log.level")
	if logLevel == "" {
		logLevel = "info"
	}
	development := viper.GetBool("log.development")

	if err := svc.InitLogger(logLevel, development); err != nil {
		// 日志初始化失败，使用默认配置
	}
}
