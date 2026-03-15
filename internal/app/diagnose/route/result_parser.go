package route

import (
	"encoding/json"
	"fmt"

	innertool "github.com/bigWhiteXie/xdiag/internal/tool"
	"github.com/cloudwego/eino/schema"
)

// routeResult 路由结果结构
type routeResult struct {
	Status   int  `json:"status"`
	TargetId uint `json:"target_id"`
}

// resultParser 负责解析 agent 输出
type resultParser struct {
	contentParser schema.MessageParser[routeResult]
}

// newResultParser 创建新的解析器
func newResultParser() *resultParser {
	return &resultParser{
		contentParser: schema.NewMessageJSONParser[routeResult](&schema.MessageJSONParseConfig{
			ParseFrom: schema.MessageParseFromContent,
		}),
	}
}

// parseFromToolCall 从工具调用解析结果
func (p *resultParser) parseFromToolCall(outputs *agentOutputs) (uint, error) {
	// 检查最后一条消息是否为工具消息
	if outputs.lastToolMessage == nil {
		return 0, fmt.Errorf("最后一条消息不是工具消息")
	}

	// 尝试反序列化为 StructuredOutputOutput
	var toolOutput innertool.StructuredOutputOutput
	if err := json.Unmarshal([]byte(outputs.lastToolMessage.Content), &toolOutput); err != nil {
		return 0, fmt.Errorf("解析工具输出失败: %w", err)
	}

	// 检查工具调用是否成功
	if toolOutput.Status != 1 {
		return 0, fmt.Errorf("工具调用失败: %s, 缺少字段: %v", toolOutput.Message, toolOutput.MissingFields)
	}

	// 提取数据
	result := routeResult{}
	if err := unmarshalMap(toolOutput.Data, &result); err != nil {
		return 0, fmt.Errorf("反序列化结果失败: %w", err)
	}

	// 检查状态
	if result.Status == 1 {
		return result.TargetId, nil
	} else if result.Status == 2 {
		return 0, nil
	}

	return 0, fmt.Errorf("无效的 status 值: %d", result.Status)
}

// unmarshalMap 将 map[string]interface{} 反序列化到结构体
func unmarshalMap(data map[string]interface{}, target interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, target)
}

// cleanJSONContent 清理 JSON 内容
func cleanJSONContent(content string) string {
	// TODO: 实现清理逻辑，移除 think 标签等
	return content
}
