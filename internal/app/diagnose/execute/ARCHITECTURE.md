# Book Executor 架构说明

## 整体架构

```
┌──────────────────────────────────────────────────────────────┐
│                          Executor                              │
│                                                                │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                   Graph Flow (Eino)                    │  │
│  │                                                          │  │
│  │   START → execute_step ⟲ → finish → END                │  │
│  │              │                                          │  │
│  │              └─ 重试逻辑(最多3次)                          │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                                │
│  ┌────────────────────────────────────────────────────────┐  │
│  │              ChatModel Agent (ReAct)                    │  │
│  │  - 接收渲染后的提示词                                     │  │
│  │  - 调用工具执行诊断步骤                                   │  │
│  │  - 通过结构化输出工具返回结果                               │  │
│  │                                                          │  │
│  │  可用工具:                                                │  │
│  │  - exec: 执行命令                                         │  │
│  │  - copy: 复制文件                                         │  │
│  │  - output_seq_result: 输出顺序步骤结果                     │  │
│  │  - output_branch_result: 输出分支选择结果                 │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                                │
│  ┌────────────────────────────────────────────────────────┐  │
│  │              支持的步骤类型                               │  │
│  │  - seq: 顺序执行步骤                                      │  │
│  │  - branch: 条件分支选择                                   │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. executor.go
**主执行器**
- `Executor`: 执行器主结构
- `Execute()`: 执行诊断流程，返回事件通道
- `GetReport()`: 消费事件通道并生成报告
- `executeStepNode`: 执行步骤节点
- `executeSeqStep`: 执行顺序步骤
- `executeBranchStep`: 执行分支步骤
- `runAgentCollecting`: 运行Agent并收集输出

### 2. types.go
**数据类型定义**
- `ExecuteState`: 执行状态（包含步骤栈）
- `StepContext`: 步骤上下文（支持嵌套）
- `ExecutedStep`: 已执行的步骤记录
- `StepResult`: 步骤执行结果
- `ExecuteEvent`: 执行事件
- `ExecuteResult`: 最终执行结果

### 3. event_emitter.go
**事件发射器**
负责发送以下事件:
- `step_start`: 步骤开始
- `step_complete`: 步骤完成
- `step_error`: 步骤错误
- `branch_select`: 分支选择
- `complete`: 执行完成
- `agent_thinking`: Agent思考
- `agent_tool_call`: Agent工具调用
- `agent_tool_result`: Agent工具结果

### 4. prompt_renderer.go
**提示词渲染器**
- `renderSeqPrompt`: 渲染顺序步骤提示词
- `renderBranchPrompt`: 渲染分支选择提示词

### 5. prompts.go
**提示词模板**
- `agentInstruction`: Agent系统指令
- `seqPromptTemplate`: 顺序步骤提示词模板
- `branchPromptTemplate`: 分支选择提示词模板

### 6. result_parser.go
**结果解析器**
- `parseFromToolCall`: 从工具调用解析结果（首选）
- `parseFromContent`: 从消息内容解析结果（fallback）
- `cleanJSONContent`: 清理JSON内容（移除think标签等）

### 7. agent_outputs.go
**Agent输出收集器**
- `addMessage`: 添加消息并发送相应事件
- `lastMessage`: 保留最后一条消息用于解析

### 8. state_methods.go
**状态管理方法**
- `hasMoreSteps`: 检查是否还有步骤需要执行
- `getCurrentStep`: 获取当前步骤和上下文
- `shouldFinish`: 检查是否应该结束执行
- `handleError`: 处理步骤错误
- `handleResult`: 处理步骤结果
- `handleBranchSelection`: 处理分支选择

### 9. report_generator.go
**报告生成器**
- `generate`: 生成Markdown诊断报告
- `writeBasicInfo`: 写入基本信息
- `writeExecutionStatus`: 写入执行状态
- `writeExecutionDetails`: 写入执行详情
- `writeSummary`: 写入总结

## 数据流

```
输入: Book + Target + Question
  │
  ├─> 创建 ExecuteState
  │    - 初始化 StepStack（包含Book的Steps）
  │    - 创建事件通道 (EventChan)
  │
  ├─> 进入 Graph 执行循环
  │    │
  │    ├─> execute_step 节点
  │    │    │
  │    │    ├─> 检查是否还有步骤 (hasMoreSteps)
  │    │    │
  │    │    ├─> 获取当前步骤 (getCurrentStep)
  │    │    │
  │    │    ├─> 发送 step_start 事件
  │    │    │
  │    │    ├─> 根据步骤类型执行
  │    │    │    │
  │    │    │    ├─> seq 类型 → executeSeqStep
  │    │    │    │    │
  │    │    │    │    ├─> 渲染提示词 (renderSeqPrompt)
  │    │    │    │    │    ├─ 当前环境 (Target信息)
  │    │    │    │    │    ├─ 用户问题
  │    │    │    │    │    ├─ 当前步骤
  │    │    │    │    │    ├─ 执行上下文 (如果有)
  │    │    │    │    │    └─ 已执行步骤和结果
  │    │    │    │    │
  │    │    │    │    ├─> Agent 执行 (runAgentCollecting)
  │    │    │    │    │    ├─ 调用工具 (exec, copy)
  │    │    │    │    │    ├─ 收集 agent_thinking 事件
  │    │    │    │    │    ├─ 收集 agent_tool_call 事件
  │    │    │    │    │    ├─ 收集 agent_tool_result 事件
  │    │    │    │    │    └─ 输出通过 output_seq_result 工具
  │    │    │    │    │
  │    │    │    │    ├─> 解析结果 (parseFromToolCall)
  │    │    │    │    │    ├─ 从工具调用解析 (首选)
  │    │    │    │    │    └─ 从内容解析 (fallback)
  │    │    │    │    │
  │    │    │    │    └─> 处理结果 (handleResult)
  │    │    │    │         ├─ status=1: 步骤完成
  │    │    │    │         │   - 记录到 ExecutedSteps
  │    │    │    │         │   - 当前索引++
  │    │    │    │         │   - 发送 step_complete 事件
  │    │    │    │         └─ status=0: 未完成
  │    │    │    │             - 设置 CurrentContext
  │    │    │    │             - 重试计数++
  │    │    │    │
  │    │    │    └─> branch 类型 → executeBranchStep
  │    │    │         │
  │    │    │         ├─> 渲染提示词 (renderBranchPrompt)
  │    │    │         │
  │    │    │         ├─> Agent 执行 (runAgentCollecting)
  │    │    │         │    └─ 输出通过 output_branch_result 工具
  │    │    │         │
  │    │    │         ├─> 解析结果 (parseFromToolCall)
  │    │    │         │
  │    │    │         └─> 处理分支选择 (handleBranchSelection)
  │    │    │              ├─ selected_case >= 0
  │    │    │              │   - 发送 branch_select 事件
  │    │    │              │   - 记录到 ExecutedSteps
  │    │    │              │   - 当前索引++
  │    │    │              │   - 将选中分支的Steps压入 StepStack
  │    │    │              └─ selected_case < 0
  │    │    │                  - 跳过该步骤
  │    │    │
  │    │    └─> 分支判断
  │    │         ├─ shouldFinish() → finish
  │    │         └─ 继续执行 → execute_step
  │
  ├─> finish 节点
  │    └─> 返回最终状态
  │
  └─> 生成 Markdown 报告
       └─> 发送 complete 事件
