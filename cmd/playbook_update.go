package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bigWhiteXie/xdiag/internal/config"
	"github.com/bigWhiteXie/xdiag/pkg/logger"
	"go.uber.org/zap"
)

func newUpdatePlaybookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "从GitHub更新playbooks和scripts",
		Long:  "从xdiag-books仓库克隆或更新playbooks和scripts到本地配置目录",
		RunE:  runUpdatePlaybook,
	}

	cmd.Flags().String("repo", "https://github.com/bigWhiteXie/xdiag-books.git", "GitHub仓库地址")

	return cmd
}

func runUpdatePlaybook(cmd *cobra.Command, args []string) error {
	repoURL, _ := cmd.Flags().GetString("repo")

	// 临时目录
	tmpDir := filepath.Join(os.TempDir(), "xdiag-books")

	// 如果临时目录已存在，先删除
	if _, err := os.Stat(tmpDir); err == nil {
		fmt.Printf("清理旧的临时目录: %s\n", tmpDir)
		if err := os.RemoveAll(tmpDir); err != nil {
			return fmt.Errorf("清理临时目录失败: %w", err)
		}
	}

	// 克隆仓库
	fmt.Printf("正在克隆仓库 %s 到 %s...\n", repoURL, tmpDir)
	cloneCmd := exec.Command("git", "clone", repoURL, tmpDir)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("克隆仓库失败: %w", err)
	}

	// 获取配置目录
	configDir := config.GetConfigDir()

	// 移动playbooks
	srcPlaybooks := filepath.Join(tmpDir, "playbooks")
	dstPlaybooks := filepath.Join(configDir, "playbooks")
	if _, err := os.Stat(srcPlaybooks); err == nil {
		fmt.Printf("正在更新playbooks到 %s...\n", dstPlaybooks)
		if err := copyDir(srcPlaybooks, dstPlaybooks); err != nil {
			return fmt.Errorf("更新playbooks失败: %w", err)
		}
		fmt.Println("✓ playbooks更新成功")
	} else {
		fmt.Println("仓库中未找到playbooks目录")
	}

	// 移动scripts
	srcScripts := filepath.Join(tmpDir, "scripts")
	dstScripts := filepath.Join(configDir, "scripts")
	if _, err := os.Stat(srcScripts); err == nil {
		fmt.Printf("正在更新scripts到 %s...\n", dstScripts)
		if err := copyDir(srcScripts, dstScripts); err != nil {
			return fmt.Errorf("更新scripts失败: %w", err)
		}
		fmt.Println("✓ scripts更新成功")
	} else {
		fmt.Println("仓库中未找到scripts目录")
	}

	// 清理临时目录
	fmt.Printf("清理临时目录: %s\n", tmpDir)
	if err := os.RemoveAll(tmpDir); err != nil {
		logger.Warn("清理临时目录失败", zap.Error(err))
	}

	fmt.Println("\n更新完成!")
	return nil
}

// copyDir 递归复制目录，如果目标文件存在则覆盖
func copyDir(src, dst string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile 复制单个文件，如果目标文件存在则覆盖
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}
