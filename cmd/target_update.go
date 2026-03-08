package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newTargetUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "更新目标",
		Long:  "更新现有目标资产的信息",
		RunE:  runTargetUpdate,
	}

	cmd.Flags().String("id", "", "目标ID (必需)")
	cmd.Flags().String("name", "", "目标名称")
	cmd.Flags().String("kind", "", "目标类型 (node/postgres/mysql/redis)")
	cmd.Flags().String("address", "", "目标地址")
	cmd.Flags().Int("port", 0, "目标端口")
	cmd.Flags().String("username", "", "用户名")
	cmd.Flags().String("password", "", "密码")
	cmd.Flags().String("ssh-key", "", "SSH 密钥路径")
	cmd.Flags().String("tags", "", "逗号分隔的标签")

	cmd.MarkFlagRequired("id")

	return cmd
}

func runTargetUpdate(cmd *cobra.Command, args []string) error {
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

	repo, cleanup, err := initTargetRepo()
	if err != nil {
		return err
	}
	defer cleanup()

	existingTarget, err := repo.GetByID(context.Background(), uint(id))
	if err != nil {
		return fmt.Errorf("failed to get target: %w", err)
	}

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

	if err := repo.Update(context.Background(), existingTarget); err != nil {
		return fmt.Errorf("failed to update target: %w", err)
	}

	fmt.Printf("✅ Target '%s' (ID: %d) updated successfully\n", existingTarget.Name, existingTarget.ID)
	return nil
}
