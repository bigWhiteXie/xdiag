package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"xdiag/internal/app/targets"
)

var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "管理目标资产",
	Long:  "用于添加、查询、更新、删除和测试目标资产的连通性",
}

func init() {
	rootCmd.AddCommand(targetCmd)
	targetCmd.AddCommand(targetAddCmd)
	targetCmd.AddCommand(targetListCmd)
	targetCmd.AddCommand(targetGetCmd)
	targetCmd.AddCommand(targetUpdateCmd)
	targetCmd.AddCommand(targetDeleteCmd)
	targetCmd.AddCommand(targetTestCmd)
}

// targetAddCmd 添加目标
var targetAddCmd = &cobra.Command{
	Use:   "add",
	Short: "添加目标",
	Long:  "添加一个新的目标资产",
	Example: `
# 添加一个节点
xdiag target add --name prod-server --kind node --address 192.168.1.100 --port 22 --username admin --ssh-key /path/to/key --tags production,web

# 添加一个 PostgreSQL 数据库
xdiag target add --name prod-db --kind postgres --address db.example.com --port 5432 --username postgres --password secret --tags production,db
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		kind, _ := cmd.Flags().GetString("kind")
		address, _ := cmd.Flags().GetString("address")
		port, _ := cmd.Flags().GetInt("port")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		tags, _ := cmd.Flags().GetString("tags")

		if name == "" || kind == "" || address == "" {
			return fmt.Errorf("--name, --kind, and --address are required")
		}

		// 创建目标对象
		target := &targets.Target{
			Name:     name,
			Kind:     kind,
			Address:  address,
			Port:     port,
			Username: username,
			Password: password,
			Tags:     tags,
		}

		// 确保配置目录存在
		configDir := filepath.Join(os.Getenv("HOME"), ".xdiag")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// 初始化存储
		dbPath := filepath.Join(configDir, "targets.db")
		repo, err := targets.NewSQLiteRepo(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}
		defer repo.Close()

		// 保存目标
		err = repo.Create(context.Background(), target)
		if err != nil {
			return fmt.Errorf("failed to add target: %w", err)
		}

		fmt.Printf("✅ Target '%s' added successfully with ID %d\n", target.Name, target.ID)
		return nil
	},
}

func init() {
	targetAddCmd.Flags().String("name", "", "目标名称 (必需)")
	targetAddCmd.Flags().String("kind", "", "目标类型 (node/postgres/mysql/redis) (必需)")
	targetAddCmd.Flags().String("address", "", "目标地址 (必需)")
	targetAddCmd.Flags().Int("port", 0, "目标端口")
	targetAddCmd.Flags().String("username", "", "用户名")
	targetAddCmd.Flags().String("password", "", "密码")
	targetAddCmd.Flags().String("ssh-key", "", "SSH 密钥路径")
	targetAddCmd.Flags().String("tags", "", "逗号分隔的标签")

	targetAddCmd.MarkFlagRequired("name")
	targetAddCmd.MarkFlagRequired("kind")
	targetAddCmd.MarkFlagRequired("address")
}

// targetListCmd 列出目标
var targetListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出目标",
	Long:  "列出所有目标资产",
	RunE: func(cmd *cobra.Command, args []string) error {
		kindFilter, _ := cmd.Flags().GetString("kind")
		tagFilter, _ := cmd.Flags().GetString("tag")

		filters := make(map[string]targets.Op)
		if kindFilter != "" {
			filters["kind"] = targets.Op{
				Op:  "eq",
				Val: kindFilter,
			}
		}
		if tagFilter != "" {
			filters["tag"] = targets.Op{
				Op:  "like",
				Val: tagFilter,
			}
		}

		// 确保配置目录存在
		configDir := filepath.Join(os.Getenv("HOME"), ".xdiag")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// 初始化存储
		dbPath := filepath.Join(configDir, "targets.db")
		repo, err := targets.NewSQLiteRepo(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}
		defer repo.Close()

		// 获取目标列表
		targetsList, err := repo.List(context.Background(), filters)
		if err != nil {
			return fmt.Errorf("failed to list targets: %w", err)
		}

		if len(targetsList) == 0 {
			fmt.Println("No targets found")
			return nil
		}

		fmt.Printf("Found %d target(s):\n", len(targetsList))
		for _, t := range targetsList {
			fmt.Printf("- ID: %d, Name: %s, Kind: %s, Address: %s:%d, Tags: %s\n",
				t.ID, t.Name, t.Kind, t.Address, t.Port, t.Tags)
		}

		return nil
	},
}

func init() {
	targetListCmd.Flags().String("kind", "", "按类型过滤")
	targetListCmd.Flags().String("tag", "", "按标签过滤")
}

// targetGetCmd 获取目标详情
var targetGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取目标详情",
	Long:  "根据名称或ID获取目标资产的详细信息",
	Example: `
xdiag target get --name myserver
xdiag target get --id 1
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		idStr, _ := cmd.Flags().GetString("id")

		var target *targets.Target
		var err error

		// 确保配置目录存在
		configDir := filepath.Join(os.Getenv("HOME"), ".xdiag")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// 初始化存储
		dbPath := filepath.Join(configDir, "targets.db")
		repo, err := targets.NewSQLiteRepo(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}
		defer repo.Close()

		if name != "" {
			target, err = repo.GetByName(context.Background(), name)
		} else if idStr != "" {
			id, convErr := strconv.ParseUint(idStr, 10, 32)
			if convErr != nil {
				return fmt.Errorf("invalid ID format: %w", convErr)
			}
			target, err = repo.GetByID(context.Background(), uint(id))
		} else {
			return fmt.Errorf("--name or --id is required")
		}

		if err != nil {
			return fmt.Errorf("failed to get target: %w", err)
		}

		fmt.Printf("Target Details:\n")
		fmt.Printf("  ID: %d\n", target.ID)
		fmt.Printf("  Name: %s\n", target.Name)
		fmt.Printf("  Kind: %s\n", target.Kind)
		fmt.Printf("  Address: %s:%d\n", target.Address, target.Port)
		fmt.Printf("  Username: %s\n", target.Username)
		fmt.Printf("  Tags: %s\n", target.Tags)
		fmt.Printf("  Created At: %s\n", target.CreatedAt)
		fmt.Printf("  Updated At: %s\n", target.UpdatedAt)

		return nil
	},
}

