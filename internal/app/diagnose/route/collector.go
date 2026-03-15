package route

import (
	"encoding/json"

	innertool "github.com/bigWhiteXie/xdiag/internal/tool"
	"github.com/bigWhiteXie/xdiag/pkg/formatter"
	"github.com/cloudwego/eino/schema"
)

// agentOutputs 收集 agent 执行过程中的输出
type agentOutputs struct {
	showDetails     bool
	formatter       *formatter.AgentFormatter
	lastToolMessage *schema.Message
}

// newAgentOutputs 创建新的输出收集器
func newAgentOutputs(showDetails bool) *agentOutputs {
	return &agentOutputs{
		showDetails: showDetails,
		formatter:   formatter.NewAgentFormatter(showDetails),
	}
}

// addMessage 添加消息到收集器
func (o *agentOutputs) addMessage(msg *schema.Message) {
	// 如果是工具消息，保存为最后一条工具消息
	if msg.Role == schema.Tool {
		o.lastToolMessage = msg
	}

	// 格式化输出
	if msg.Role == schema.Assistant && msg.Content != "" {
		hasToolCall := len(msg.ToolCalls) > 0
		o.formatter.FormatLLMResponse(msg.Content, hasToolCall)
	}

	if msg.Role == schema.Tool && msg.Content != "" {
		o.formatter.FormatToolResult(msg.Content)
	}

	// 处理工具调用
	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			o.formatter.FormatToolCall(toolCall.Function.Name, toolCall.Function.Arguments)
		}
	}
}

// getTargetID 获取最后一条工具消息中的 targetID
// 如果是结构化工具调用且调用成功，返回 targetID 和 true
// 否则返回 0 和 false
func (o *agentOutputs) getTargetID() (uint, bool) {
	if o.lastToolMessage == nil {
		return 0, false
	}

	// 尝试解析为 StructuredOutputOutput
	var toolOutput innertool.StructuredOutputOutput
	if err := json.Unmarshal([]byte(o.lastToolMessage.Content), &toolOutput); err != nil {
		return 0, false
	}

	// 检查工具调用是否成功（status == 1）
	if toolOutput.Status != 1 {
		return 0, false
	}

	// 提取 routeResult 数据
	result := routeResult{}
	if err := unmarshalMap(toolOutput.Data, &result); err != nil {
		return 0, false
	}

	// 检查路由状态是否成功
	if result.Status == 1 || result.Status == 2 {
		return result.TargetId, true
	}

	return 0, false
}
