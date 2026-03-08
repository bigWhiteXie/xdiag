package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newTargetDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "删除目标",
		Long:  "根据ID删除目标资产",
		RunE:  runTargetDelete,
	}

	cmd.Flags().String("id", "", "目标ID (必需)")
	cmd.MarkFlagRequired("id")

	return cmd
}

func runTargetDelete(cmd *cobra.Command, args []string) error {
	idStr, _ := cmd.Flags().GetString("id")

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	repo, cleanup, err := initTargetRepo()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := repo.Delete(context.Background(), uint(id)); err != nil {
		return fmt.Errorf("failed to delete target: %w", err)
	}

	fmt.Printf("✅ Target with ID %d deleted successfully\n", id)
	return nil
}
