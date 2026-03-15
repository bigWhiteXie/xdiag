package execute

import (
	"fmt"
	"strings"
)

// reportGenerator 负责生成诊断报告
type reportGenerator struct{}

// newReportGenerator 创建新的报告生成器
func newReportGenerator() *reportGenerator {
	return &reportGenerator{}
}

// generate 生成诊断报告
func (r *reportGenerator) generate(state *ExecuteState) string {
	var sb strings.Builder

	sb.WriteString("# 诊断报告\n\n")
	r.writeBasicInfo(&sb, state)
	r.writeExecutionStatus(&sb, state)
	r.writeExecutionDetails(&sb, state)
	r.writeSummary(&sb, state)

	return sb.String()
}

// writeBasicInfo 写入基本信息
func (r *reportGenerator) writeBasicInfo(sb *strings.Builder, state *ExecuteState) {
	sb.WriteString("## 基本信息\n\n")
	fmt.Fprintf(sb, "- **诊断方案**: %s\n", state.Book.Name)
	fmt.Fprintf(sb, "- **目标名称**: %s\n", state.Target.Name)
	fmt.Fprintf(sb, "- **目标类型**: %s\n", state.Target.Kind)
	fmt.Fprintf(sb, "- **目标地址**: %s:%d\n", state.Target.Address, state.Target.Port)
	fmt.Fprintf(sb, "- **用户问题**: %s\n\n", state.Question)
}

// writeExecutionStatus 写入执行状态
func (r *reportGenerator) writeExecutionStatus(sb *strings.Builder, state *ExecuteState) {
	sb.WriteString("## 执行状态\n\n")
	if state.Error != "" {
		fmt.Fprintf(sb, "**执行失败**: %s\n\n", state.Error)
	} else {
		sb.WriteString("**执行成功**\n\n")
	}
}

// writeExecutionDetails 写入执行详情
func (r *reportGenerator) writeExecutionDetails(sb *strings.Builder, state *ExecuteState) {
	sb.WriteString("## 执行详情\n\n")
	for i, executed := range state.ExecutedSteps {
		fmt.Fprintf(sb, "### 步骤 %d: %s\n\n", i+1, executed.Step.Desc)
		fmt.Fprintf(sb, "- **类型**: %s\n", executed.Step.Kind)
		fmt.Fprintf(sb, "- **结果**: %s\n\n", executed.Result.Result)
	}
}

// writeSummary 写入总结
func (r *reportGenerator) writeSummary(sb *strings.Builder, state *ExecuteState) {
	sb.WriteString("## 总结\n\n")
	if state.Error != "" {
		fmt.Fprintf(sb, "诊断过程中遇到错误，已完成 %d/%d 个步骤。\n", len(state.ExecutedSteps), len(state.Book.Steps))
	} else {
		fmt.Fprintf(sb, "诊断成功完成，共执行 %d 个步骤。\n", len(state.ExecutedSteps))
	}
}
