package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"xdiag/internal/app/targets"
)

func newTargetListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出目标",
		Long:  "列出所有目标资产",
		RunE:  runTargetList,
	}

	cmd.Flags().String("kind", "", "按类型过滤")
	cmd.Flags().String("tag", "", "按标签过滤")

	return cmd
}

func runTargetList(cmd *cobra.Command, args []string) error {
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

	repo, cleanup, err := initTargetRepo()
	if err != nil {
		return err
	}
	defer cleanup()

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
}
