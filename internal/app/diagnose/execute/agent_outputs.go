package execute

import (
	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/cloudwego/eino/schema"
)

// agentOutputs 负责收集agent执行过程中的输出
type agentOutputs struct {
	showDetails bool
	lastMessage *schema.Message
}

// newAgentOutputs 创建新的agent输出收集器
func newAgentOutputs(showDetails bool) *agentOutputs {
	return &agentOutputs{showDetails: showDetails}
}

// addMessage 添加消息并发送相应事件
func (o *agentOutputs) addMessage(msg *schema.Message, eventChan chan<- ExecuteEvent, step playbook.Step) {
	emitter := newEventEmitter(o.showDetails)

	// 记录助手消息
	if msg.Role == schema.Assistant && msg.Content != "" {
		if o.lastMessage == nil {
			o.lastMessage = msg
		}
		emitter.sendAgentThinking(eventChan, step, msg.Content)
	}

	// 记录工具调用
	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			emitter.sendAgentToolCall(eventChan, step, toolCall.Function.Name, toolCall.Function.Arguments)
		}
	}

	// 记录工具执行结果
	if msg.Role == schema.Tool && msg.Content != "" {
		o.lastMessage = msg
		emitter.sendAgentToolResult(eventChan, step, msg.Content)
	}
}
