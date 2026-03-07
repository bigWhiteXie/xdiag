package cmd

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"xdiag/internal/app/diagnose"
	"xdiag/internal/app/targets"
	"xdiag/internal/config"
	"xdiag/internal/llm"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diag",
	Short: "执行智能诊断",
	Long:  `根据用户描述执行智能诊断，包括目标匹配、方案检索和执行等步骤。`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userDescription := args[0]
		ctx := context.Background()
		conf, err := config.LoadConfig()
		if err != nil {
			return err
		}
		// 初始化LLM客户端
		client, err := llm.NewClient(ctx, &conf.LLM)

		if err != nil {
			log.Fatalf("Failed to create LLM client: %v", err)
		}

		// 获取数据库路径
		dataDir := viper.GetString("data_dir")
		if dataDir == "" {
			log.Fatal("无法获取data目录")
		}
		dbPath := filepath.Join(dataDir, "xdiag.db")

		// 初始化目标仓库
		targetRepo, err := targets.NewSQLiteRepo(dbPath)
		if err != nil {
			log.Fatalf("Failed to create target repo: %v", err)
		}
		defer targetRepo.Close()

		// 创建诊断服务
		diagService := diagnose.NewService(client.Model, targetRepo)

		// 执行Route Target阶段
		output, err := diagService.RouteTarget(ctx, userDescription)
		if err != nil {
			log.Fatalf("Failed to route target: %v", err)
		}

		// 显示匹配到的目标
		fmt.Printf("根据您的描述 '%s'，匹配到以下目标:\n", userDescription)
		if len(output.Targets) == 0 {
			fmt.Println("未找到匹配的目标")
		} else {
			for _, target := range output.Targets {
				fmt.Printf("- %s (%s): %s:%d\n", target.Name, target.Kind, target.Address, target.Port)
			}
		}

		// 接收LLM消息
		fmt.Println("\nLLM分析过程:")
		for msg := range output.MessageChan {
			fmt.Printf("LLM: %s\n", msg.Content)
		}
	},
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)
}
