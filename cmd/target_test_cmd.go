package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"xdiag/internal/app/targets"
)

func newTargetTestConnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "测试目标连通性",
		Long:  "测试目标资产的连通性和认证",
		Example: `
xdiag target test --name myserver
xdiag target test --id 1
`,
		RunE: runTargetTestConn,
	}

	cmd.Flags().String("name", "", "目标名称")
	cmd.Flags().String("id", "", "目标ID")

	return cmd
}

func runTargetTestConn(cmd *cobra.Command, args []string) error {
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

	tester, err := targets.NewConnectivityTester(target.Kind)
	if err != nil {
		return fmt.Errorf("failed to create connectivity tester: %w", err)
	}

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
}
