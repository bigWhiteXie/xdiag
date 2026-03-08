package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/bigWhiteXie/xdiag/internal/app/targets"
)

func newTargetGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "获取目标详情",
		Long:  "根据名称或ID获取目标资产的详细信息",
		Example: `
xdiag target get --name myserver
xdiag target get --id 1
`,
		RunE: runTargetGet,
	}

	cmd.Flags().String("name", "", "目标名称")
	cmd.Flags().String("id", "", "目标ID")

	return cmd
}

func runTargetGet(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	idStr, _ := cmd.Flags().GetString("id")

	repo, cleanup, err := initTargetRepo()
	if err != nil {
		return err
	}
	defer cleanup()

	var target *targets.Target
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
}