```

## 状态转换

```
ExecuteState 使用 StepStack 支持嵌套的步骤执行:

StepStack 结构:
  [
    { Steps: [Book的Steps], CurrentIndex: 0 },  // 主步骤
    { Steps: [分支的Steps], CurrentIndex: 0 },  // 嵌套分支步骤
    ...
  ]

初始状态:
  StepStack = [{ Steps: Book.Steps, CurrentIndex: 0 }]
  ExecutedSteps = []
  RetryCount = 0
  Error = ""

执行中:
  ┌─────────────────────────────────────┐
  │  步骤执行成功 (status=1)             │
  │  - ExecutedSteps += 当前步骤         │
  │  - CurrentIndex++                   │
  │  - RetryCount = 0                   │
  │  - CurrentContext = ""              │
  └─────────────────────────────────────┘
           │
           ├─ 当前层级还有步骤 → 继续执行
           └─ 当前层级完成 → 弹出栈顶
               ├─ 栈为空 → 完成
               └─ 栈不为空 → 继续执行上一层

  ┌─────────────────────────────────────┐
  │  步骤未完成 (status=0)               │
  │  - CurrentContext = result          │
  │  - RetryCount++                     │
  └─────────────────────────────────────┘
           │
           ├─ RetryCount < 3 → 重试
           └─ RetryCount >= 3 → 失败

  ┌─────────────────────────────────────┐
  │  分支选择 (selected_case >= 0)      │
  │  - ExecutedSteps += 分支选择记录    │
  │  - CurrentIndex++                   │
  │  - 将选中分支的Steps压入StepStack   │
  └─────────────────────────────────────┘

  ┌─────────────────────────────────────┐
  │  无分支匹配 (selected_case < 0)     │
  │  - 跳过该步骤Branch                 │
  │  - CurrentIndex++                   │
  └─────────────────────────────────────┘

  ┌─────────────────────────────────────┐
  │  执行出错                            │
  │  - Error = 错误信息                  │
  │  - RetryCount++                     │
  └─────────────────────────────────────┘
           │
           ├─ RetryCount < 3 → 重试
           └─ RetryCount >= 3 → 失败
```

## 提示词模板结构

### 顺序步骤提示词 (seqPromptTemplate)

```
你是一个专业的系统诊断专家。请根据以下信息执行当前诊断步骤。

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

