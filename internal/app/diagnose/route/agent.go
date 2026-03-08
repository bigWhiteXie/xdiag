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
	"github.com/bigWhiteXie/xdiag/pkg/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	template = `
你是一个专业的用户问题分析专家，能够根据用户的故障描述，使用可用的工具找到最合适的目标，当无相关目标时将status设置为2即可。当确定target后需要按照如下格式输出字符串此时表示当前行为已经结束：
<output>
{
   "status": 1, // 1表示已找到目标，2表示没有相关目标
   "target_id": 4 // 目标ID，仅当status为1时包含此字段
}
</output>
严格按照示例格式返回，不要包含任何额外的文本或解释。

# 工作方式
根据用户问题，先尽可能用精确的条件、多条件去查找相关target，当找不到时在逐步放宽条件直到找到target或发现target不存在。
不要进行联想，当放宽查询条件后仍未找到符合用户描述的target时按要求返回即可

# 禁止
1. 禁止去分析报错原因

# 当前支持的target类型：%s
如果你需要使用kind查找target，必须使用上述类型
`
)

type RouteTargetAgent struct {
	recAgent *adk.ChatModelAgent
}

func NewTargetRouteAgent(ctx context.Context) (*RouteTargetAgent, error) {
	targetRepo := svc.GetServiceContext().TargetsRepo
	kinds, err := targetRepo.GetAllKinds()
	if err != nil {
		return nil, err
	}
	kindStr := strings.Join(kinds, ",")
	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "目标路由器",
		Description: "一个能够根据用户故障描述路由到最合适目标的agent",
		Instruction: fmt.Sprintf(template, kindStr),
		Model:       svc.GetServiceContext().Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{innertool.NewTargetFinderTool(svc.GetServiceContext().TargetsRepo)},
			},
		},
	})

	if err != nil {
		log.Fatal(fmt.Errorf("创建聊天模型失败: %w", err))
	}

	return &RouteTargetAgent{
		recAgent: a,
	}, nil
}

func (a *RouteTargetAgent) Run(ctx context.Context, question string) (uint, error) {
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           a.recAgent,
		EnableStreaming: false,
	})

	type Answer struct {
		Status   int  `json:"status"`
		TargetId uint `json:"target_id"`
	}

	// 使用 Query 方法，它会自动处理整个 ReAct 循环
	iter := runner.Query(ctx, question)

	var lastMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		// 记录事件日志
		str, _ := json.Marshal(event)
		log.Printf("[route agent event]: %s", str)

		// 处理错误
		if event.Err != nil {
			return 0, fmt.Errorf("agent error: %w", event.Err)
		}

		// 获取输出消息
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				log.Printf("[route agent] failed to get message: %v", err)
				continue
			}

			// 只处理 assistant 角色的消息
			if msg.Role == schema.Assistant {
				lastMessage = msg
			}
		}
	}

	// 处理最终结果
	if lastMessage == nil {
		return 0, errors.New("no response from agent")
	}

	jsonStr := utils.ParseJsonByLabel("output", lastMessage.Content)
	if jsonStr == "" {
		return 0, fmt.Errorf("no output tag found in response: %s", lastMessage.Content)
	}

	ans := &Answer{}
	if err := json.Unmarshal([]byte(jsonStr), ans); err != nil {
		return 0, fmt.Errorf("failed to parse json: %w, content: %s", err, jsonStr)
	}

	if ans.Status == 1 {
		return ans.TargetId, nil
	} else if ans.Status == 2 {
		return 0, nil
	}

	return 0, fmt.Errorf("unexpected status: %d", ans.Status)
}
