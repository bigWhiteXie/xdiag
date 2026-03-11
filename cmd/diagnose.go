package cmd

import (
	"context"
	"fmt"
	"path"

	"github.com/spf13/cobra"

	"github.com/bigWhiteXie/xdiag/internal/app/diagnose/execute"
	"github.com/bigWhiteXie/xdiag/internal/app/diagnose/match"
	"github.com/bigWhiteXie/xdiag/internal/app/diagnose/route"
	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/bigWhiteXie/xdiag/internal/app/targets"
	"github.com/bigWhiteXie/xdiag/internal/config"
	"github.com/bigWhiteXie/xdiag/internal/llm"
	"github.com/bigWhiteXie/xdiag/internal/svc"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diag",
	Short: "执行智能诊断",
	Long:  "根据用户描述执行智能诊断，包括目标匹配、方案检索和执行等步骤",
	RunE:  runDiagnose,
}

func init() {
	diagnoseCmd.Flags().String("question", "", "目标名称 (必需)")
	diagnoseCmd.Flags().Bool("show", false, "是否显示诊断过程细节")
	rootCmd.AddCommand(diagnoseCmd)
}

func runDiagnose(cmd *cobra.Command, args []string) error {
	userDescription, _ := cmd.Flags().GetString("question")
	showDetails, _ := cmd.Flags().GetBool("show")
	ctx := context.Background()

	config, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	// 构建repo和model对象
	targetRepo, err := targets.NewSQLiteRepo(path.Join(config.DataDir, "targets.db"))
	if err != nil {
		return fmt.Errorf("创建sqlite repo失败: %w", err)
	}
	svc.SetTargetsRepo(targetRepo)
	llm, err := llm.NewClient(ctx, &config.LLM)
	if err != nil {
		return fmt.Errorf("创建llm client失败: %w", err)
	}
	svc.SetModel(llm)
	svc.SetBookRepo(playbook.NewRepo(config.PlaybooksDir))

	agent, err := route.NewTargetRouteAgent(ctx, showDetails)
	if err != nil {
		return fmt.Errorf("创建路由代理失败: %w", err)
	}

	fmt.Printf("正在分析您的问题: %s......\n", userDescription)
	// ========================定位目标========================
	targetID, err := agent.Run(ctx, userDescription)
	if err != nil {
		return fmt.Errorf("路由目标失败: %w", err)
	}

	if targetID == 0 {
		fmt.Println("未找到匹配的目标")
		return nil
	}

	Diagtarget, err := targetRepo.GetByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("获取目标具体信息失败: %w", err)
	}
	fmt.Printf("✅ 找到目标, ip:%s port:%d kind:%s tags:%s\n", Diagtarget.Address, Diagtarget.Port, Diagtarget.Kind, Diagtarget.Tags)
	_, err = targets.TestConnectivity(ctx, Diagtarget)
	if err != nil {
		return fmt.Errorf("测试目标连通性失败: %w", err)
	}
	// ========================匹配方案========================
	matcher, err := match.NewMatcher(svc.GetServiceContext().BookRepo, svc.GetServiceContext().Model, showDetails)
	if err != nil {
		return fmt.Errorf("创建匹配器失败: %w", err)
	}
	matchResult, err := matcher.Match(ctx, Diagtarget, userDescription)
	if err != nil {
		return fmt.Errorf("匹配失败: %w", err)
	}
	if !matchResult.Success {
		fmt.Printf("未匹配到playbook, 具体信息: %s\n", matchResult.Message)
	}

	fmt.Printf("✅ 匹配成功 方案:%s 描述:%s\n", matchResult.Ref.Name, matchResult.Ref.Desc)

	// ========================执行方案========================
	executor, err := execute.NewExecutor(ctx)
	if err != nil {
		return fmt.Errorf("创建方案执行器失败:%s", err)
	}
	book, err := svc.GetServiceContext().BookRepo.GetBook(matchResult.Playbook.Name, matchResult.Ref.Name)
	if err != nil {
		return fmt.Errorf("获取诊断方案失败:%s", err)
	}
	evtChan, err := executor.Execute(ctx, book, Diagtarget, userDescription, showDetails)
	if err != nil {
		return fmt.Errorf("执行诊断失败: %w", err)
	}
	report := executor.GetReport(evtChan, showDetails)
	fmt.Printf("诊断报告:\n%s", report)
	return nil
}