func init() {
	targetGetCmd.Flags().String("name", "", "目标名称")
	targetGetCmd.Flags().String("id", "", "目标ID")
}

// targetUpdateCmd 更新目标
var targetUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新目标",
	Long:  "更新现有目标资产的信息",
	RunE: func(cmd *cobra.Command, args []string) error {
		idStr, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		kind, _ := cmd.Flags().GetString("kind")
		address, _ := cmd.Flags().GetString("address")
		port, _ := cmd.Flags().GetInt("port")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		tags, _ := cmd.Flags().GetString("tags")

		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid ID format: %w", err)
		}

		// 确保配置目录存在
		configDir := filepath.Join(os.Getenv("HOME"), ".xdiag")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// 初始化存储
		dbPath := filepath.Join(configDir, "targets.db")
		repo, err := targets.NewSQLiteRepo(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}
		defer repo.Close()

		// 获取现有目标
		existingTarget, err := repo.GetByID(context.Background(), uint(id))
		if err != nil {
			return fmt.Errorf("failed to get target: %w", err)
		}

		// 更新字段
		if name != "" {
			existingTarget.Name = name
		}
		if kind != "" {
			existingTarget.Kind = kind
		}
		if address != "" {
			existingTarget.Address = address
		}
		if port != 0 {
			existingTarget.Port = port
		}
		if username != "" {
			existingTarget.Username = username
		}
		if password != "" {
			existingTarget.Password = password
		}

		if tags != "" {
			existingTarget.Tags = tags
		}

		// 保存更新
		err = repo.Update(context.Background(), existingTarget)
		if err != nil {
			return fmt.Errorf("failed to update target: %w", err)
		}

		fmt.Printf("✅ Target '%s' (ID: %d) updated successfully\n", existingTarget.Name, existingTarget.ID)
		return nil
	},
}

