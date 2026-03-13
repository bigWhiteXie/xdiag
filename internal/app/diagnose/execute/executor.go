package execute

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/bigWhiteXie/xdiag/internal/app/targets"
	"github.com/bigWhiteXie/xdiag/internal/svc"
	itool "github.com/bigWhiteXie/xdiag/internal/tool"
	"github.com/bigWhiteXie/xdiag/pkg/formatter"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	maxRetries       = 3
	eventChanBuffer  = 100
	thinkTagStart    = "<think>"
	thinkTagEnd      = "</think>"
	thinkTagEndLen   = 8
	statusIncomplete = 0
	statusComplete   = 1
	invalidCaseIndex = -1
)

// Event type constants
const (
	EventTypeStepStart       = "step_start"
	EventTypeStepComplete    = "step_complete"
	EventTypeStepError       = "step_error"
	EventTypeBranchSelect    = "branch_select"
	EventTypeComplete        = "complete"
	EventTypeAgentThinking   = "agent_thinking"
	EventTypeAgentToolCall   = "agent_tool_call"
	EventTypeAgentToolResult = "agent_tool_result"
)

// Step kind constants
const (
	StepKindBranch = "branch"
)

// Graph node names
const (
	nodeExecuteStep = "execute_step"
	nodeFinish      = "finish"
)

// ExecuteState 定义执行状态
type ExecuteState struct {
	Book             *playbook.Book
	Target           *targets.Target
	Question         string
	CurrentStepIndex int
	ExecutedSteps    []ExecutedStep // 记录执行过的步骤和结果
	CurrentContext   string         // 当前步骤的执行上下文
	RetryCount       int            // 记录重试次数
	Error            string
	StepStack        []StepContext // 步骤栈，用于处理嵌套的分支
	EventChan        chan<- ExecuteEvent
	ShowDetails      bool // 是否显示详细信息
}

// StepContext 步骤上下文，用于处理嵌套分支
type StepContext struct {
	Steps        []playbook.Step
	CurrentIndex int
}

// ExecutedStep 已执行的步骤
type ExecutedStep struct {
	Step   playbook.Step
	Result StepResult
}

// StepResult 步骤执行结果
type StepResult struct {
	Status       int    `json:"status"`        // 0: 未完成, 1: 已完成
	Result       string `json:"result"`        // 步骤执行中发现的信息、执行操作的结果等
	SelectedCase int    `json:"selected_case"` // 对于branch类型，选择的分支索引（从0开始）
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	Success   bool
	Report    string
	Error     string
	EventChan <-chan ExecuteEvent
}

// ExecuteEvent 执行事件
type ExecuteEvent struct {
	Type    string // step_start, step_complete, step_error, branch_select, complete, agent_thinking, agent_tool_call, agent_tool_result
	Step    *playbook.Step
	Result  *StepResult
	Message string
	Error   string
}

// Executor Book执行器
type Executor struct {
	recAgent     *adk.ChatModelAgent
	graph        compose.Runnable[*ExecuteState, *ExecuteState]
	seqParser    schema.MessageParser[StepResult] // 普通步骤结果解析器
	branchParser schema.MessageParser[StepResult] // 分支步骤结果解析器
}

// cleanJSONContent 清理消息内容，提取纯 JSON
func cleanJSONContent(content string) string {
	// 移除 <think> 标签及其内容
	for {
		startIdx := strings.Index(content, thinkTagStart)
		if startIdx == -1 {
			break
		}
		endIdx := strings.Index(content[startIdx:], thinkTagEnd)
		if endIdx == -1 {
			break
		}
		content = content[:startIdx] + content[startIdx+endIdx+thinkTagEndLen:]
	}

	// 查找第一个 { 和最后一个 }
	startIdx := strings.Index(content, "{")
	if startIdx == -1 {
		return content
	}

	endIdx := strings.LastIndex(content, "}")
	if endIdx == -1 || endIdx < startIdx {
		return content
	}

	return strings.TrimSpace(content[startIdx : endIdx+1])
}

