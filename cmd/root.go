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
	Long:  `xdiag 是一个基于 AI 的智能诊断工具，帮助用户诊断系统问题。`,
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
	// 尝试查找并读取配置文件
	configDir := viper.GetString("config_dir")
	if configDir == "" {
		configDir = "$HOME/.xdiag"
	}
	
	// 展开$HOME环境变量
	configDir = os.ExpandEnv(configDir)
	
	viper.AddConfigPath(configDir)
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")

	// 尝试读取配置文件，如果不存在则跳过
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不是必须的，所以忽略错误
	}
	
	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		// 仅打印到stderr，不中断程序
	}
	
	// 确保playbooks目录存在
	playbooksDir := filepath.Join(configDir, "playbooks")
	if err := os.MkdirAll(playbooksDir, 0755); err != nil {
		// 仅打印到stderr，不中断程序
	}
	
	// 确保data目录存在
	dataDir := filepath.Join(configDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		// 仅打印到stderr，不中断程序
	}
	
	// 设置playbooks目录到viper
	viper.Set("playbooks_dir", playbooksDir)
	viper.Set("data_dir", dataDir)
}