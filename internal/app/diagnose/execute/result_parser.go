package execute

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/kaptinlin/jsonrepair"
)

const (
	seqResultToolName    = "output_seq_result"
	branchResultToolName = "output_branch_result"
)

// resultParser 负责解析agent输出
type resultParser struct {
	contentParser schema.MessageParser[StepResult]
}

// newResultParser 创建新的解析器
func newResultParser() *resultParser {
	return &resultParser{

		contentParser: schema.NewMessageJSONParser[StepResult](&schema.MessageJSONParseConfig{
			ParseFrom: schema.MessageParseFromContent,
		}),
	}
}

// parseFromToolCall 从工具调用解析结果
func (p *resultParser) parseFromToolCall(outputs *agentOutputs, expectedToolName string) (StepResult, error) {
	if outputs.lastMessage == nil {
		return StepResult{}, fmt.Errorf("没有可解析的消息")
	}

	// 检查是否有工具调用
	if outputs.lastMessage.Role != schema.Tool {
		return StepResult{SelectedCase: -1}, fmt.Errorf("没有找到结构化工具调用")
	}

	// 检查工具名称
	toolName := outputs.lastMessage.ToolName
	if toolName != expectedToolName {
		return StepResult{}, fmt.Errorf("工具名称不匹配，期望: %s, 实际: %s", expectedToolName, toolName)
	}
	res := StepResult{}
	if err := json.Unmarshal([]byte(outputs.lastMessage.Content), &res); err != nil {
		return StepResult{}, fmt.Errorf("解析工具调用结果失败: %v, 内容: %s", err, outputs.lastMessage.Content)
	}

	return res, nil
}

// parseFromContent 从消息内容解析结果（fallback）
func (p *resultParser) parseFromContent(outputs *agentOutputs) (StepResult, error) {
	if outputs.lastMessage == nil {
		return StepResult{}, fmt.Errorf("没有可解析的消息")
	}

	content := outputs.lastMessage.Content
	content = cleanJSONContent(content)

	repaired, err := jsonrepair.JSONRepair(content)
	if err != nil {
		return StepResult{}, fmt.Errorf("JSON修复失败: %w", err)
	}

	// 更新消息内容并解析
	outputs.lastMessage.Content = repaired
	return p.contentParser.Parse(context.Background(), outputs.lastMessage)
}