// NewExecutor 创建新的执行器
func NewExecutor(ctx context.Context) (*Executor, error) {
	targetRepo := svc.GetServiceContext().TargetsRepo
	// 创建 ReAct Agent
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "诊断执行器",
		Description: "一个能够执行诊断步骤的agent",
		Instruction: getAgentInstruction(),
		Model:       svc.GetServiceContext().Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					itool.NewExecTool(targetRepo),
					itool.NewCopyTool(targetRepo),
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("创建agent失败: %w", err)
	}

	// 创建 JSON 解析器 - 从消息内容中解析
	seqParser := schema.NewMessageJSONParser[StepResult](&schema.MessageJSONParseConfig{
		ParseFrom: schema.MessageParseFromContent,
	})

	branchParser := schema.NewMessageJSONParser[StepResult](&schema.MessageJSONParseConfig{
		ParseFrom: schema.MessageParseFromContent,
	})

	executor := &Executor{
		recAgent:     agent,
		seqParser:    seqParser,
		branchParser: branchParser,
	}

	// 构建 Graph
	graph, err := executor.buildGraph()
	if err != nil {
		return nil, fmt.Errorf("构建graph失败: %w", err)
	}
	executor.graph = graph

	return executor, nil
}

// Execute 执行Book
func (e *Executor) Execute(ctx context.Context, book *playbook.Book, target *targets.Target, question string, showDetails bool) (chan ExecuteEvent, error) {
	// 创建事件通道
	eventChan := make(chan ExecuteEvent, eventChanBuffer)

	// 初始化状态
	state := &ExecuteState{
		Book:             book,
		Target:           target,
		Question:         question,
		CurrentStepIndex: 0,
		ExecutedSteps:    []ExecutedStep{},
		CurrentContext:   "",
		RetryCount:       0,
		Error:            "",
		StepStack: []StepContext{
			{
				Steps:        book.Steps,
				CurrentIndex: 0,
			},
		},
		EventChan:   eventChan,
		ShowDetails: showDetails,
	}

	// 在goroutine中执行graph
	go func() {
		defer close(eventChan)

		finalState, err := e.graph.Invoke(ctx, state)
		if err != nil {
			eventChan <- ExecuteEvent{
				Type:  EventTypeComplete,
				Error: fmt.Sprintf("执行失败: %v", err),
			}
			return
		}

		// 发送完成事件
		eventChan <- ExecuteEvent{
			Type:    EventTypeComplete,
			Message: e.generateReport(finalState),
		}
	}()

	return eventChan, nil
}

func (e *Executor) GetReport(ch chan ExecuteEvent, showDetails bool) string {
	eventFormatter := formatter.NewEventFormatter(showDetails)
	var lastMsg ExecuteEvent

	for event := range ch {
		lastMsg = event
		if event.Type == EventTypeComplete {
			break
		}

		// 使用格式化器输出事件
		if showDetails {
			stepDesc := ""
			result := ""
			if event.Step != nil {
				stepDesc = event.Step.Desc
			}
			if event.Result != nil {
				result = event.Result.Result
			}

			output := eventFormatter.FormatEvent(
				event.Type,
				stepDesc,
				result,
				event.Message,
				event.Error,
			)
			if output != "" {
				fmt.Print(output)
			}
		}
	}

	return lastMsg.Message
}

// buildGraph 构建执行图
func (e *Executor) buildGraph() (compose.Runnable[*ExecuteState, *ExecuteState], error) {
	graph := compose.NewGraph[*ExecuteState, *ExecuteState]()

	// 添加节点
	if err := graph.AddLambdaNode(nodeExecuteStep, compose.InvokableLambda(e.executeStepNode)); err != nil {
		return nil, fmt.Errorf("添加execute_step节点失败: %w", err)
	}

	if err := graph.AddLambdaNode(nodeFinish, compose.InvokableLambda(e.finishNode)); err != nil {
		return nil, fmt.Errorf("添加finish节点失败: %w", err)
	}

	// 设置入口
	graph.AddEdge(compose.START, nodeExecuteStep)

	// 添加分支逻辑
	err := graph.AddBranch(nodeExecuteStep, compose.NewGraphBranch(func(ctx context.Context, state *ExecuteState) (string, error) {
		// 检查是否有错误且超过重试次数
		if state.Error != "" && state.RetryCount >= maxRetries {
			return nodeFinish, nil
		}

		// 检查步骤栈是否为空
		if len(state.StepStack) == 0 {
			return nodeFinish, nil
		}

		// 获取当前栈顶
		currentContext := &state.StepStack[len(state.StepStack)-1]

		// 检查当前层级的步骤是否都已完成
		if currentContext.CurrentIndex >= len(currentContext.Steps) {
			// 弹出当前层级
			state.StepStack = state.StepStack[:len(state.StepStack)-1]

			// 如果栈为空，说明所有步骤都已完成
			if len(state.StepStack) == 0 {
				return nodeFinish, nil
			}

			// 继续执行上一层的下一个步骤
			return nodeExecuteStep, nil
		}

		// 继续执行下一步
		return nodeExecuteStep, nil
	}, map[string]bool{
		nodeExecuteStep: true,
		nodeFinish:      true,
	}))
	if err != nil {
		return nil, fmt.Errorf("添加分支失败: %w", err)
	}

	// finish -> END
	graph.AddEdge(nodeFinish, compose.END)

	// 编译graph
	compiled, err := graph.Compile(context.Background())
	if err != nil {
		return nil, fmt.Errorf("编译graph失败: %w", err)
	}

	return compiled, nil
}

