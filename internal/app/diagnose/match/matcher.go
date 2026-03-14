package match

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/bigWhiteXie/xdiag/internal/app/targets"
	"github.com/bigWhiteXie/xdiag/internal/tool"
	"github.com/bigWhiteXie/xdiag/pkg/formatter"
	"github.com/bigWhiteXie/xdiag/pkg/logger"
	"github.com/bigWhiteXie/xdiag/pkg/utils"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// MatchState 定义状态机的状态
type MatchState struct {
	Target            *targets.Target
	Question          string
	AllPlaybooks      []playbook.Playbook
	ExcludedPlaybooks []string
	SelectedPlaybook  *playbook.Playbook
	SelectedRef       *playbook.Ref
	RefStatus         int // 0: 未找到合适的ref, 1: 找到合适的ref
}

// PlaybookSelection LLM 步骤1的输出
type PlaybookSelection struct {
	PlaybookName string `json:"playbook_name"`
	Reason       string `json:"reason"`
}

// RefSelection LLM 步骤2的输出
type RefSelection struct {
	RefName string `json:"ref_name"`
	Status  int    `json:"status"` // 0: 未找到合适的ref, 1: 找到合适的ref
	Reason  string `json:"reason"`
}

// MatchResult 匹配结果
type MatchResult struct {
	Playbook *playbook.Playbook
	Ref      *playbook.Ref
	Success  bool
	Message  string
}

// ChatModelInterface 定义匹配器需要的最小 LLM 接口
type ChatModelInterface interface {
	Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error)
}

// Matcher 方案匹配器
type Matcher struct {
	repo       playbook.Repo
	chatModel  ChatModelInterface
	graph      compose.Runnable[*MatchState, *MatchState]
	maxRetries int // 每个节点的最大重试次数
	formatter  *formatter.AgentFormatter
}

// NewMatcher 创建新的方案匹配器
func NewMatcher(repo playbook.Repo, chatModel ChatModelInterface, showDetails bool) (*Matcher, error) {
	m := &Matcher{
		repo:       repo,
		chatModel:  chatModel,
		maxRetries: 3, // 默认最大重试3次
		formatter:  formatter.NewAgentFormatter(showDetails),
	}

	// 构建 Graph
	graph, err := m.buildGraph()
	if err != nil {
		return nil, fmt.Errorf("构建graph失败: %w", err)
	}
	m.graph = graph

	return m, nil
}

// Match 执行匹配
func (m *Matcher) Match(ctx context.Context, target *targets.Target, question string) (*MatchResult, error) {
	// 加载所有playbooks
	allPlaybooks, err := m.repo.ListPlaybooks(nil)
	if err != nil {
		return nil, fmt.Errorf("加载playbooks失败: %w", err)
	}

	// 初始化状态
	state := &MatchState{
		Target:            target,
		Question:          question,
		AllPlaybooks:      allPlaybooks,
		ExcludedPlaybooks: []string{},
		RefStatus:         0,
	}

	// 执行graph
	finalState, err := m.graph.Invoke(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("执行匹配失败: %w", err)
	}

	// 构建结果
	return m.buildResult(finalState), nil
}

// buildGraph 构建状态机图
func (m *Matcher) buildGraph() (compose.Runnable[*MatchState, *MatchState], error) {
	graph := compose.NewGraph[*MatchState, *MatchState]()

	// 添加节点
	err := graph.AddLambdaNode("select_playbook", compose.InvokableLambda(m.selectPlaybookNode))
	if err != nil {
		return nil, fmt.Errorf("添加select_playbook节点失败: %w", err)
	}

	err = graph.AddLambdaNode("select_ref", compose.InvokableLambda(m.selectRefNode))
	if err != nil {
		return nil, fmt.Errorf("添加select_ref节点失败: %w", err)
	}

	err = graph.AddLambdaNode("finish", compose.InvokableLambda(m.finishNode))
	if err != nil {
		return nil, fmt.Errorf("添加finish节点失败: %w", err)
	}

	// 设置入口
	graph.AddEdge(compose.START, "select_playbook")

	// 添加边
	// select_playbook -> select_ref (找到playbook)
	graph.AddEdge("select_playbook", "select_ref")

	// select_ref -> finish (找到合适的ref)
	// select_ref -> select_playbook (未找到合适的ref，重新选择playbook)
	err = graph.AddBranch("select_ref", compose.NewGraphBranch(func(ctx context.Context, state *MatchState) (string, error) {
		if state.RefStatus == 1 {
			return "finish", nil
		}
		// 检查是否还有可选的playbook
		if len(state.ExcludedPlaybooks) >= len(state.AllPlaybooks) {
			return "finish", nil
		}
		return "select_playbook", nil
	}, map[string]bool{
		"finish":          true,
		"select_playbook": true,
	}))
	if err != nil {
		return nil, fmt.Errorf("添加分支失败: %w", err)
	}

	// finish -> END
	graph.AddEdge("finish", compose.END)

	// 编译graph
	compiled, err := graph.Compile(context.Background())
	if err != nil {
		return nil, fmt.Errorf("编译graph失败: %w", err)
	}

	return compiled, nil
}

