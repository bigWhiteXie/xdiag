package utils

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// FormatMessages 将 []*schema.Message 数组格式化为可读性强的字符串
// 清楚地显示每个消息的角色、内容、工具调用参数、工具返回结果等信息
func FormatMessages(messages []*schema.Message) string {
	if len(messages) == 0 {
		return "[]"
	}

	var buf bytes.Buffer

	for i, msg := range messages {
		buf.WriteString(formatSingleMessage(msg, i+1))
		buf.WriteString("\n")
	}

	return buf.String()
}

// formatSingleMessage 格式化单个 schema.Message
func formatSingleMessage(msg *schema.Message, index int) string {
	var buf bytes.Buffer

	// 前缀显示索引和角色
	roleStr := getRoleString(msg.Role)
	buf.WriteString(fmt.Sprintf("=== 消息 %d (%s) ===\n", index, roleStr))

	// 显示名称（如果有）
	if msg.Name != "" {
		buf.WriteString(fmt.Sprintf("名称: %s\n", msg.Name))
	}

	// 显示工具调用ID（如果有）
	if msg.ToolCallID != "" {
		buf.WriteString(fmt.Sprintf("工具调用ID: %s\n", msg.ToolCallID))
	}

	// 显示工具名称（如果有）
	if msg.ToolName != "" {
		buf.WriteString(fmt.Sprintf("工具名称: %s\n", msg.ToolName))
	}

	// 显示推理内容（如果有）
	if msg.ReasoningContent != "" {
		buf.WriteString("思考过程:\n")
		buf.WriteString(wrapText(msg.ReasoningContent, "  "))
		buf.WriteString("\n")
	}

	// 显示文本内容
	if msg.Content != "" {
		buf.WriteString("内容:\n")
		buf.WriteString(wrapText(msg.Content, "  "))
		buf.WriteString("\n")
	}

	// 显示多模态内容（旧格式，已弃用）
	if len(msg.MultiContent) > 0 {
		buf.WriteString("多模态内容 (旧格式):\n")
		for i, part := range msg.MultiContent {
			buf.WriteString(fmt.Sprintf("  部分 %d: 类型=%v\n", i+1, part.Type))
			if part.Text != "" {
				buf.WriteString(fmt.Sprintf("    文本: %s\n", part.Text))
			}
		}
	}

	// 显示用户输入多模态内容
	if len(msg.UserInputMultiContent) > 0 {
		buf.WriteString("用户输入多模态内容:\n")
		for i, part := range msg.UserInputMultiContent {
			buf.WriteString(fmt.Sprintf("  部分 %d: 类型=%v\n", i+1, part.Type))
			if part.Text != "" {
				buf.WriteString(fmt.Sprintf("    文本: %s\n", part.Text))
			}
		}
	}

	// 显示助手生成多模态内容
	if len(msg.AssistantGenMultiContent) > 0 {
		buf.WriteString("助手生成多模态内容:\n")
		for i, part := range msg.AssistantGenMultiContent {
			buf.WriteString(fmt.Sprintf("  部分 %d: 类型=%v\n", i+1, part.Type))
			if part.Text != "" {
				buf.WriteString(fmt.Sprintf("    文本: %s\n", part.Text))
			}
		}
	}

	// 显示工具调用
	if len(msg.ToolCalls) > 0 {
		buf.WriteString("工具调用:\n")
		for i, toolCall := range msg.ToolCalls {
			buf.WriteString(fmt.Sprintf("  工具调用 %d:\n", i+1))
			buf.WriteString(fmt.Sprintf("    ID: %s\n", toolCall.ID))
			buf.WriteString(fmt.Sprintf("    函数名: %s\n", toolCall.Function.Name))
			if toolCall.Function.Arguments != "" {
				buf.WriteString("    参数:\n")
				buf.WriteString(wrapText(toolCall.Function.Arguments, "      "))
				buf.WriteString("\n")
			}
		}
	}

	// 显示响应元数据
	if msg.ResponseMeta != nil {
		buf.WriteString("响应元数据:\n")
		if msg.ResponseMeta.FinishReason != "" {
			buf.WriteString(fmt.Sprintf("  结束原因: %s\n", msg.ResponseMeta.FinishReason))
		}
		if msg.ResponseMeta.Usage != nil {
			buf.WriteString(fmt.Sprintf("   Tokens: 输入=%d, 输出=%d, 总=%d\n",
				msg.ResponseMeta.Usage.PromptTokens,
				msg.ResponseMeta.Usage.CompletionTokens,
				msg.ResponseMeta.Usage.TotalTokens))
		}
		if msg.ResponseMeta.LogProbs != nil {
			buf.WriteString("  日志概率信息: 已包含\n")
		}
	}

	// 显示额外信息
	if len(msg.Extra) > 0 {
		buf.WriteString("额外信息:\n")
		for k, v := range msg.Extra {
			buf.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
	}

	return buf.String()
}

// getRoleString 将 RoleType 转换为可读字符串
func getRoleString(role schema.RoleType) string {
	switch role {
	case schema.System:
		return "System"
	case schema.User:
		return "User"
	case schema.Assistant:
		return "Assistant"
	case schema.Tool:
		return "Tool"
	default:
		return fmt.Sprintf("Unknown(%d)", role)
	}
}

// wrapText 对文本进行换行和缩进处理
func wrapText(text, indent string) string {
	if text == "" {
		return ""
	}

	var buf bytes.Buffer
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if line != "" {
			buf.WriteString(indent)
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}

	// 去掉最后一个换行符
	result := buf.String()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result
}
