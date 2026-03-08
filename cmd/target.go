package cmd

import (
	"github.com/spf13/cobra"
)

var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "管理目标资产",
	Long:  "用于添加、查询、更新、删除和测试目标资产的连通性",
}

func init() {
	rootCmd.AddCommand(targetCmd)
	targetCmd.AddCommand(newTargetAddCmd())
	targetCmd.AddCommand(newTargetListCmd())
	targetCmd.AddCommand(newTargetGetCmd())
	targetCmd.AddCommand(newTargetUpdateCmd())
	targetCmd.AddCommand(newTargetDeleteCmd())
	targetCmd.AddCommand(newTargetTestConnCmd())
}