// executeStepNode 执行步骤节点
func (e *Executor) executeStepNode(ctx context.Context, state *ExecuteState) (*ExecuteState, error) {
	// 检查步骤栈是否为空
	if len(state.StepStack) == 0 {
		return state, nil
	}

	// 获取当前栈顶
	currentContext := &state.StepStack[len(state.StepStack)-1]

	// 检查是否还有步骤需要执行
	if currentContext.CurrentIndex >= len(currentContext.Steps) {
		return state, nil
	}

	currentStep := currentContext.Steps[currentContext.CurrentIndex]

	// 发送步骤开始事件
	e.sendEvent(state, ExecuteEvent{
		Type:    EventTypeStepStart,
		Step:    &currentStep,
		Message: fmt.Sprintf("开始执行步骤: %s", currentStep.Desc),
	})

	// 根据步骤类型处理
	if currentStep.Kind == StepKindBranch {
		return e.executeBranchStep(ctx, state, currentStep, currentContext)
	}

	return e.executeSeqStep(ctx, state, currentStep, currentContext)
}

// sendEvent 发送事件（如果通道存在）
func (e *Executor) sendEvent(state *ExecuteState, event ExecuteEvent) {
	if state.EventChan != nil {
		state.EventChan <- event
	}
}

// executeSeqStep 执行顺序步骤
func (e *Executor) executeSeqStep(ctx context.Context, state *ExecuteState, currentStep playbook.Step, currentContext *StepContext) (*ExecuteState, error) {
	// 渲染提示词模板
	prompt, err := e.renderPrompt(state, currentStep)
	if err != nil {
		return e.handleStepError(state, currentStep, fmt.Sprintf("渲染提示词失败: %v", err))
	}

	// 使用 ReAct Agent 执行步骤
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           e.recAgent,
		EnableStreaming: false,
	})

	iter := runner.Query(ctx, prompt)

	var lastMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return e.handleStepError(state, currentStep, fmt.Sprintf("agent执行错误: %v", event.Err))
		}

		// 处理agent输出事件
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil {
				e.handleAgentMessage(state, currentStep, msg, &lastMessage)
			}
		}
	}

	// 解析最后一条消息的执行状态
	if lastMessage == nil {
		return e.handleStepError(state, currentStep, "agent未返回响应")
	}

	// 清理消息内容，移除 <think> 等标签
	lastMessage.Content = cleanJSONContent(lastMessage.Content)

	// 解析结果 - 使用 MessageJSONParser
	result, err := e.seqParser.Parse(ctx, lastMessage)
	if err != nil {
		return e.handleStepError(state, currentStep, fmt.Sprintf("解析结果失败: %v", err))
	}

	// 处理结果
	return e.handleStepResult(state, currentStep, currentContext, result)
}

// handleStepError 处理步骤错误
func (e *Executor) handleStepError(state *ExecuteState, currentStep playbook.Step, errMsg string) (*ExecuteState, error) {
	state.Error = errMsg
	state.RetryCount++
	e.sendEvent(state, ExecuteEvent{
		Type:  EventTypeStepError,
		Step:  &currentStep,
		Error: state.Error,
	})
	return state, nil
}

