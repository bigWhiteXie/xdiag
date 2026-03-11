package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/bigWhiteXie/xdiag/internal/config"
	"github.com/bigWhiteXie/xdiag/internal/llm"
)

var playbookCmd = &cobra.Command{
	Use:   "playbook",
	Short: "管理诊断剧本(Playbook)",
	Long:  "管理诊断剧本，包括列表、显示、更新和执行等操作",
}

func init() {
	rootCmd.AddCommand(playbookCmd)
	playbookCmd.AddCommand(newListPlaybookCmd())
	playbookCmd.AddCommand(newShowPlaybookCmd())
	playbookCmd.AddCommand(newGenerateBookCmd())
}

func newListPlaybookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出所有可用的Playbook",
		Long:  "列出所有在系统中注册的Playbook及其基本信息",
		RunE:  runListPlaybook,
	}
}

func runListPlaybook(cmd *cobra.Command, args []string) error {
	conf, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	repo := playbook.NewRepo(conf.PlaybooksDir)
	playbooks, err := repo.ListPlaybooks(nil)
	if err != nil {
		return fmt.Errorf("加载Playbook列表时出错: %w", err)
	}

	fmt.Println("Available Playbooks:")
	fmt.Println("NAME\t\t\tDESC\t\t\tREQUIRED_TAGS")
	for _, pb := range playbooks {
		tags := fmt.Sprintf("%v", pb.Tags)
		if len(pb.Tags) == 0 {
			tags = "none"
		}
		dirName := filepath.Base(pb.Path)
		fmt.Printf("%-20s\t%-30s\t%s\n", dirName, pb.Desc, tags)
	}

	return nil
}

func newShowPlaybookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [playbook-name]",
		Short: "显示指定Playbook的详细信息",
		Long:  "显示指定名称的Playbook的详细信息，包括其所有诊断方案",
		Args:  cobra.ExactArgs(1),
		RunE:  runShowPlaybook,
	}
}

func runShowPlaybook(cmd *cobra.Command, args []string) error {
	name := args[0]

	conf, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	repo := playbook.NewRepo(conf.PlaybooksDir)
	playbooks, err := repo.ListPlaybooks(nil)
	if err != nil {
		return fmt.Errorf("加载Playbook '%s' 时出错: %w", name, err)
	}

	var pb *playbook.Playbook
	for _, p := range playbooks {
		if filepath.Base(p.Path) == name {
			pb = &p
			break
		}
	}

	if pb == nil {
		return fmt.Errorf("找不到Playbook '%s'", name)
	}

	fmt.Printf("Playbook: %s\n", pb.Name)
	fmt.Printf("Description: %s\n", pb.Desc)

	tags := fmt.Sprintf("%v", pb.Tags)
	if len(pb.Tags) == 0 {
		tags = "none"
	}
	fmt.Printf("Required Tags: %s\n\n", tags)

	fmt.Println("Diagnosis Refs:")
	for i, ref := range pb.Refs {
		if ref.Name != "" {
			fmt.Printf("  %d. %s: %s\n", i+1, ref.Name, ref.Desc)
		}
	}

	return nil
}

func newGenerateBookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "生成新的诊断方案",
		Long:  "根据描述生成新的诊断方案(Book)，需要指定playbook名称和方案描述",
		RunE:  runGenerateBook,
	}

	cmd.Flags().String("playbook", "", "playbook名称(必填)")
	cmd.Flags().String("name", "", "诊断方案名称(必填)")
	cmd.Flags().String("desc", "", "诊断方案描述(必填)")
	cmd.Flags().Bool("show", false, "是否显示模型详细输出信息")

	// 设置为必填参数
	_ = cmd.MarkFlagRequired("playbook")
	_ = cmd.MarkFlagRequired("desc")

	return cmd
}

func runGenerateBook(cmd *cobra.Command, args []string) error {
	playbookName, err := cmd.Flags().GetString("playbook")
	name, err := cmd.Flags().GetString("name")
	showDetails, _ := cmd.Flags().GetBool("show")

	if err != nil {
		return fmt.Errorf("获取playbook名称失败: %w", err)
	}

	description, err := cmd.Flags().GetString("desc")
	if err != nil {
		return fmt.Errorf("获取方案描述失败: %w", err)
	}

	conf, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	llmClient, err := llm.NewClient(context.Background(), &conf.LLM)
	if err != nil {
		return fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	// 创建Generator实例
	generator := playbook.NewGenerator(llmClient, conf.PlaybooksDir)

	// 准备请求参数
	req := playbook.GenerateBookRequest{
		Name:         name,
		PlaybookName: playbookName,
		Description:  description,
	}

	// 生成并保存诊断方案
	genbook, err := generator.GenerateAndSave(context.Background(), req, showDetails)
	if err != nil {
		return fmt.Errorf("生成诊断方案失败: %w", err)
	}

	fmt.Printf("诊断方案已成功生成到Playbook '%s'\n", playbookName)

	// 加载并显示生成的内容
	bytes, _ := json.Marshal(genbook)
	fmt.Println(string(bytes))

	return nil
}
