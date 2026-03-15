package route

import (
	"context"
	"fmt"
	"strings"

	"github.com/bigWhiteXie/xdiag/internal/svc"
	innertool "github.com/bigWhiteXie/xdiag/internal/tool"
	"github.com/bigWhiteXie/xdiag/pkg/logger"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"go.uber.org/zap"
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
	recAgent  *adk.ChatModelAgent
	collector *agentOutputs
	parser    *resultParser
	history   *executionHistory
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
		WrapData:    true,
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
		logger.Fatal("创建聊天模型失败", zap.Error(err))
	}

	return &RouteTargetAgent{
		recAgent: a,
		parser:   newResultParser(),
		history:  newExecutionHistory(),
	}, nil
}

// buildFullQuestion 构建包含历史记录的完整问题
func (a *RouteTargetAgent) buildFullQuestion(question string) string {
	historyContext := a.history.buildContext()
	return fmt.Sprintf(questionTemplate, question, historyContext)
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
		logger.Warn("route agent attempt failed", zap.Int("attempt", i+1), zap.Int("maxRetries", maxRetries), zap.Error(err))
	}

	return 0, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (a *RouteTargetAgent) run(ctx context.Context, question string) (uint, error) {
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           a.recAgent,
		EnableStreaming: false,
	})
	ctxWithCanceld, cancel := context.WithCancel(ctx)
	a.collector = newAgentOutputs(true)

	iter := runner.Query(ctxWithCanceld, a.buildFullQuestion(question))

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return 0, fmt.Errorf("agent error: %w", event.Err)
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			// 判断是否是output工具调用，是则立马退出
			a.collector.addMessage(msg)
			if id, ok := a.collector.getTargetID(); ok {
				cancel()
				return id, nil
			}
		}
	}

	// 解析结果
	targetId, err := a.parser.parseFromToolCall(a.collector)
	if err != nil {
		// Fallback
		a.history.addFailure(err.Error())
		return 0, err
	}

	if targetId > 0 {
		a.history.addSuccess(targetId)
	} else {
		a.history.addNotFound()
	}

	return targetId, nil
}
