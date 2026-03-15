package execute

import (
	"strings"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/bigWhiteXie/xdiag/internal/app/targets"
)

const (
	thinkTagStart  = "<think>"
	thinkTagEnd    = "</think>"
	thinkTagEndLen = 8
)

// ExecuteState 定义执行状态
type ExecuteState struct {
	Book           *playbook.Book
	Target         *targets.Target
	Question       string
	ExecutedSteps  []ExecutedStep // 记录执行过的步骤和结果
	CurrentContext string         // 当前步骤的执行上下文
	RetryCount     int            // 记录重试次数
	Error          string
	StepStack      []StepContext // 步骤栈，用于处理嵌套的分支
	EventChan      chan<- ExecuteEvent
}

// StepContext 步骤上下文，用于处理嵌套分支
type StepContext struct {
	Steps        []playbook.Step
	CurrentIndex int
}

// ExecutedStep 已执行的步骤
type ExecutedStep struct {
	Step   playbook.Step
	Result StepResult
}

// StepResult 步骤执行结果
type StepResult struct {
	Status       int    `json:"status"`        // 0: 未完成, 1: 已完成
	Result       string `json:"result"`        // 步骤执行中发现的信息、执行操作的结果等
	SelectedCase int    `json:"selected_case"` // 对于branch类型，选择的分支索引（从0开始）
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	Success   bool
	Report    string
	Error     string
	EventChan <-chan ExecuteEvent
}

// ExecuteEvent 执行事件
type ExecuteEvent struct {
	Type    string // step_start, step_complete, step_error, branch_select, complete, agent_thinking, agent_tool_call, agent_tool_result
	Step    *playbook.Step
	Result  *StepResult
	Message string
	Error   string
}

// cleanJSONContent 清理消息内容，提取纯 JSON
func cleanJSONContent(content string) string {
	// 移除成对的 <think> 标签及其内容
	for {
		startIdx := strings.Index(content, thinkTagStart)
		if startIdx == -1 {
			break
		}
		endIdx := strings.Index(content[startIdx:], thinkTagEnd)
		if endIdx == -1 {
			// 如果没有找到结束标签，移除单独的开始标签
			content = content[:startIdx] + content[startIdx+len(thinkTagStart):]
			continue
		}
		content = content[:startIdx] + content[startIdx+endIdx+thinkTagEndLen:]
	}

	// 移除单独出现的 </think> 标签
	content = strings.ReplaceAll(content, thinkTagEnd, "")

	// 查找第一个 { 和最后一个 }
	startIdx := strings.Index(content, "{")
	if startIdx == -1 {
		return content
	}

	endIdx := strings.LastIndex(content, "}")
	if endIdx == -1 || endIdx < startIdx {
		return content
	}

	return strings.TrimSpace(content[startIdx : endIdx+1])
}