// selectPlaybookNode 步骤1: 选择playbook
func (m *Matcher) selectPlaybookNode(ctx context.Context, state *MatchState) (*MatchState, error) {
	// 过滤掉已排除的playbooks
	availablePlaybooks := []playbook.Playbook{}
	for _, pb := range state.AllPlaybooks {
		excluded := false
		for _, excludedName := range state.ExcludedPlaybooks {
			if pb.Name == excludedName {
				excluded = true
				break
			}
		}
		if !excluded {
			availablePlaybooks = append(availablePlaybooks, pb)
		}
	}

	if len(availablePlaybooks) == 0 {
		return state, fmt.Errorf("没有可用的playbook")
	}

	// 构建playbook列表的描述
	playbooksDesc := m.buildPlaybooksDescription(availablePlaybooks)

	// 构建提示词
	prompt := fmt.Sprintf(`你是一个智能诊断助手。请根据以下信息选择最合适的诊断方案(playbook)。

目标资产信息:
- 名称: %s
- 类型: %s
- 地址: %s:%d
- 标签: %s

用户问题: %s

可用的诊断方案:
%s

请分析目标资产的类型、标签和用户问题，选择最合适的诊断方案。`,
		state.Target.Name,
		state.Target.Kind,
		state.Target.Address,
		state.Target.Port,
		state.Target.Tags,
		state.Question,
		playbooksDesc,
	)

	// 创建结构化输出工具
	structTool := tool.NewStructuredOutputTool(tool.StructuredOutputConfig{
		Description: "当确定playbook后或无相关playbook时，使用此工具输出结果",
		Fields: []tool.FieldDefinition{
			{
				Name:        "playbook_name",
				Type:        "string",
				Description: "选择的playbook名称",
				Required:    true,
				Example:     "mysql_diagnostics",
			},
			{
				Name:        "reason",
				Type:        "string",
				Description: "选择该playbook的理由",
				Required:    true,
				Example:     "该方案适用于MySQL数据库的性能诊断",
			},
		},
	})

	// 获取工具信息
	toolInfo, err := structTool.Info(ctx)
	if err != nil {
		return state, fmt.Errorf("获取工具信息失败: %w", err)
	}

	// 重试循环
	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	for retry := 0; retry < m.maxRetries; retry++ {
		// 调用LLM with tool
		resp, err := m.chatModel.Generate(ctx, messages,
			model.WithTools([]*schema.ToolInfo{toolInfo}))
		if err != nil {
			return state, fmt.Errorf("LLM调用失败: %w", err)
		}

		// 格式化输出 LLM 响应
		m.formatter.FormatLLMResponse(resp.Content, len(resp.ToolCalls) > 0)

		// 检查是否返回了tool call
		if len(resp.ToolCalls) == 0 {
			// 未进行工具调用，添加提示消息并重试
			feedbackMsg := "你必须调用 output_result 工具来输出结果，请重新尝试。"
			messages = append(messages,
				schema.AssistantMessage(resp.Content, nil),
				schema.UserMessage(feedbackMsg))
			continue
		}

		// 执行工具调用
		toolCall := resp.ToolCalls[0]
		m.formatter.FormatToolCall(toolCall.Function.Name, toolCall.Function.Arguments)

		result, err := structTool.InvokableRun(ctx, toolCall.Function.Arguments)
		if err != nil {
			return state, fmt.Errorf("执行结构化输出工具失败: %w", err)
		}

		m.formatter.FormatToolResult(result)

		// 解析结果
		var output tool.StructuredOutputOutput
		if err := json.Unmarshal([]byte(result), &output); err != nil {
			return state, fmt.Errorf("解析工具输出失败: %w", err)
		}

		// 检查工具调用状态
		if output.Status != 1 {

			return state, errors.New("未找到相关合适的playbook")
		}

		// 成功获取结果
		var selection PlaybookSelection
		if err := utils.UnmarshalMap(output.Data, &selection); err != nil {
			return state, fmt.Errorf("转换数据失败: %w", err)
		}

		// 查找选中的playbook
		for i := range availablePlaybooks {
			if availablePlaybooks[i].Name == selection.PlaybookName {
				state.SelectedPlaybook = &availablePlaybooks[i]
				return state, nil
			}
		}

		// 未找到对应的playbook，反馈给模型
		feedbackMsg := fmt.Sprintf("未找到名为 '%s' 的playbook，请从可用的诊断方案列表中选择一个存在的方案。", selection.PlaybookName)
		messages = append(messages,
			schema.AssistantMessage(resp.Content, resp.ToolCalls),
			schema.UserMessage(feedbackMsg))
	}

	return state, fmt.Errorf("选择playbook失败: 已达到最大重试次数 %d", m.maxRetries)
}

