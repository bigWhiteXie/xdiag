package execute

import (
	"context"
	"fmt"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/bigWhiteXie/xdiag/internal/app/targets"
	"github.com/bigWhiteXie/xdiag/internal/svc"
	itool "github.com/bigWhiteXie/xdiag/internal/tool"
	"github.com/bigWhiteXie/xdiag/pkg/formatter"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

const (
	eventChanBuffer = 100
)

// Graph node names
const (
	nodeExecuteStep = "execute_step"
	nodeFinish      = "finish"
)

// Executor Book执行器
type Executor struct {
	recAgent    *adk.ChatModelAgent
	graph       compose.Runnable[*ExecuteState, *ExecuteState]
	parser      *resultParser
	emitter     *eventEmitter
	reporter    *reportGenerator
	showDetails bool
}

// NewExecutor 创建新的执行器
func NewExecutor(ctx context.Context, showDetails bool) (*Executor, error) {
	// 创建 agent
	agent, err := createAgent(ctx)
	if err != nil {
		return nil, err
	}

	// 创建解析器
	parser := newResultParser()

	// 创建事件发射器
	emitter := newEventEmitter(showDetails)

	// 创建报告生成器
	reporter := newReportGenerator()

	executor := &Executor{
		recAgent:    agent,
		parser:      parser,
		emitter:     emitter,
		reporter:    reporter,
		showDetails: showDetails,
	}

	// 构建执行图
	graph, err := executor.buildGraph()
	if err != nil {
		return nil, fmt.Errorf("构建graph失败: %w", err)
	}
	executor.graph = graph

	return executor, nil
}

// Execute 执行Book
func (e *Executor) Execute(ctx context.Context, book *playbook.Book, target *targets.Target, question string) (chan ExecuteEvent, error) {
	eventChan := make(chan ExecuteEvent, eventChanBuffer)

	state := e.createInitialState(book, target, question, e.showDetails, eventChan)

	// 在goroutine中执行graph
	go func() {
		defer close(eventChan)

		finalState, err := e.graph.Invoke(ctx, state)
		if err != nil {
			e.emitter.send(eventChan, ExecuteEvent{
				Type:  EventTypeComplete,
				Error: fmt.Sprintf("执行失败: %v", err),
			})
			return
		}

		e.emitter.send(eventChan, ExecuteEvent{
			Type:    EventTypeComplete,
			Message: e.reporter.generate(finalState),
		})
	}()

	return eventChan, nil
}

// GetReport 消费事件通道并返回最终报告
func (e *Executor) GetReport(ch chan ExecuteEvent) string {
	eventFormatter := formatter.NewEventFormatter(e.showDetails)
	var lastMsg ExecuteEvent

	for event := range ch {
		lastMsg = event
		// if event.Type == EventTypeComplete {
		// 	break
		// }

		if e.showDetails {
			if output := eventFormatter.FormatEvent(
				event.Type,
				getStepDesc(event.Step),
				getResultDesc(event.Result),
				event.Message,
				event.Error,
			); output != "" {
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

	// 设置流转逻辑
	if err := e.setGraphFlow(graph); err != nil {
		return nil, err
	}

	// 编译graph
	compiled, err := graph.Compile(context.Background())
	if err != nil {
		return nil, fmt.Errorf("编译graph失败: %w", err)
	}

	return compiled, nil
}

// executeStepNode 执行步骤节点
func (e *Executor) executeStepNode(ctx context.Context, state *ExecuteState) (*ExecuteState, error) {
	// 检查步骤栈是否为空或当前层级已完成
	if !state.hasMoreSteps() {
		return state, nil
	}

	currentStep, currentContext := state.getCurrentStep()

	// 发送步骤开始事件
	e.emitter.sendStepStart(state.EventChan, currentStep)

	// 根据步骤类型执行
	if currentStep.Kind == StepKindBranch {
		return e.executeBranchStep(ctx, state, currentStep, currentContext)
	}
	return e.executeSeqStep(ctx, state, currentStep, currentContext)
}

// executeSeqStep 执行顺序步骤
func (e *Executor) executeSeqStep(ctx context.Context, state *ExecuteState, step playbook.Step, stepCtx *StepContext) (*ExecuteState, error) {
	prompt, err := renderSeqPrompt(state, step)
	if err != nil {
		return state.handleError(step, fmt.Sprintf("渲染提示词失败: %v", err)), nil
	}

	// 执行 agent 并收集输出
	outputs, err := e.runAgentCollecting(ctx, prompt, state.EventChan, step)
	if err != nil {
		return state.handleError(step, err.Error()), nil
	}

	// 解析结果
	result, err := e.parser.parseFromToolCall(outputs.lastMessage, seqResultToolName)
	if err != nil {
		// 如果有文本输出，将其添加到上下文并重试
		if outputs.textContent != "" {
			return state.handleTextOutput(step, stepCtx, outputs.textContent), nil
		}
		return state.handleError(step, fmt.Sprintf("解析结果失败: %v", err)), nil
	}

	// 处理结果
	return state.handleResult(step, stepCtx, result), nil
}

// executeBranchStep 执行分支步骤
func (e *Executor) executeBranchStep(ctx context.Context, state *ExecuteState, step playbook.Step, stepCtx *StepContext) (*ExecuteState, error) {
	if len(step.Cases) == 0 {
		return state.handleError(step, "branch步骤的cases不能为空"), nil
	}

	prompt, err := renderBranchPrompt(state, step)
	if err != nil {
		return state.handleError(step, fmt.Sprintf("渲染分支提示词失败: %v", err)), nil
	}

	// 执行 agent 并收集输出
	outputs, err := e.runAgentCollecting(ctx, prompt, state.EventChan, step)
	if err != nil {
		return state.handleError(step, err.Error()), nil
	}

	// 解析结果
	result, err := e.parser.parseFromToolCall(outputs.lastMessage, branchResultToolName)
	if err != nil {
		// 如果有文本输出，将其添加到上下文并重试
		if outputs.textContent != "" {
			return state.handleTextOutput(step, stepCtx, outputs.textContent), nil
		}
		return state.handleError(step, fmt.Sprintf("解析结果失败: %v", err)), nil
	}

	// 处理分支选择
	return state.handleBranchSelection(step, stepCtx, result), nil
}

// finishNode 完成节点
func (e *Executor) finishNode(ctx context.Context, state *ExecuteState) (*ExecuteState, error) {
	return state, nil
}

// runAgentCollecting 执行agent并收集所有输出
func (e *Executor) runAgentCollecting(ctx context.Context, prompt string, eventChan chan<- ExecuteEvent, step playbook.Step) (*agentOutputs, error) {
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           e.recAgent,
		EnableStreaming: false,
	})
	cctx, cancel := context.WithCancel(ctx)
	outputs := newAgentOutputs(e.showDetails)

	iter := runner.Query(cctx, prompt)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, fmt.Errorf("agent执行错误: %v", event.Err)
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}

			outputs.addMessage(msg, eventChan, step)
			expectToolName := seqResultToolName
			if step.Kind == StepKindBranch {
				expectToolName = branchResultToolName
			}
			if _, err := e.parser.parseFromToolCall(msg, expectToolName); err == nil {
				cancel()
				return outputs, nil
			}
		}
	}

	if outputs.lastMessage == nil && outputs.textContent == "" {
		return nil, fmt.Errorf("agent未返回响应")
	}

	return outputs, nil
}