func init() {
	targetUpdateCmd.Flags().String("id", "", "目标ID (必需)")
	targetUpdateCmd.Flags().String("name", "", "目标名称")
	targetUpdateCmd.Flags().String("kind", "", "目标类型 (node/postgres/mysql/redis)")
	targetUpdateCmd.Flags().String("address", "", "目标地址")
	targetUpdateCmd.Flags().Int("port", 0, "目标端口")
	targetUpdateCmd.Flags().String("username", "", "用户名")
	targetUpdateCmd.Flags().String("password", "", "密码")
	targetUpdateCmd.Flags().String("ssh-key", "", "SSH 密钥路径")
	targetUpdateCmd.Flags().String("tags", "", "逗号分隔的标签")

	targetUpdateCmd.MarkFlagRequired("id")
}

// targetDeleteCmd 删除目标
var targetDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除目标",
	Long:  "根据ID删除目标资产",
	RunE: func(cmd *cobra.Command, args []string) error {
		idStr, _ := cmd.Flags().GetString("id")

		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid ID format: %w", err)
		}

		// 确保配置目录存在
		configDir := filepath.Join(os.Getenv("HOME"), ".xdiag")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// 初始化存储
		dbPath := filepath.Join(configDir, "targets.db")
		repo, err := targets.NewSQLiteRepo(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}
		defer repo.Close()

		// 删除目标
		err = repo.Delete(context.Background(), uint(id))
		if err != nil {
			return fmt.Errorf("failed to delete target: %w", err)
		}

		fmt.Printf("✅ Target with ID %d deleted successfully\n", id)
		return nil
	},
}

func init() {
	targetDeleteCmd.Flags().String("id", "", "目标ID (必需)")

	targetDeleteCmd.MarkFlagRequired("id")
}

// targetTestCmd 测试目标连通性
var targetTestCmd = &cobra.Command{
	Use:   "test",
	Short: "测试目标连通性",
	Long:  "测试目标资产的连通性和认证",
	Example: `
xdiag target test --name myserver
xdiag target test --id 1
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		idStr, _ := cmd.Flags().GetString("id")

		var target *targets.Target
		var err error

		// 确保配置目录存在
		configDir := filepath.Join(os.Getenv("HOME"), ".xdiag")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// 初始化存储
		dbPath := filepath.Join(configDir, "targets.db")
		repo, err := targets.NewSQLiteRepo(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}
		defer repo.Close()

		if name != "" {
			target, err = repo.GetByName(context.Background(), name)
		} else if idStr != "" {
			id, convErr := strconv.ParseUint(idStr, 10, 32)
			if convErr != nil {
				return fmt.Errorf("invalid ID format: %w", convErr)
			}
			target, err = repo.GetByID(context.Background(), uint(id))
		} else {
			return fmt.Errorf("--name or --id is required")
		}

		if err != nil {
			return fmt.Errorf("failed to get target: %w", err)
		}

		// 创建连通性测试器
		tester, err := targets.NewConnectivityTester(target.Kind)
		if err != nil {
			return fmt.Errorf("failed to create connectivity tester: %w", err)
		}

		// 执行测试
		result, err := tester.Test(context.Background(), target)
		if err != nil {
			return fmt.Errorf("failed to test connectivity: %w", err)
		}

		fmt.Printf("Connectivity Test Result for '%s' (ID: %d):\n", target.Name, target.ID)
		fmt.Printf("  Status: %s\n", result.Status)
		fmt.Printf("  Ping Status: %s\n", result.PingStatus)
		fmt.Printf("  Auth Status: %s\n", result.AuthStatus)
		fmt.Printf("  Message: %s\n", result.Message)

		if len(result.ExtraDetails) > 0 {
			fmt.Printf("  Extra Details:\n")
			for key, value := range result.ExtraDetails {
				fmt.Printf("    %s: %s\n", key, value)
			}
		}

		return nil
	},
}

func init() {
	targetTestCmd.Flags().String("name", "", "目标名称")
	targetTestCmd.Flags().String("id", "", "目标ID")
}
