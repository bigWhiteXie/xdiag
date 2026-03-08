package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"xdiag/internal/app/playbook"
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
	playbookCmd.AddCommand(newUpdatePlaybookCmd())
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
	playbooksDir := viper.GetString("playbooks_dir")
	if playbooksDir == "" {
		return fmt.Errorf("无法获取playbooks目录")
	}

	repo := playbook.NewRepo(playbooksDir)
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
	playbooksDir := viper.GetString("playbooks_dir")
	if playbooksDir == "" {
		return fmt.Errorf("无法获取playbooks目录")
	}

	repo := playbook.NewRepo(playbooksDir)
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

func newUpdatePlaybookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "从GitHub更新Playbook",
		Long:  "从GitHub仓库下载最新的Playbook定义文件",
		RunE:  runUpdatePlaybook,
	}
}

func runUpdatePlaybook(cmd *cobra.Command, args []string) error {
	fmt.Println("正在从GitHub拉取最新的诊断方案...")

	playbooksDir := viper.GetString("playbooks_dir")
	if playbooksDir == "" {
		return fmt.Errorf("无法获取playbooks目录")
	}

	gitDir := filepath.Join(playbooksDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		fmt.Println("检测到现有git仓库，正在更新...")
		gitCmd := exec.Command("git", "pull")
		gitCmd.Dir = playbooksDir
		output, err := gitCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("更新失败: %w\n输出: %s", err, output)
		}
		fmt.Println("Playbook更新成功!")
		fmt.Printf("输出: %s\n", output)
	} else {
		repoURL := "https://github.com/example/xdiag-playbooks.git"
		if repoURL != "" {
			fmt.Printf("正在克隆仓库到: %s\n", playbooksDir)

			if _, err := exec.LookPath("git"); err != nil {
				return fmt.Errorf("系统中未找到git命令，请先安装git")
			}

			gitCmd := exec.Command("git", "clone", repoURL, playbooksDir)
			output, err := gitCmd.CombinedOutput()
			if err != nil {
				fmt.Printf("克隆失败: %v\n输出: %s\n", err, output)
				fmt.Println("使用示例仓库URL进行演示，实际部署时请替换为真实仓库")
			} else {
				fmt.Println("Playbook克隆成功!")
			}
		}
	}

	fmt.Println("Playbook更新完成")
	return nil
}
