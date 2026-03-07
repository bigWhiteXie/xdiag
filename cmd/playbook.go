package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"xdiag/internal/app/playbook"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var playbookCmd = &cobra.Command{
	Use:   "playbook",
	Short: "管理诊断剧本(Playbook)",
	Long:  `管理诊断剧本，包括列表、显示、更新和执行等操作。`,
}

var listPlaybookCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有可用的Playbook",
	Long:  `列出所有在系统中注册的Playbook及其基本信息。`,
	Run: func(cmd *cobra.Command, args []string) {
		playbooksDir := viper.GetString("playbooks_dir")
		if playbooksDir == "" {
			log.Fatal("无法获取playbooks目录")
		}

		repo := playbook.NewRepo(playbooksDir)
		playbooks, err := repo.ListPlaybooks(nil) // 不使用标签过滤
		if err != nil {
			log.Printf("加载Playbook列表时出错: %v", err)
			return
		}

		fmt.Println("Available Playbooks:")
		fmt.Println("NAME\t\t\tDESC\t\t\tREQUIRED_TAGS")
		for _, pb := range playbooks {
			tags := fmt.Sprintf("%v", pb.Tags)
			if len(pb.Tags) == 0 {
				tags = "none"
			}
			// 使用目录名作为显示名称
			dirName := filepath.Base(pb.Path)
			fmt.Printf("%-20s\t%-30s\t%s\n", dirName, pb.Desc, tags)
		}
	},
}

var showPlaybookCmd = &cobra.Command{
	Use:   "show [playbook-name]",
	Short: "显示指定Playbook的详细信息",
	Long:  `显示指定名称的Playbook的详细信息，包括其所有诊断方案。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		playbooksDir := viper.GetString("playbooks_dir")
		if playbooksDir == "" {
			log.Fatal("无法获取playbooks目录")
		}

		repo := playbook.NewRepo(playbooksDir)
		playbooks, err := repo.ListPlaybooks(nil)
		if err != nil {
			log.Printf("加载Playbook '%s' 时出错: %v", name, err)
			return
		}

		// 找到指定的playbook
		var pb *playbook.Playbook
		for _, p := range playbooks {
			if filepath.Base(p.Path) == name {
				pb = &p
				break
			}
		}

		if pb == nil {
			log.Printf("找不到Playbook '%s'", name)
			return
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
	},
}

var updatePlaybookCmd = &cobra.Command{
	Use:   "update",
	Short: "从GitHub更新Playbook",
	Long:  `从GitHub仓库下载最新的Playbook定义文件。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("正在从GitHub拉取最新的诊断方案...")

		playbooksDir := viper.GetString("playbooks_dir")
		if playbooksDir == "" {
			log.Fatal("无法获取playbooks目录")
		}

		// 尝试检查git仓库是否存在，如果存在则pull，否则clone
		gitDir := filepath.Join(playbooksDir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			// 目录是git仓库，执行pull
			fmt.Println("检测到现有git仓库，正在更新...")
			cmd := exec.Command("git", "pull")
			cmd.Dir = playbooksDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("更新失败: %v\n", err)
				fmt.Printf("输出: %s\n", output)
			} else {
				fmt.Println("Playbook更新成功!")
				fmt.Printf("输出: %s\n", output)
			}
		} else {
			// 尝试克隆仓库 (这里使用一个示例仓库URL，实际使用时需要替换成真实的仓库)
			repoURL := "https://github.com/example/xdiag-playbooks.git"
			if repoURL != "" {
				fmt.Printf("正在克隆仓库到: %s\n", playbooksDir)

				// 检查git是否可用
				if _, err := exec.LookPath("git"); err != nil {
					fmt.Println("错误: 系统中未找到git命令，请先安装git")
					return
				}

				cmd := exec.Command("git", "clone", repoURL, playbooksDir)
				output, err := cmd.CombinedOutput()
				if err != nil {
					fmt.Printf("克隆失败: %v\n", err)
					fmt.Printf("输出: %s\n", output)
					// 如果克隆失败，可能是仓库不存在，但我们仍然可以继续
					fmt.Println("使用示例仓库URL进行演示，实际部署时请替换为真实仓库")
				} else {
					fmt.Println("Playbook克隆成功!")
				}
			}
		}

		fmt.Println("Playbook更新完成")
	},
}

func init() {
	playbookCmd.AddCommand(listPlaybookCmd)
	playbookCmd.AddCommand(showPlaybookCmd)
	playbookCmd.AddCommand(updatePlaybookCmd)
	rootCmd.AddCommand(playbookCmd)
}
