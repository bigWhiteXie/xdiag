package match

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bigWhiteXie/xdiag/internal/tool"
	"github.com/bigWhiteXie/xdiag/pkg/formatter"
	"github.com/bigWhiteXie/xdiag/pkg/utils"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// LLMRunnerConfig LLM 运行器配置
type LLMRunnerConfig struct {
	MaxRetries int                    // 最大重试次数
	Formatter  *formatter.AgentFormatter // 格式化器
}

// LLMRunner LLM 调用执行器
type LLMRunner struct {
	chatModel ChatModelInterface
	config    LLMRunnerConfig
}

// NewLLMRunner 创建 LLM 运行器
func NewLLMRunner(chatModel ChatModelInterface, config LLMRunnerConfig) *LLMRunner {
	return &LLMRunner{
		chatModel: chatModel,
		config:    config,
	}
}

// RunWithStructuredOutput 使用结构化输出工具运行 LLM
func (r *LLMRunner) RunWithStructuredOutput(
	ctx context.Context,
	prompt string,
	toolConfig tool.StructuredOutputConfig,
	resultParser func(map[string]interface{}) (interface{}, error),
) (interface{}, error) {
	structTool := tool.NewStructuredOutputTool(toolConfig)

	toolInfo, err := structTool.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工具信息失败: %w", err)
	}

	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	for retry := 0; retry < r.config.MaxRetries; retry++ {
		resp, err := r.chatModel.Generate(ctx, messages, model.WithTools([]*schema.ToolInfo{toolInfo}))
		if err != nil {
			return nil, fmt.Errorf("LLM调用失败: %w, message: %s", err, utils.FormatMessages(messages))
		}

		r.config.Formatter.FormatLLMResponse(resp.Content, len(resp.ToolCalls) > 0)

		if !r.hasToolCall(resp) {
			messages = r.addFeedback(messages, resp, "你必须调用 output_result 工具来输出结果，请重新尝试。")
			continue
		}

		toolCall := resp.ToolCalls[0]
		r.config.Formatter.FormatToolCall(toolCall.Function.Name, toolCall.Function.Arguments)

		result, err := structTool.InvokableRun(ctx, toolCall.Function.Arguments)
		if err != nil {
			return nil, fmt.Errorf("执行结构化输出工具失败: %w", err)
		}

		r.config.Formatter.FormatToolResult(result)

		var output tool.StructuredOutputOutput
		if err := json.Unmarshal([]byte(result), &output); err != nil {
			return nil, fmt.Errorf("解析工具输出失败: %w", err)
		}

		// 检查是否有缺失字段
		if len(output.MissingFields) > 0 {
			feedbackMsg := fmt.Sprintf("缺少必填字段: %s", strings.Join(output.MissingFields, ", "))
			messages = append(messages,
				schema.AssistantMessage(resp.Content, resp.ToolCalls),
				schema.ToolMessage(result, toolCall.ID),
				schema.UserMessage(feedbackMsg))
			continue
		}

		// 解析结果
		parsedResult, err := resultParser(output.Data)
		if err != nil {
			return nil, err
		}

		return parsedResult, nil
	}

	return nil, fmt.Errorf("已达到最大重试次数 %d", r.config.MaxRetries)
}

// hasToolCall 检查响应是否包含工具调用
func (r *LLMRunner) hasToolCall(resp *schema.Message) bool {
	return len(resp.ToolCalls) > 0
}

// addFeedback 添加反馈消息
func (r *LLMRunner) addFeedback(messages []*schema.Message, resp *schema.Message, feedback string) []*schema.Message {
	if strings.TrimSpace(resp.Content) != "" {
		messages = append(messages, schema.AssistantMessage(resp.Content, nil))
	}
	return append(messages, schema.UserMessage(feedback))
}