// handleAgentMessage 处理agent消息
func (e *Executor) handleAgentMessage(state *ExecuteState, currentStep playbook.Step, msg *schema.Message, lastMessage **schema.Message) {
	// 输出思考内容
	if msg.Role == schema.Assistant && msg.Content != "" {
		*lastMessage = msg
		e.sendEvent(state, ExecuteEvent{
			Type:    EventTypeAgentThinking,
			Step:    &currentStep,
			Message: msg.Content,
		})
	}

	// 输出工具调用
	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			e.sendEvent(state, ExecuteEvent{
				Type:    EventTypeAgentToolCall,
				Step:    &currentStep,
				Message: fmt.Sprintf("调用工具: %s\n参数: %s", toolCall.Function.Name, toolCall.Function.Arguments),
			})
		}
	}

	// 输出工具执行结果（当 Role 为 Tool 时）
	if msg.Role == schema.Tool && msg.Content != "" {
		e.sendEvent(state, ExecuteEvent{
			Type:    EventTypeAgentToolResult,
			Step:    &currentStep,
			Message: fmt.Sprintf("工具结果: %s", msg.Content),
		})
	}
}

// handleStepResult 处理步骤结果
func (e *Executor) handleStepResult(state *ExecuteState, currentStep playbook.Step, currentContext *StepContext, result StepResult) (*ExecuteState, error) {
	if result.Status == statusComplete {
		// 步骤完成
		state.ExecutedSteps = append(state.ExecutedSteps, ExecutedStep{
			Step:   currentStep,
			Result: result,
		})
		currentContext.CurrentIndex++
		state.RetryCount = 0
		state.Error = ""
		state.CurrentContext = ""

		e.sendEvent(state, ExecuteEvent{
			Type:   EventTypeStepComplete,
			Step:   &currentStep,
			Result: &result,
		})
	} else {
		// 步骤未完成，设置上下文并重试
		state.CurrentContext = result.Result
		state.RetryCount++
	}

	return state, nil
}

// executeBranchStep 执行分支步骤
func (e *Executor) executeBranchStep(ctx context.Context, state *ExecuteState, currentStep playbook.Step, currentContext *StepContext) (*ExecuteState, error) {
	if len(currentStep.Cases) == 0 {
		state.Error = "branch步骤的cases不能为空"
		e.sendEvent(state, ExecuteEvent{
			Type:  EventTypeStepError,
			Step:  &currentStep,
			Error: state.Error,
		})
		return state, nil
	}

	// 渲染分支选择提示词
	prompt, err := e.renderBranchPrompt(state, currentStep)
	if err != nil {
		return e.handleStepError(state, currentStep, fmt.Sprintf("渲染分支提示词失败: %v", err))
	}

	// 使用 ReAct Agent 选择分支
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           e.recAgent,
		EnableStreaming: false,
	})

	iter := runner.Query(ctx, prompt)

	var lastMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return e.handleStepError(state, currentStep, fmt.Sprintf("agent执行错误: %v", event.Err))
		}

		// 如果开启详细模式，输出agent的思考和工具调用过程
		if state.ShowDetails && event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil {
				e.handleAgentMessage(state, currentStep, msg, &lastMessage)
			}
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			if msg.Role == schema.Assistant {
				lastMessage = msg
			}
		}
	}

	if lastMessage == nil {
		return e.handleStepError(state, currentStep, "agent未返回响应")
	}

	// 清理消息内容，移除 <think> 等标签
	lastMessage.Content = cleanJSONContent(lastMessage.Content)

	// 解析结果 - 使用 MessageJSONParser
	result, err := e.branchParser.Parse(ctx, lastMessage)
	if err != nil {
		return e.handleStepError(state, currentStep, fmt.Sprintf("解析结果失败: %v", err))
	}

	// 当所有分支都不满足条件时，跳到下一个步骤即可
	if result.SelectedCase < 0 || result.SelectedCase >= len(currentStep.Cases) {
		return e.handleStepResult(state, currentStep, currentContext, result)
	}

	// 记录分支选择
	selectedCase := currentStep.Cases[result.SelectedCase]
	state.ExecutedSteps = append(state.ExecutedSteps, ExecutedStep{
		Step:   currentStep,
		Result: result,
	})

	e.sendEvent(state, ExecuteEvent{
		Type:    EventTypeBranchSelect,
		Step:    &currentStep,
		Result:  &result,
		Message: fmt.Sprintf("选择分支: %s", selectedCase.Case),
	})

	// 移动到当前层级的下一个步骤
	currentContext.CurrentIndex++

	// 将选中的分支步骤压入栈
	if len(selectedCase.Steps) > 0 {
		state.StepStack = append(state.StepStack, StepContext{
			Steps:        selectedCase.Steps,
			CurrentIndex: 0,
		})
	}

	state.RetryCount = 0
	state.Error = ""
	state.CurrentContext = ""

	return state, nil
}