// selectRefNode 步骤2: 选择ref
func (m *Matcher) selectRefNode(ctx context.Context, state *MatchState) (*MatchState, error) {
	if state.SelectedPlaybook == nil {
		return state, fmt.Errorf("未选择playbook")
	}

	// 构建refs列表的描述
	refsDesc := m.buildRefsDescription(state.SelectedPlaybook.Refs)

	// 构建提示词
	prompt := fmt.Sprintf(`你是一个智能诊断助手。请根据以下信息从当前诊断方案中选择最合适的具体诊断参考(ref)。

目标资产信息:
- 名称: %s
- 类型: %s
- 地址: %s:%d
- 标签: %s

用户问题: %s

当前诊断方案: %s
方案描述: %s

可用的诊断参考:
%s

请分析目标资产和用户问题，选择最合适的诊断参考。如果没有合适的诊断参考，请将status设置为0。`,
		state.Target.Name,
		state.Target.Kind,
		state.Target.Address,
		state.Target.Port,
		state.Target.Tags,
		state.Question,
		state.SelectedPlaybook.Name,
		state.SelectedPlaybook.Desc,
		refsDesc,
	)

	// 创建结构化输出工具
	structTool := tool.NewStructuredOutputTool(tool.StructuredOutputConfig{
		Description: "当确定相关ref或当前无相关ref时使用此工具结构化输出",
		Fields: []tool.FieldDefinition{
			{
				Name:        "ref_name",
				Type:        "string",
				Description: "选择的ref名称(如果没有合适的则为空字符串)",
				Required:    false,
				Example:     "slow_query_analysis",
			},
			{
				Name:        "status",
				Type:        "number",
				Description: "1表示找到合适的ref, 0表示未找到",
				Required:    true,
				Example:     1,
			},
			{
				Name:        "reason",
				Type:        "string",
				Description: "选择理由或未找到的原因",
				Required:    true,
				Example:     "该ref适用于慢查询分析场景",
			},
		},
	})

	// 获取工具信息
	toolInfo, err := structTool.Info(ctx)
	if err != nil {
		return state, fmt.Errorf("获取工具信息失败: %w", err)
	}

	// 重试循环
	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	for retry := 0; retry < m.maxRetries; retry++ {
		// 调用LLM with tool
		resp, err := m.chatModel.Generate(ctx, messages,
			model.WithTools([]*schema.ToolInfo{toolInfo}))
		if err != nil {
			logger.Errorf("LLM调用失败: %w", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// 格式化输出 LLM 响应
		m.formatter.FormatLLMResponse(resp.Content, len(resp.ToolCalls) > 0)

		// 检查是否返回了tool call
		if len(resp.ToolCalls) == 0 {
			// 未进行工具调用，添加提示消息并重试
			feedbackMsg := "你必须调用 output_result 工具来输出结果，请重新尝试。"
			messages = append(messages,
				schema.AssistantMessage(resp.Content, nil),
				schema.UserMessage(feedbackMsg))
			continue
		}

		// 执行工具调用
		toolCall := resp.ToolCalls[0]
		m.formatter.FormatToolCall(toolCall.Function.Name, toolCall.Function.Arguments)

		result, err := structTool.InvokableRun(ctx, toolCall.Function.Arguments)
		if err != nil {
			return state, fmt.Errorf("执行结构化输出工具失败: %w", err)
		}

		m.formatter.FormatToolResult(result)

		// 解析结果
		var output tool.StructuredOutputOutput
		if err := json.Unmarshal([]byte(result), &output); err != nil {
			return state, fmt.Errorf("解析工具输出失败: %w", err)
		}

		// 检查工具调用状态
		if output.Status != 1 {
			// 工具调用失败，将缺失信息反馈给模型
			messages = append(messages,
				schema.AssistantMessage(resp.Content, resp.ToolCalls),
				schema.ToolMessage(result, toolCall.ID))
			continue
		}

		// 成功获取结果
		var selection RefSelection
		if err := utils.UnmarshalMap(output.Data, &selection); err != nil {
			return state, fmt.Errorf("转换数据失败: %w", err)
		}

		state.RefStatus = selection.Status

		if selection.Status == 1 {
			// 找到合适的ref
			for i := range state.SelectedPlaybook.Refs {
				if state.SelectedPlaybook.Refs[i].Name == selection.RefName {
					state.SelectedRef = &state.SelectedPlaybook.Refs[i]
					return state, nil
				}
			}

			// 未找到对应的ref，反馈给模型
			feedbackMsg := fmt.Sprintf("未找到名为 '%s' 的ref，请从可用的诊断参考列表中选择一个存在的ref，或将status设置为0表示没有合适的ref。", selection.RefName)
			messages = append(messages,
				schema.AssistantMessage(resp.Content, resp.ToolCalls),
				schema.UserMessage(feedbackMsg))
			continue
		} else {
			// 未找到合适的ref，将当前playbook加入排除列表
			state.ExcludedPlaybooks = append(state.ExcludedPlaybooks, state.SelectedPlaybook.Name)
			state.SelectedPlaybook = nil
			return state, nil
		}
	}

	return state, fmt.Errorf("选择ref失败: 已达到最大重试次数 %d", m.maxRetries)
}

// finishNode 完成节点
func (m *Matcher) finishNode(ctx context.Context, state *MatchState) (*MatchState, error) {
	return state, nil
}

// buildResult 构建最终结果
func (m *Matcher) buildResult(state *MatchState) *MatchResult {
	if state.SelectedPlaybook != nil && state.SelectedRef != nil {
		return &MatchResult{
			Playbook: state.SelectedPlaybook,
			Ref:      state.SelectedRef,
			Success:  true,
			Message:  "成功匹配到合适的诊断方案",
		}
	}

	return &MatchResult{
		Success: false,
		Message: "未找到合适的诊断方案",
	}
}

// buildPlaybooksDescription 构建playbooks的描述
func (m *Matcher) buildPlaybooksDescription(playbooks []playbook.Playbook) string {
	var sb strings.Builder
	for i, pb := range playbooks {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, pb.Name))
		sb.WriteString(fmt.Sprintf("   描述: %s\n", pb.Desc))
		if len(pb.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("   标签: %s\n", strings.Join(pb.Tags, ", ")))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// buildRefsDescription 构建refs的描述
func (m *Matcher) buildRefsDescription(refs []playbook.Ref) string {
	var sb strings.Builder
	for i, ref := range refs {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, ref.Name))
		sb.WriteString(fmt.Sprintf("   描述: %s\n", ref.Desc))
		if ref.Log != "" {
			sb.WriteString(fmt.Sprintf("   日志: %s\n", ref.Log))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
