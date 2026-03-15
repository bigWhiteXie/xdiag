package execute

import (
	"fmt"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
)

const (
	EventTypeStepStart       = "step_start"
	EventTypeStepComplete    = "step_complete"
	EventTypeStepError       = "step_error"
	EventTypeBranchSelect    = "branch_select"
	EventTypeComplete        = "complete"
	EventTypeAgentThinking   = "agent_thinking"
	EventTypeAgentToolCall   = "agent_tool_call"
	EventTypeAgentToolResult = "agent_tool_result"
)

const (
	StepKindBranch = "branch"
)

// eventEmitter 负责发送执行事件
type eventEmitter struct {
	showDetails bool
}

// newEventEmitter 创建新的事件发射器
func newEventEmitter(showDetails bool) *eventEmitter {
	return &eventEmitter{showDetails: showDetails}
}

// send 发送事件
func (e *eventEmitter) send(eventChan chan<- ExecuteEvent, event ExecuteEvent) {
	if eventChan != nil {
		eventChan <- event
	}
}

// sendStepStart 发送步骤开始事件
func (e *eventEmitter) sendStepStart(eventChan chan<- ExecuteEvent, step playbook.Step) {
	e.send(eventChan, ExecuteEvent{
		Type:    EventTypeStepStart,
		Step:    &step,
		Message: fmt.Sprintf("开始执行步骤: %s", step.Desc),
	})
}

// sendStepComplete 发送步骤完成事件
func (e *eventEmitter) sendStepComplete(eventChan chan<- ExecuteEvent, step playbook.Step, result StepResult) {
	e.send(eventChan, ExecuteEvent{
		Type:   EventTypeStepComplete,
		Step:   &step,
		Result: &result,
	})
}

// sendAgentThinking 发送agent思考事件
func (e *eventEmitter) sendAgentThinking(eventChan chan<- ExecuteEvent, step playbook.Step, content string) {
	if !e.showDetails {
		return
	}
	e.send(eventChan, ExecuteEvent{
		Type:    EventTypeAgentThinking,
		Step:    &step,
		Message: content,
	})
}

// sendAgentToolCall 发送agent工具调用事件
func (e *eventEmitter) sendAgentToolCall(eventChan chan<- ExecuteEvent, step playbook.Step, toolName, args string) {
	if !e.showDetails {
		return
	}
	e.send(eventChan, ExecuteEvent{
		Type:    EventTypeAgentToolCall,
		Step:    &step,
		Message: fmt.Sprintf("调用工具: %s\n参数: %s", toolName, args),
	})
}

// sendAgentToolResult 发送agent工具结果事件
func (e *eventEmitter) sendAgentToolResult(eventChan chan<- ExecuteEvent, step playbook.Step, result string) {
	if !e.showDetails {
		return
	}
	e.send(eventChan, ExecuteEvent{
		Type:    EventTypeAgentToolResult,
		Step:    &step,
		Message: fmt.Sprintf("工具结果: %s", result),
	})
}