// finishNode 完成节点
func (e *Executor) finishNode(ctx context.Context, state *ExecuteState) (*ExecuteState, error) {
	return state, nil
}

// renderPrompt 渲染提示词模板（用于seq类型步骤）
func (e *Executor) renderPrompt(state *ExecuteState, currentStep playbook.Step) (string, error) {
	const tmpl = `你是一个专业的系统诊断专家。请根据以下信息执行当前诊断步骤。

## 当前环境
- 目标名称: {{.Target.Name}}
- 目标类型: {{.Target.Kind}}
- 目标地址: {{.Target.Address}}:{{.Target.Port}}
- 目标标签: {{.Target.Tags}}

## 用户问题
{{.Question}}

## 当前需要执行的步骤
- 类型: {{.CurrentStep.Kind}}
- 描述: {{.CurrentStep.Desc}}
{{if .CurrentContext}}
## 执行上下文
{{.CurrentContext}}
{{end}}
{{if .ExecutedSteps}}
## 已执行的步骤和结果
{{range $index, $step := .ExecutedSteps}}
### 步骤 {{add $index 1}}: {{$step.Step.Desc}}
- 类型: {{$step.Step.Kind}}
- 结果: {{$step.Result.Result}}
{{end}}
{{end}}

请执行当前步骤，并将结果以 JSON 格式直接输出（不需要 <output> 标签）。

JSON 格式要求：
{
  "status": 1,
  "result": "步骤执行中发现的信息、执行操作的结果等"
}

注意：
- status: 0表示未完成，1表示已完成
- result: 必须包含步骤执行的详细信息`

	data := map[string]any{
		"Target":         state.Target,
		"Question":       state.Question,
		"CurrentStep":    currentStep,
		"CurrentContext": state.CurrentContext,
		"ExecutedSteps":  state.ExecutedSteps,
	}

	return e.executeTemplate("prompt", tmpl, data)
}

// renderBranchPrompt 渲染分支选择提示词模板
func (e *Executor) renderBranchPrompt(state *ExecuteState, currentStep playbook.Step) (string, error) {
	// 验证必要字段是否存在
	if state.Target == nil {
		return "", fmt.Errorf("target 不能为 nil")
	}

	const tmpl = `你是一个专业的系统诊断专家。请根据以下信息选择合适的诊断分支。若没有分支符合条件，则将selected_case设置为-1

## 当前环境
{{with .Target}}
- 目标名称: {{.Name}}
- 目标类型: {{.Kind}}
- 目标地址: {{.Address}}:{{.Port}}
- 目标标签: {{.Tags}}
{{end}}

## 用户问题
{{.Question}}

## 当前分支步骤
- 描述: {{.CurrentStep.Desc}}

## 可选分支
{{range $index, $case := .CurrentStep.Cases}}
### 分支 {{$index}}: {{$case.Case}}
{{end}}
{{if .ExecutedSteps}}
## 已执行的步骤和结果
{{range $index, $step := .ExecutedSteps}}
### 步骤 {{add $index 1}}: {{$step.Step.Desc}}
- 类型: {{$step.Step.Kind}}
- 结果: {{$step.Result.Result}}
{{end}}
{{end}}

请根据当前情况和已执行步骤的结果，选择最合适的分支，并将结果以 JSON 格式直接输出。

JSON 格式要求：
{
  "status": 1,
  "result": "分支选择的原因和依据",
  "selected_case": 0
}

注意：
- status: 0表示未完成，1表示已完成
- result: 必须说明选择该分支的原因
- selected_case: 选中的分支索引（从0开始），若没有符合条件的分支则设置为-1`

	data := map[string]any{
		"Target":        state.Target,
		"Question":      state.Question,
		"CurrentStep":   currentStep,
		"ExecutedSteps": state.ExecutedSteps,
	}

	return e.executeTemplate("branch_prompt", tmpl, data)
}

