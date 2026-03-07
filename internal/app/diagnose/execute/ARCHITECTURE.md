# Book Executor 架构说明

## 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                         Executor                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                    Graph Flow                          │  │
│  │                                                        │  │
│  │  START → execute_step ⟲ → finish → END               │  │
│  │              │                                         │  │
│  │              └─ 重试逻辑(最多3次)                      │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                  ReAct Agent                           │  │
│  │  - 接收渲染后的提示词                                  │  │
│  │  - 调用工具执行诊断步骤                                │  │
│  │  - 返回JSON格式的执行结果                              │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## 数据流

```
输入: Book + Target + Question
  │
  ├─> 初始化 ExecuteState
  │
  ├─> 进入 Graph 执行循环
  │    │
  │    ├─> execute_step 节点
  │    │    │
  │    │    ├─> 渲染提示词模板
  │    │    │    ├─ 当前环境 (Target信息)
  │    │    │    ├─ 用户问题
  │    │    │    ├─ 当前步骤
  │    │    │    ├─ 执行上下文 (如果有)
  │    │    │    └─ 已执行步骤和结果
  │    │    │
  │    │    ├─> ReAct Agent 执行
  │    │    │    └─> 返回 StepResult {status, result}
  │    │    │
  │    │    └─> 处理结果
  │    │         ├─ status=1: 步骤完成，移到下一步
  │    │         └─ status=0: 未完成，设置上下文重试
  │    │
  │    └─> 分支判断
  │         ├─ 所有步骤完成 → finish
  │         ├─ 重试次数超限 → finish
  │         └─ 继续执行 → execute_step
  │
  └─> 生成 Markdown 报告
       └─> 输出: ExecuteResult
```

## 状态转换

```
ExecuteState 状态变化:

初始状态:
  CurrentStepIndex = 0
  ExecutedSteps = []
  RetryCount = 0
  Error = ""

执行中:
  ┌─────────────────────────────────────┐
  │  步骤执行成功 (status=1)             │
  │  - ExecutedSteps += 当前步骤         │
  │  - CurrentStepIndex++               │
  │  - RetryCount = 0                   │
  │  - CurrentContext = ""              │
  └─────────────────────────────────────┘
           │
           ├─ 还有步骤 → 继续执行
           └─ 无步骤 → 完成

  ┌─────────────────────────────────────┐
  │  步骤未完成 (status=0)               │
  │  - CurrentContext = result          │
  │  - RetryCount++                     │
  └─────────────────────────────────────┘
           │
           ├─ RetryCount < 3 → 重试
           └─ RetryCount >= 3 → 失败

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

```
┌──────────────────────────────────────────┐
│ ## 当前环境                               │
│ - 目标名称: {{.Target.Name}}             │
│ - 目标类型: {{.Target.Kind}}             │
│ - 目标地址: {{.Target.Address}}:{{.Port}}│
│ - 目标标签: {{.Target.Tags}}             │
├──────────────────────────────────────────┤
│ ## 用户问题                               │
│ {{.Question}}                            │
├──────────────────────────────────────────┤
│ ## 当前需要执行的步骤                     │
│ - 类型: {{.CurrentStep.Kind}}           │
│ - 描述: {{.CurrentStep.Desc}}           │
├──────────────────────────────────────────┤
│ ## 执行上下文 (可选)                      │
│ {{.CurrentContext}}                      │
├──────────────────────────────────────────┤
│ ## 已执行的步骤和结果                     │
│ {{range .ExecutedSteps}}                 │
│ ### 步骤: {{.Step.Desc}}                 │
│ - 结果: {{.Result.Result}}               │
│ {{end}}                                  │
└──────────────────────────────────────────┘
```

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

## 未完成的步骤 (如果有)
- [步骤描述] (类型: [类型])
...

## 总结
[执行概况统计]
```

## 扩展点

### 1. 添加自定义工具

在 `NewExecutor` 中的 `ToolsConfig` 添加工具:

```go
ToolsConfig: adk.ToolsConfig{
    ToolsNodeConfig: compose.ToolsNodeConfig{
        Tools: []tool.BaseTool{
            // 添加你的工具
            NewSSHTool(),
            NewDatabaseQueryTool(),
            NewLogAnalysisTool(),
        },
    },
},
```

### 2. 支持 CaseBlock

修改 `executeStepNode` 处理带条件分支的步骤:

```go
if len(currentStep.Cases) > 0 {
    // 根据条件选择合适的case执行
    selectedCase := selectCase(state, currentStep.Cases)
    // 递归执行case中的steps
}
```

### 3. 流式执行

实现实时进度反馈:

```go
type ProgressCallback func(stepIndex int, stepDesc string, result StepResult)

func (e *Executor) ExecuteWithProgress(
    ctx context.Context,
    book *playbook.Book,
    target *targets.Target,
    question string,
    callback ProgressCallback,
) (*ExecuteResult, error)
```

### 4. 自定义报告格式

实现 `ReportGenerator` 接口:

```go
type ReportGenerator interface {
    Generate(state *ExecuteState) string
}

// 支持 Markdown, HTML, JSON 等格式
```
