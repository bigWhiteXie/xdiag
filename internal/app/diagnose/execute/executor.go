package execute

import (
	"context"
	"encoding/json"
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
	maxRetries = 3
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
	Type    string // step_start, step_complete, step_error, branch_select, complete
	Step    *playbook.Step
	Result  *StepResult
	Message string
	Error   string
}

// Executor Book执行器
type Executor struct {
	recAgent *adk.ChatModelAgent
	graph    compose.Runnable[*ExecuteState, *ExecuteState]
}

// NewExecutor 创建新的执行器
func NewExecutor(ctx context.Context) (*Executor, error) {
	targetRepo := svc.GetServiceContext().TargetsRepo
	// 创建 ReAct Agent
	// 创建结构化输出工具
	structuredOutputTool := itool.NewStructuredOutputTool(itool.StructuredOutputConfig{
		Description: "用于输出诊断步骤的执行结果",
		Fields: []itool.FieldDefinition{
			{
				Name:        "status",
				Type:        "number",
				Description: "步骤执行状态，0表示未完成，1表示已完成",
				Required:    true,
				Example:     1,
			},
			{
				Name:        "result",
				Type:        "string",
				Description: "该步骤执行中发现的信息、执行操作的结果等",
				Required:    true,
				Example:     "检查完成，发现端口8080正在监听",
			},
			{
				Name:        "selected_case",
				Type:        "number",
				Description: "对于branch类型步骤，选择的分支索引（从0开始），若没有符合条件的分支则设置为-1",
				Required:    false,
				Example:     0,
			},
		},
	})

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
					structuredOutputTool,
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("创建agent失败: %w", err)
	}

	executor := &Executor{
		recAgent: agent,
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
func (e *Executor) Execute(ctx context.Context, book *playbook.Book, target *targets.Target, question string) (chan ExecuteEvent, error) {
	// 创建事件通道
	eventChan := make(chan ExecuteEvent, 100)

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
		EventChan: eventChan,
	}

	// 在goroutine中执行graph
	go func() {
		defer close(eventChan)

		finalState, err := e.graph.Invoke(ctx, state)
		if err != nil {
			eventChan <- ExecuteEvent{
				Type:  "complete",
				Error: fmt.Sprintf("执行失败: %v", err),
			}
			return
		}

		// 发送完成事件
		eventChan <- ExecuteEvent{
			Type:    "complete",
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
		if event.Type == "complete" {
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
	err := graph.AddLambdaNode("execute_step", compose.InvokableLambda(e.executeStepNode))
	if err != nil {
		return nil, fmt.Errorf("添加execute_step节点失败: %w", err)
	}

	err = graph.AddLambdaNode("finish", compose.InvokableLambda(e.finishNode))
	if err != nil {
		return nil, fmt.Errorf("添加finish节点失败: %w", err)
	}

	// 设置入口
	graph.AddEdge(compose.START, "execute_step")

	// 添加分支逻辑
	err = graph.AddBranch("execute_step", compose.NewGraphBranch(func(ctx context.Context, state *ExecuteState) (string, error) {
		// 检查是否有错误且超过重试次数
		if state.Error != "" && state.RetryCount >= maxRetries {
			return "finish", nil
		}

		// 检查步骤栈是否为空
		if len(state.StepStack) == 0 {
			return "finish", nil
		}

		// 获取当前栈顶
		currentContext := &state.StepStack[len(state.StepStack)-1]

		// 检查当前层级的步骤是否都已完成
		if currentContext.CurrentIndex >= len(currentContext.Steps) {
			// 弹出当前层级
			state.StepStack = state.StepStack[:len(state.StepStack)-1]

			// 如果栈为空，说明所有步骤都已完成
			if len(state.StepStack) == 0 {
				return "finish", nil
			}

			// 继续执行上一层的下一个步骤
			return "execute_step", nil
		}

		// 继续执行下一步
		return "execute_step", nil
	}, map[string]bool{
		"execute_step": true,
		"finish":       true,
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
	if state.EventChan != nil {
		state.EventChan <- ExecuteEvent{
			Type:    "step_start",
			Step:    &currentStep,
			Message: fmt.Sprintf("开始执行步骤: %s", currentStep.Desc),
		}
	}

	// 根据步骤类型处理
	if currentStep.Kind == "branch" {
		return e.executeBranchStep(ctx, state, currentStep, currentContext)
	}

	return e.executeSeqStep(ctx, state, currentStep, currentContext)
}

// executeSeqStep 执行顺序步骤
func (e *Executor) executeSeqStep(ctx context.Context, state *ExecuteState, currentStep playbook.Step, currentContext *StepContext) (*ExecuteState, error) {
	// 渲染提示词模板
	prompt, err := e.renderPrompt(state, currentStep)
	if err != nil {
		state.Error = fmt.Sprintf("渲染提示词失败: %v", err)
		state.RetryCount++
		if state.EventChan != nil {
			state.EventChan <- ExecuteEvent{
				Type:  "step_error",
				Step:  &currentStep,
				Error: state.Error,
			}
		}
		return state, nil
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
			state.Error = fmt.Sprintf("agent执行错误: %v", event.Err)
			state.RetryCount++
			if state.EventChan != nil {
				state.EventChan <- ExecuteEvent{
					Type:  "step_error",
					Step:  &currentStep,
					Error: state.Error,
				}
			}
			return state, nil
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
		state.Error = "agent未返回响应"
		state.RetryCount++
		if state.EventChan != nil {
			state.EventChan <- ExecuteEvent{
				Type:  "step_error",
				Step:  &currentStep,
				Error: state.Error,
			}
		}
		return state, nil
	}

	// 解析结果 - 从工具调用中获取
	result, err := e.extractResultFromMessage(lastMessage)
	if err != nil {
		state.Error = fmt.Sprintf("解析结果失败: %v", err)
		state.RetryCount++
		if state.EventChan != nil {
			state.EventChan <- ExecuteEvent{
				Type:  "step_error",
				Step:  &currentStep,
				Error: state.Error,
			}
		}
		return state, nil
	}

	// 处理结果
	if result.Status == 1 {
		// 步骤完成
		state.ExecutedSteps = append(state.ExecutedSteps, ExecutedStep{
			Step:   currentStep,
			Result: *result,
		})
		currentContext.CurrentIndex++
		state.RetryCount = 0
		state.Error = ""
		state.CurrentContext = ""

		if state.EventChan != nil {
			state.EventChan <- ExecuteEvent{
				Type:   "step_complete",
				Step:   &currentStep,
				Result: result,
			}
		}
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
		if state.EventChan != nil {
			state.EventChan <- ExecuteEvent{
				Type:  "step_error",
				Step:  &currentStep,
				Error: state.Error,
			}
		}
		return state, nil
	}

	// 渲染分支选择提示词
	prompt, err := e.renderBranchPrompt(state, currentStep)
	if err != nil {
		state.Error = fmt.Sprintf("渲染分支提示词失败: %v", err)
		state.RetryCount++
		if state.EventChan != nil {
			state.EventChan <- ExecuteEvent{
				Type:  "step_error",
				Step:  &currentStep,
				Error: state.Error,
			}
		}
		return state, nil
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
			state.Error = fmt.Sprintf("agent执行错误: %v", event.Err)
			state.RetryCount++
			if state.EventChan != nil {
				state.EventChan <- ExecuteEvent{
					Type:  "step_error",
					Step:  &currentStep,
					Error: state.Error,
				}
			}
			return state, nil
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
		state.Error = "agent未返回响应"
		state.RetryCount++
		if state.EventChan != nil {
			state.EventChan <- ExecuteEvent{
				Type:  "step_error",
				Step:  &currentStep,
				Error: state.Error,
			}
		}
		return state, nil
	}

	// 解析结果 - 从工具调用中获取
	result, err := e.extractResultFromMessage(lastMessage)
	if err != nil {
		state.Error = fmt.Sprintf("解析结果失败: %v", err)
		state.RetryCount++
		if state.EventChan != nil {
			state.EventChan <- ExecuteEvent{
				Type:  "step_error",
				Step:  &currentStep,
				Error: state.Error,
			}
		}
		return state, nil
	}

	// 当所有分支都不满足条件时，跳到下一个步骤即可
	if result.SelectedCase < 0 || result.SelectedCase >= len(currentStep.Cases) {
		// 步骤完成
		state.ExecutedSteps = append(state.ExecutedSteps, ExecutedStep{
			Step:   currentStep,
			Result: *result,
		})
		currentContext.CurrentIndex++
		state.RetryCount = 0
		state.Error = ""
		state.CurrentContext = ""

		if state.EventChan != nil {
			state.EventChan <- ExecuteEvent{
				Type:   "step_complete",
				Step:   &currentStep,
				Result: result,
			}
		}
		return state, nil
	}

	// 记录分支选择
	selectedCase := currentStep.Cases[result.SelectedCase]
	state.ExecutedSteps = append(state.ExecutedSteps, ExecutedStep{
		Step:   currentStep,
		Result: *result,
	})

	if state.EventChan != nil {
		state.EventChan <- ExecuteEvent{
			Type:    "branch_select",
			Step:    &currentStep,
			Result:  result,
			Message: fmt.Sprintf("选择分支: %s", selectedCase.Case),
		}
	}

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

// extractResultFromMessage 从消息中提取结构化输出结果
func (e *Executor) extractResultFromMessage(msg *schema.Message) (*StepResult, error) {
	// 查找工具调用
	for _, toolCall := range msg.ToolCalls {
		if toolCall.Function.Name == itool.StructOutputToolName {
			// 解析工具调用的参数
			var toolOutput itool.StructuredOutputOutput
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &toolOutput); err != nil {
				continue
			}

			// 检查是否成功
			if toolOutput.Status != 1 {
				return nil, fmt.Errorf("工具调用失败: %s", toolOutput.Message)
			}

			// 转换为 StepResult
			result := &StepResult{}

			if status, ok := toolOutput.Data["status"].(float64); ok {
				result.Status = int(status)
			}

			if resultStr, ok := toolOutput.Data["result"].(string); ok {
				result.Result = resultStr
			}

			if selectedCase, ok := toolOutput.Data["selected_case"].(float64); ok {
				result.SelectedCase = int(selectedCase)
			}

			return result, nil
		}
	}

	return nil, fmt.Errorf("未找到工具调用结果")
}

// renderPrompt 渲染提示词模板（用于seq类型步骤）
func (e *Executor) renderPrompt(state *ExecuteState, currentStep playbook.Step) (string, error) {
	tmpl := `你是一个专业的系统诊断专家。请根据以下信息执行当前诊断步骤。

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

请执行当前步骤，并使用 output_result 工具输出结果。`

	t, err := template.New("prompt").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(tmpl)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"Target":         state.Target,
		"Question":       state.Question,
		"CurrentStep":    currentStep,
		"CurrentContext": state.CurrentContext,
		"ExecutedSteps":  state.ExecutedSteps,
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderBranchPrompt 渲染分支选择提示词模板
func (e *Executor) renderBranchPrompt(state *ExecuteState, currentStep playbook.Step) (string, error) {
	// 验证必要字段是否存在
	if state.Target == nil {
		return "", fmt.Errorf("target 不能为 nil")
	}

	tmpl := `你是一个专业的系统诊断专家。请根据以下信息选择合适的诊断分支。若没有分支符合条件，则将selected_case设置为-1

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

请根据当前情况和已执行步骤的结果，选择最合适的分支，并使用 output_result 工具输出结果。`

	t, err := template.New("branch_prompt").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(tmpl)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"Target":        state.Target,
		"Question":      state.Question,
		"CurrentStep":   currentStep,
		"ExecutedSteps": state.ExecutedSteps,
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

	sb.WriteString("# 诊断报告\n\n")

	sb.WriteString("## 基本信息\n\n")
	sb.WriteString(fmt.Sprintf("- **诊断方案**: %s\n", state.Book.Name))
	sb.WriteString(fmt.Sprintf("- **目标名称**: %s\n", state.Target.Name))
	sb.WriteString(fmt.Sprintf("- **目标类型**: %s\n", state.Target.Kind))
	sb.WriteString(fmt.Sprintf("- **目标地址**: %s:%d\n", state.Target.Address, state.Target.Port))
	sb.WriteString(fmt.Sprintf("- **用户问题**: %s\n\n", state.Question))

	if state.Error != "" {
		sb.WriteString("## 执行状态\n\n")
		sb.WriteString(fmt.Sprintf("**执行失败**: %s\n\n", state.Error))
	} else {
		sb.WriteString("## 执行状态\n\n")
		sb.WriteString("**执行成功**\n\n")
	}

	sb.WriteString("## 执行详情\n\n")
	for i, executed := range state.ExecutedSteps {
		sb.WriteString(fmt.Sprintf("### 步骤 %d: %s\n\n", i+1, executed.Step.Desc))
		sb.WriteString(fmt.Sprintf("- **类型**: %s\n", executed.Step.Kind))
		sb.WriteString(fmt.Sprintf("- **结果**: %s\n\n", executed.Result.Result))
	}

	sb.WriteString("## 总结\n\n")
	if state.Error != "" {
		sb.WriteString(fmt.Sprintf("诊断过程中遇到错误，已完成 %d/%d 个步骤。\n", len(state.ExecutedSteps), len(state.Book.Steps)))
	} else {
		sb.WriteString(fmt.Sprintf("诊断成功完成，共执行 %d 个步骤。\n", len(state.ExecutedSteps)))
	}

	return sb.String()
}

// getAgentInstruction 获取Agent指令
func getAgentInstruction() string {
	return `你是一个专业的系统诊断专家，能够根据给定的诊断步骤执行相应的操作。

# 工作方式
1. 仔细阅读当前需要执行的步骤描述
2. 根据步骤类型和描述，使用可用的工具执行相应的操作
3. 收集执行过程中的信息和结果
4. 使用 output_result 工具输出结构化结果

# 注意事项
- status为1表示步骤已完成，为0表示未完成
- result字段应包含详细的执行信息，包括发现的问题、执行的操作、获取的关键数据等
- 如果步骤未完成，在result中说明原因和已获取的信息
- 对于分支选择步骤，需要设置 selected_case 字段为选中的分支索引（从0开始），若没有符合条件的分支则设置为-1`
}
