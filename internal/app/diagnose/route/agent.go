package route

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bigWhiteXie/xdiag/internal/svc"
	innertool "github.com/bigWhiteXie/xdiag/internal/tool"
	"github.com/bigWhiteXie/xdiag/pkg/formatter"
	"github.com/bigWhiteXie/xdiag/pkg/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

const (
	template = `
你是一个专业的用户问题分析专家，能够根据用户的故障描述，使用可用的工具找到最合适的目标。

# 工作方式
1. 根据用户问题，先尽可能用精确的条件、多条件去查找相关target
2. 当找不到时逐步放宽条件直到找到target或发现target不存在
3. 不要进行联想，当放宽查询条件后仍未找到符合用户描述的target时，使用结构化输出工具返回status=2
4. **重要**：当找到合适的target后，必须立即调用结构化输出工具(output_result)返回结果，status=1并提供target_id

# 禁止
1. 禁止去分析报错原因

# 当前支持的target类型：%s
如果你需要使用kind查找target，必须使用上述类型
`

	questionTemplate = `
用户问题：%s

%s
`
)

type RouteTargetAgent struct {
	recAgent         *adk.ChatModelAgent
	executionHistory []string
	formatter        *formatter.AgentFormatter
}

func NewTargetRouteAgent(ctx context.Context, showDetails bool) (*RouteTargetAgent, error) {
	targetRepo := svc.GetServiceContext().TargetsRepo
	kinds, err := targetRepo.GetAllKinds()
	if err != nil {
		return nil, err
	}
	kindStr := strings.Join(kinds, ",")

	// 创建结构化输出工具
	structuredTool := innertool.NewStructuredOutputTool(innertool.StructuredOutputConfig{
		Description: "当确定target后或无相关目标时，使用此工具输出结果",
		Fields: []innertool.FieldDefinition{
			{
				Name:        "status",
				Type:        "number",
				Description: "1表示已找到目标，2表示没有相关目标",
				Required:    true,
				Example:     1,
			},
			{
				Name:        "target_id",
				Type:        "number",
				Description: "目标ID，仅当status为1时需要提供此字段",
				Required:    false,
				Example:     4,
			},
		},
	})

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "目标路由器",
		Description: "一个能够根据用户故障描述路由到最合适目标的agent",
		Instruction: fmt.Sprintf(template, kindStr),
		Model:       svc.GetServiceContext().Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					innertool.NewTargetFinderTool(svc.GetServiceContext().TargetsRepo),
					structuredTool,
				},
			},
		},
	})

	if err != nil {
		log.Fatal(fmt.Errorf("创建聊天模型失败: %w", err))
	}

	return &RouteTargetAgent{
		recAgent:         a,
		executionHistory: make([]string, 0),
		formatter:        formatter.NewAgentFormatter(showDetails),
	}, nil
}

func (a *RouteTargetAgent) Run(ctx context.Context, question string) (uint, error) {
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		targetId, err := a.run(ctx, question)
		if err == nil {
			return targetId, nil
		}
		lastErr = err
		log.Printf("[route agent] attempt %d/%d failed: %v", i+1, maxRetries, err)
	}

	return 0, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (a *RouteTargetAgent) run(ctx context.Context, question string) (uint, error) {
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           a.recAgent,
		EnableStreaming: false,
	})

	type Answer struct {
		Status   int  `json:"status"`
		TargetId uint `json:"target_id"`
	}

	// 构建包含历史记录的问题
	historyContext := ""
	if len(a.executionHistory) > 0 {
		historyContext = "# 历史执行记录\n" + strings.Join(a.executionHistory, "\n")
	}
	fullQuestion := fmt.Sprintf(questionTemplate, question, historyContext)

	// 使用 Query 方法，它会自动处理整个 ReAct 循环
	iter := runner.Query(ctx, fullQuestion)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		// 处理错误
		if event.Err != nil {
			return 0, fmt.Errorf("agent error: %w", event.Err)
		}

		// 格式化输出
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, _ := event.Output.MessageOutput.GetMessage()

			// 格式化 LLM 响应
			if event.Output.MessageOutput.Role == "assistant" {
				hasToolCall := len(msg.ToolCalls) > 0
				a.formatter.FormatLLMResponse(msg.Content, hasToolCall)
			}

			// 格式化工具调用
			if event.Output.MessageOutput.Role == "tool" {
				a.formatter.FormatToolResult(msg.Content)
			}
		}

		// 获取工具调用输出 - 从 MessageOutput 中提取
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				log.Printf("[route agent] failed to get message: %v", err)
				continue
			}

			// 检查是否是工具角色的消息
			if event.Output.MessageOutput.Role == "tool" && event.Output.MessageOutput.ToolName == innertool.StructOutputToolName {
				// 尝试反序列化工具输出
				var toolOutput innertool.StructuredOutputOutput
				if err := json.Unmarshal([]byte(msg.Content), &toolOutput); err != nil {
					log.Printf("[route agent] failed to parse tool output: %v", err)
					a.executionHistory = append(a.executionHistory, fmt.Sprintf("- 工具调用失败: 输出格式错误 - %v", err))
					continue
				}

				// 检查工具调用是否成功
				if toolOutput.Status != 1 {
					// 记录失败原因，继续执行
					a.executionHistory = append(a.executionHistory,
						fmt.Sprintf("- 工具调用失败: %s, 缺少字段: %v", toolOutput.Message, toolOutput.MissingFields))
					continue
				}

				// 将 map 反序列化为 Answer 结构体
				ans := &Answer{}
				if err := utils.UnmarshalMap(toolOutput.Data, ans); err != nil {
					log.Printf("[route agent] failed to unmarshal answer: %v", err)
					a.executionHistory = append(a.executionHistory, fmt.Sprintf("- 数据反序列化失败: %v", err))
					continue
				}

				// 成功获取结果
				if ans.Status == 1 {
					a.executionHistory = append(a.executionHistory, fmt.Sprintf("- 成功找到目标: target_id=%d", ans.TargetId))
					return ans.TargetId, nil
				} else if ans.Status == 2 {
					a.executionHistory = append(a.executionHistory, "- 未找到相关目标")
					return 0, nil
				}
			}
		}
	}

	// 如果循环结束仍未找到结果
	return 0, errors.New("agent completed without calling output_result tool")
}
