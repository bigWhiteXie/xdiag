package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"xdiag/internal/app/targets"
	"xdiag/internal/config"
)

func newTargetAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "添加目标",
		Long:  "添加一个新的目标资产",
		Example: `
# 添加一个节点
xdiag target add --name prod-server --kind node --address 192.168.1.100 --port 22 --username admin --ssh-key /path/to/key --tags production,web

# 添加一个 PostgreSQL 数据库
xdiag target add --name prod-db --kind postgres --address db.example.com --port 5432 --username postgres --password secret --tags production,db
`,
		RunE: runTargetAdd,
	}

	cmd.Flags().String("kind", "", "目标类型 (node/postgres/mysql/redis) (必需)")
	cmd.Flags().String("address", "", "目标地址 (必需)")
	cmd.Flags().Int("port", 0, "目标端口")
	cmd.Flags().String("username", "", "用户名")
	cmd.Flags().String("password", "", "密码")
	cmd.Flags().String("ssh-key", "", "SSH 密钥路径")
	cmd.Flags().String("tags", "", "逗号分隔的标签")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("kind")
	cmd.MarkFlagRequired("address")

	return cmd
}

func runTargetAdd(cmd *cobra.Command, args []string) error {
	kind, _ := cmd.Flags().GetString("kind")
	address, _ := cmd.Flags().GetString("address")
	port, _ := cmd.Flags().GetInt("port")
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")
	tags, _ := cmd.Flags().GetString("tags")

	if kind == "" || address == "" {
		return fmt.Errorf("--kind, and --address are required")
	}

	name := fmt.Sprintf("%s-%s:%d", kind, address, port)
	target := &targets.Target{
		Name:     name,
		Kind:     kind,
		Address:  address,
		Port:     port,
		Username: username,
		Password: password,
		Tags:     tags,
	}

	repo, cleanup, err := initTargetRepo()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := repo.Create(context.Background(), target); err != nil {
		return fmt.Errorf("failed to add target: %w", err)
	}

	fmt.Printf("✅ Target '%s' added successfully with ID %d\n", target.Name, target.ID)
	return nil
}

func initTargetRepo() (*targets.SQLiteRepo, func(), error) {
	conf, _ := config.LoadConfig()

	dbPath := filepath.Join(conf.DataDir, "targets.db")
	repo, err := targets.NewSQLiteRepo(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	return repo, func() { repo.Close() }, nil
}