请执行当前步骤，必须使用 `output_seq_result` 工具输出结果。

可用的工具：
- output_seq_result: 输出顺序步骤的执行结果
  - status (number, 必填): 0表示未完成，1表示已完成
  - result (string, 必填): 步骤执行的详细结果
- exec: 执行命令
- copy: 复制文件

重要提示：
1. 必须使用 output_seq_result 工具输出结果
2. 不要直接输出JSON文本
3. 确保status字段正确设置
4. result字段应包含详细的执行信息
```

### 分支选择提示词 (branchPromptTemplate)

```
你是一个专业的系统诊断专家。请根据以下信息选择合适的诊断分支。

## 当前环境
- 目标名称: {{.Target.Name}}
- 目标类型: {{.Target.Kind}}
- 目标地址: {{.Target.Address}}:{{.Target.Port}}
- 目标标签: {{.Target.Tags}}

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

请根据当前情况和已执行步骤的结果，选择最合适的分支，必须使用 `output_branch_result` 工具输出结果。

可用的工具：
- output_branch_result: 输出分支选择的结果
  - status (number, 必填): 0表示未完成，1表示已完成
  - result (string, 必填): 选择该分支的原因和依据
  - selected_case (number, 必填): 选中的分支索引（从0开始），若没有符合条件的分支则设置为-1

重要提示：
1. 必须使用 output_branch_result 工具输出结果
2. 不要直接输出JSON文本
3. selected_case必须是有效的分支索引或-1
4. result字段应说明选择该分支的原因
```

## 事件类型

| 事件类型 | 说明 | 数据 |
|---------|------|------|
| `step_start` | 步骤开始执行 | `Step` |
| `step_complete` | 步骤执行完成 | `Step`, `Result` |
| `step_error` | 步骤执行错误 | `Step`, `Error` |
| `branch_select` | 分支被选中 | `Step`, `Result`, `Message` |
| `agent_thinking` | Agent思考内容 | `Step`, `Message` |
| `agent_tool_call` | Agent调用工具 | `Step`, `Message` |
| `agent_tool_result` | 工具执行结果 | `Step`, `Message` |
| `complete` | 执行完成 | `Message` (报告), `Error` |

## 报告格式

```markdown
# 诊断报告

## 基本信息
- **诊断方案**: [Book名称]
- **目标名称**: [Target名称]
- **目标类型**: [Target类型]
- **目标地址**: [地址:端口]
- **用户问题**: [问题描述]

## 执行状态
**执行成功** / **执行失败**: [错误信息]

## 执行详情
### 步骤 1: [步骤描述]
- **类型**: [步骤类型]
- **结果**: [执行结果]

### 步骤 2: [步骤描述]
...

## 总结
- 成功: 共执行 [N] 个步骤
- 失败: 已完成 [N]/[M] 个步骤
```

## 扩展点

### 1. 添加自定义工具

在 `createAgent` 中的 `ToolsConfig` 添加工具:

```go
resultTools := []tool.BaseTool{seqResultTool, branchResultTool}

agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeariConfig{
            Tools: append([]tool.BaseTool{
                itool.NewExecTool(targetRepo),
                itool.NewCopyTool(targetRepo),
                // 添加你的工具
                NewSSHTool(),
                NewDatabaseQueryTool(),
            }, resultTools...),
        },
    },
})
```

### 2. 自定义报告格式

修改 `reportGenerator` 的 `generate` 方法或实现新的报告生成器:

```go
type HTMLReportGenerator struct{}

func (r *HTMLReportGenerator) generate(state *ExecuteState) string {
    // 生成HTML格式报告
}
```

### 3. 添加新的步骤类型

1. 在 `executor.go` 中添加新的执行方法
2. 在 `prompts.go` 中添加对应的提示词模板
3. 在 `prompt_renderer.go` 中添加渲染函数
4. 在 `createAgent` 中添加对应的输出工具

示例:
```go
// executeLoopStep 执行循环步骤
func (e *Executor) executeLoopStep(ctx context.Context, state *ExecuteState, step playbook.Step, stepCtx *StepContext) (*ExecuteState, error) {
    // 实现循环逻辑
}
```

### 4. 事件处理

监听事件通道实现自定义处理:

```go
eventChan, err := executor.Execute(ctx, book, target, question, true)
if err != nil {
    return err
}

go func() {
    for event := range eventChan {
        switch event.Type {
        case EventTypeStepStart:
            log.Printf("步骤开始: %s", event.Step.Desc)
        case EventTypeAgentToolCall:
            // 记录工具调用
        // ... 其他事件处理
        }
    }
}()
```

## 执行示例

```go
// 创建执行器
executor, err := NewExecutor(ctx)
if err != nil {
    return err
}

// 执行诊断
eventChan, err := executor.Execute(ctx, book, target, "检查服务状态", true)
if err != nil {
    return err
}

// 获取报告
report := executor.GetReport(eventChan, true)
fmt.Println(report)
```
