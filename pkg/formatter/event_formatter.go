package formatter

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// Event 执行事件接口
type Event interface {
	GetType() string
	GetMessage() string
	GetError() string
}

// StepEvent 步骤事件
type StepEvent interface {
	Event
	GetStepDesc() string
	GetResult() string
}

// EventFormatter 格式化输出执行事件
type EventFormatter struct {
	showDetails bool
	stepCounter int
}

// NewEventFormatter 创建新的事件格式化器
func NewEventFormatter(showDetails bool) *EventFormatter {
	return &EventFormatter{
		showDetails: showDetails,
		stepCounter: 0,
	}
}

// AgentFormatter 格式化 Agent 输出
type AgentFormatter struct {
	showDetails bool
}

// NewAgentFormatter 创建新的 Agent 格式化器
func NewAgentFormatter(showDetails bool) *AgentFormatter {
	return &AgentFormatter{
		showDetails: showDetails,
	}
}

// FormatToolCall 格式化工具调用
func (f *AgentFormatter) FormatToolCall(toolName string, args string) {
	if !f.showDetails {
		return
	}

	toolColor := color.New(color.FgBlue)
	fmt.Print(toolColor.Sprint("🔧 工具调用: "))
	fmt.Println(toolName)

	if args != "" {
		argsColor := color.New(color.FgCyan)
		fmt.Print(argsColor.Sprint("   参数: "))
		fmt.Println(args)
	}
}

// FormatToolResult 格式化工具结果
func (f *AgentFormatter) FormatToolResult(result string) {
	if !f.showDetails {
		return
	}

	resultColor := color.New(color.FgGreen)
	fmt.Print(resultColor.Sprint("   结果: "))

	// 如果结果太长，截断显示
	if len(result) > 200 {
		fmt.Println(result[:200] + "...")
	} else {
		fmt.Println(result)
	}
}

// FormatThinking 格式化思考过程
func (f *AgentFormatter) FormatThinking(content string) {
	if !f.showDetails {
		return
	}

	thinkColor := color.New(color.FgYellow)
	fmt.Print(thinkColor.Sprint("💭 思考: "))
	fmt.Println(content)
}

// FormatLLMCall 格式化 LLM 调用
func (f *AgentFormatter) FormatLLMCall(prompt string) {
	if !f.showDetails {
		return
	}

	llmColor := color.New(color.FgMagenta)
	fmt.Println(llmColor.Sprint("🤖 LLM 调用"))

	// 只显示 prompt 的前 100 个字符
	if len(prompt) > 100 {
		fmt.Printf("   提示词: %s...\n", prompt[:100])
	} else {
		fmt.Printf("   提示词: %s\n", prompt)
	}
}

// FormatLLMResponse 格式化 LLM 响应
func (f *AgentFormatter) FormatLLMResponse(content string, hasToolCall bool) {
	if !f.showDetails {
		return
	}

	responseColor := color.New(color.FgGreen)
	fmt.Print(responseColor.Sprint("   响应: "))

	if hasToolCall {
		fmt.Println("[调用工具]")
	} else if len(content) > 150 {
		fmt.Println(content[:150] + "...")
	} else {
		fmt.Println(content)
	}
}

// FormatEvent 格式化单个事件
func (f *EventFormatter) FormatEvent(eventType string, stepDesc string, result string, message string, errMsg string) string {
	if !f.showDetails {
		return ""
	}

	var sb strings.Builder

	switch eventType {
	case "step_start":
		f.stepCounter++
		sb.WriteString(f.formatStepStart(stepDesc))
	case "step_complete":
		sb.WriteString(f.formatStepComplete(result))
	case "step_error":
		sb.WriteString(f.formatStepError(errMsg))
	case "branch_select":
		sb.WriteString(f.formatBranchSelect(message, result))
	case "complete":
		sb.WriteString(f.formatComplete(errMsg))
	}

	return sb.String()
}

// formatStepStart 格式化步骤开始事件
func (f *EventFormatter) formatStepStart(stepDesc string) string {
	var sb strings.Builder

	titleColor := color.New(color.FgCyan, color.Bold)
	sb.WriteString("\n")
	sb.WriteString(titleColor.Sprintf("┌─ 步骤 %d ─────────────────────────────────────────────────────────\n", f.stepCounter))
	sb.WriteString(titleColor.Sprintf("│ %s\n", stepDesc))
	sb.WriteString(titleColor.Sprint("└───────────────────────────────────────────────────────────────────\n"))

	return sb.String()
}

// formatStepComplete 格式化步骤完成事件
func (f *EventFormatter) formatStepComplete(result string) string {
	var sb strings.Builder

	successColor := color.New(color.FgGreen)
	labelColor := color.New(color.FgYellow)

	sb.WriteString(successColor.Sprint("✓ 步骤完成\n"))
	sb.WriteString(labelColor.Sprint("执行结果:\n"))
	sb.WriteString(f.formatResult(result))
	sb.WriteString("\n")

	return sb.String()
}

// formatStepError 格式化步骤错误事件
func (f *EventFormatter) formatStepError(errMsg string) string {
	var sb strings.Builder

	errorColor := color.New(color.FgRed, color.Bold)
	sb.WriteString(errorColor.Sprint("✗ 步骤执行失败\n"))
	sb.WriteString(errorColor.Sprintf("错误: %s\n\n", errMsg))

	return sb.String()
}

// formatBranchSelect 格式化分支选择事件
func (f *EventFormatter) formatBranchSelect(message string, result string) string {
	var sb strings.Builder

	branchColor := color.New(color.FgMagenta)
	labelColor := color.New(color.FgYellow)

	sb.WriteString(branchColor.Sprint("⎇ 分支选择\n"))
	sb.WriteString(labelColor.Sprint("选择的分支:\n"))
	sb.WriteString(fmt.Sprintf("  %s\n", message))
	sb.WriteString(labelColor.Sprint("选择理由:\n"))
	sb.WriteString(f.formatResult(result))
	sb.WriteString("\n")

	return sb.String()
}

// formatComplete 格式化完成事件
func (f *EventFormatter) formatComplete(errMsg string) string {
	var sb strings.Builder

	if errMsg != "" {
		errorColor := color.New(color.FgRed, color.Bold)
		sb.WriteString("\n")
		sb.WriteString(errorColor.Sprint("═══════════════════════════════════════════════════════════════════\n"))
		sb.WriteString(errorColor.Sprint("  执行失败\n"))
		sb.WriteString(errorColor.Sprint("═══════════════════════════════════════════════════════════════════\n"))
		sb.WriteString(fmt.Sprintf("%s\n", errMsg))
	} else {
		successColor := color.New(color.FgGreen, color.Bold)
		sb.WriteString("\n")
		sb.WriteString(successColor.Sprint("═══════════════════════════════════════════════════════════════════\n"))
		sb.WriteString(successColor.Sprint("  执行完成\n"))
		sb.WriteString(successColor.Sprint("═══════════════════════════════════════════════════════════════════\n"))
	}

	return sb.String()
}

// formatResult 格式化结果文本，添加缩进
func (f *EventFormatter) formatResult(result string) string {
	lines := strings.Split(result, "\n")
	var sb strings.Builder

	for _, line := range lines {
		if line != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	return sb.String()
}