// setGraphFlow 设置图的流转逻辑
func (e *Executor) setGraphFlow(graph *compose.Graph[*ExecuteState, *ExecuteState]) error {
	// 添加入口
	graph.AddEdge(compose.START, nodeExecuteStep)

	// 添加分支逻辑
	err := graph.AddBranch(nodeExecuteStep, compose.NewGraphBranch(func(ctx context.Context, state *ExecuteState) (string, error) {
		// 检查错误重试
		if state.shouldFinish() {
			return nodeFinish, nil
		}
		return nodeExecuteStep, nil
	}, map[string]bool{
		nodeExecuteStep: true,
		nodeFinish:      true,
	}))
	if err != nil {
		return fmt.Errorf("添加分支失败: %w", err)
	}

	// finish -> END
	graph.AddEdge(nodeFinish, compose.END)

	return nil
}

// createInitialState 创建初始状态
func (e *Executor) createInitialState(book *playbook.Book, target *targets.Target, question string, showDetails bool, eventChan chan<- ExecuteEvent) *ExecuteState {
	return &ExecuteState{
		Book:          book,
		Target:        target,
		Question:      question,
		ExecutedSteps: []ExecutedStep{},
		StepStack: []StepContext{
			{
				Steps:        book.Steps,
				CurrentIndex: 0,
			},
		},
		EventChan: eventChan,
	}
}

// createAgent 创建诊断agent
func createAgent(ctx context.Context) (*adk.ChatModelAgent, error) {
	targetRepo := svc.GetServiceContext().TargetsRepo

	seqResultTool := itool.NewStructuredOutputTool(itool.StructuredOutputConfig{
		Name:        seqResultToolName,
		Description: "输出顺序步骤的执行结果，必须使用此工具输出结果",
		WrapData:    false,
		Fields: []itool.FieldDefinition{
			{Name: "status", Type: "number", Description: "步骤执行状态，0表示未完成，1表示已完成", Required: true, Example: 1},
			{Name: "result", Type: "string", Description: "步骤执行中发现的信息、执行操作的结果等", Required: true, Example: "执行成功，未发现问题"},
		},
	})

	branchResultTool := itool.NewStructuredOutputTool(itool.StructuredOutputConfig{
		Name:        branchResultToolName,
		Description: "输出分支选择的结果，必须使用此工具输出结果",
		WrapData:    false,
		Fields: []itool.FieldDefinition{
			{Name: "status", Type: "number", Description: "步骤执行状态，0表示未完成，1表示已完成", Required: true, Example: 1},
			{Name: "result", Type: "string", Description: "选择该分支的原因和依据", Required: true, Example: "根据日志分析，问题出在数据库连接"},
			{Name: "selected_case", Type: "number", Description: "选中的分支索引（从0开始），若没有符合条件的分支则设置为-1", Required: true, Example: 0},
		},
	})

	resultTools := []tool.BaseTool{seqResultTool, branchResultTool}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "诊断执行器",
		Description: "一个能够执行诊断步骤的agent",
		Instruction: getAgentInstruction(),
		Model:       svc.GetServiceContext().Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: append([]tool.BaseTool{
					itool.NewExecTool(targetRepo),
					itool.NewCopyTool(targetRepo),
				}, resultTools...),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("创建agent失败: %w", err)
	}

	return agent, nil
}

// 辅助函数
func getStepDesc(step *playbook.Step) string {
	if step != nil {
		return step.Desc
	}
	return ""
}

func getResultDesc(result *StepResult) string {
	if result != nil {
		return result.Result
	}
	return ""
}