// executeTemplate 执行模板渲染
func (e *Executor) executeTemplate(name, tmpl string, data map[string]any) (string, error) {
	t, err := template.New(name).Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateReport 生成诊断报告
func (e *Executor) generateReport(state *ExecuteState) string {
	var sb strings.Builder

	// 基本信息
	sb.WriteString("# 诊断报告\n\n## 基本信息\n\n")
	e.writeBasicInfo(&sb, state)

	// 执行状态
	sb.WriteString("\n## 执行状态\n\n")
	e.writeExecutionStatus(&sb, state)

	// 执行详情
	sb.WriteString("\n## 执行详情\n\n")
	e.writeExecutionDetails(&sb, state)

	// 总结
	sb.WriteString("\n## 总结\n\n")
	e.writeSummary(&sb, state)

	return sb.String()
}

// writeBasicInfo 写入基本信息
func (e *Executor) writeBasicInfo(sb *strings.Builder, state *ExecuteState) {
	fmt.Fprintf(sb, "- **诊断方案**: %s\n", state.Book.Name)
	fmt.Fprintf(sb, "- **目标名称**: %s\n", state.Target.Name)
	fmt.Fprintf(sb, "- **目标类型**: %s\n", state.Target.Kind)
	fmt.Fprintf(sb, "- **目标地址**: %s:%d\n", state.Target.Address, state.Target.Port)
	fmt.Fprintf(sb, "- **用户问题**: %s\n", state.Question)
}

// writeExecutionStatus 写入执行状态
func (e *Executor) writeExecutionStatus(sb *strings.Builder, state *ExecuteState) {
	if state.Error != "" {
		fmt.Fprintf(sb, "**执行失败**: %s\n", state.Error)
	} else {
		sb.WriteString("**执行成功**\n")
	}
}

// writeExecutionDetails 写入执行详情
func (e *Executor) writeExecutionDetails(sb *strings.Builder, state *ExecuteState) {
	for i, executed := range state.ExecutedSteps {
		fmt.Fprintf(sb, "### 步骤 %d: %s\n\n", i+1, executed.Step.Desc)
		fmt.Fprintf(sb, "- **类型**: %s\n", executed.Step.Kind)
		fmt.Fprintf(sb, "- **结果**: %s\n\n", executed.Result.Result)
	}
}

// writeSummary 写入总结
func (e *Executor) writeSummary(sb *strings.Builder, state *ExecuteState) {
	if state.Error != "" {
		fmt.Fprintf(sb, "诊断过程中遇到错误，已完成 %d/%d 个步骤。\n", len(state.ExecutedSteps), len(state.Book.Steps))
	} else {
		fmt.Fprintf(sb, "诊断成功完成，共执行 %d 个步骤。\n", len(state.ExecutedSteps))
	}
}

// getAgentInstruction 获取Agent指令
func getAgentInstruction() string {
	return `你是一个专业的系统诊断专家，能够根据给定的诊断步骤执行相应的操作。

# 工作方式
1. 仔细阅读当前需要执行的步骤描述
2. 根据步骤类型和描述，使用可用的工具执行相应的操作
3. 收集执行过程中的信息和结果
4. 将结果以 JSON 格式直接输出

# 输出格式
**重要：直接输出 JSON，不要添加任何思考过程、标签或其他文本。**

对于普通步骤，直接输出：
{
  "status": 1,
  "result": "步骤执行的详细结果"
}

对于分支选择步骤，直接输出：
{
  "status": 1,
  "result": "分支选择的原因",
  "selected_case": 0
}

# 注意事项
- 不要使用 <think>、<output> 或任何其他标签
- 不要在 JSON 前后添加任何说明文字
- 直接以 { 开始，以 } 结束
- status为1表示步骤已完成，为0表示未完成
- result字段应包含详细的执行信息，包括发现的问题、执行的操作、获取的关键数据等
- 对于分支选择步骤，需要设置 selected_case 字段为选中的分支索引（从0开始），若没有符合条件的分支则设置为-1
- 必须确保输出的 JSON 格式正确，可以被正常解析`
}
