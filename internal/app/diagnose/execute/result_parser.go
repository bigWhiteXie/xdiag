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
func (p *resultParser) parseFromToolCall(message *schema.Message, expectedToolName string) (StepResult, error) {
	if message == nil {
		return StepResult{}, fmt.Errorf("没有可解析的消息")
	}

	// 检查是否有工具调用
	if message.Role != schema.Tool {
		return StepResult{SelectedCase: -1}, fmt.Errorf("没有找到结构化工具调用")
	}

	// 检查工具名称
	toolName := message.ToolName
	if toolName != expectedToolName {
		return StepResult{}, fmt.Errorf("工具名称不匹配，期望: %s, 实际: %s", expectedToolName, toolName)
	}
	res := StepResult{}
	if err := json.Unmarshal([]byte(message.Content), &res); err != nil {
		return StepResult{}, fmt.Errorf("解析工具调用结果失败: %v, 内容: %s", err, message.Content)
	}

	return res, nil
}

// parseFromTextContent 从文本内容解析结果（当模型返回文本而非工具调用时）
func (p *resultParser) parseFromTextContent(outputs *agentOutputs) (StepResult, error) {
	if outputs.textContent == "" {
		return StepResult{}, fmt.Errorf("没有可解析的文本内容")
	}

	content := cleanJSONContent(outputs.textContent)

	repaired, err := jsonrepair.JSONRepair(content)
	if err != nil {
		return StepResult{}, fmt.Errorf("JSON修复失败: %w", err)
	}

	// 创建临时消息用于解析
	tmpMsg := &schema.Message{
		Role:    schema.Assistant,
		Content: repaired,
	}

	result, err := p.contentParser.Parse(context.Background(), tmpMsg)
	if err != nil {
		return StepResult{}, fmt.Errorf("从文本内容解析失败: %w", err)
	}

	return result, nil
}
