# LLM JSON输出优化计划

## 痛点总结

1. **小模型JSON输出不稳定** - 小模型难以严格遵循JSON格式要求
2. **缺乏强制JSON输出的机制** - 仅依赖提示词约束，不可靠
3. **cleanJSONContent局限性** - 只处理``标签，不处理markdown代码块等
4. **错误处理不完善** - 重试策略简单，无错误反馈
5. **StructuredOutputTool未充分利用** - 工具调用方式更可靠但未用于结果输出

## StructuredOutputTool优化

已对 `internal/tool/structured_output.go` 进行以下优化：

1. **支持自定义工具名称** - 添加 `Name` 字段到 `StructuredOutputConfig`
2. **支持直接输出模式** - 添加 `WrapData` 字段，设置为 `false` 时直接返回原始数据而非包装在 `data` 字段中

**优化后的使用方式**：

```go
// 创建顺序步骤结果工具（直接返回原始数据）
seqResultTool := tool.NewStructuredOutputTool(tool.StructuredOutputConfig{
    Name:        "output_seq_result",
    Description: "输出顺序步骤的执行结果",
    WrapData:    false, // 直接返回原始数据，不包装
    Fields: []tool.FieldDefinition{
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
            Description: "步骤执行中发现的信息、执行操作的结果等",
            Required:    true,
            Example:     "执行成功，未发现问题",
        },
    },
})

// 创建分支选择结果工具（直接返回原始数据）
branchResultTool := tool.NewStructuredOutputTool(tool.StructuredOutputConfig{
    Name:        "output_branch_result",
    Description: "输出分支选择的结果",
    WrapData:    false, // 直接返回原始数据，不包装
    Fields: []tool.FieldDefinition{
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
            Description: "选择该分支的原因和依据",
            Required:    true,
            Example:     "根据日志分析，问题出在数据库连接",
        },
        {
            Name:        "selected_case",
            Type:        "number",
            Description: "选中的分支索引（从0开始），若没有符合条件的分支则设置为-1",
            Required:    true,
            Example:     0,
        },
    },
})
```

## 优化方案

### 方案一：使用Tool调用方式（推荐）

**优势**：
- 通过Tool Schema强制约束输出格式
- 小模型对Tool调用支持较好
- eino原生支持，无需额外处理
- 更易调试和错误追踪
- **直接利用优化后的StructuredOutputTool，无需创建新工具**

**实现步骤**：

1. **修改Agent配置** (Executor.NewExecutor)
   - 使用优化后的 `StructuredOutputTool` 创建两个专用工具：
     - `output_seq_result` - 用于顺序步骤结果输出
     - `output_branch_result` - 用于分支选择结果输出
   - 将这两个工具添加到Agent的工具列表中

2. **修改executeSeqStep**
   - 更新提示词，明确要求使用 `output_seq_result` 工具输出结果
   - 从ToolCall中解析结果（优先）
   - 如果模型不使用工具，降级到原有Content解析
   - 使用 `MessageParseFromToolCall` 模式解析Tool调用结果

3. **修改executeBranchStep**
   - 更新提示词，明确要求使用 `output_branch_result` 工具输出结果
   - 从ToolCall中解析结果（优先）
   - 如果模型不使用工具，降级到原有Content解析
   - 使用 `MessageParseFromToolCall` 模式解析Tool调用结果

4. **改进cleanJSONContent（作为fallback）**
   - 移除markdown代码块标记（```json ... ```）
   - 移除XML-like标签（<output>...</output>）
   - 验证提取的JSON括号匹配
   - 移除JSON前后的任何非JSON内容

## 修改Executor详细计划

### 1. 在Executor结构体中添加工具解析器

```go
type Executor struct {
    recAgent     *adk.ChatModelAgent
    graph        compose.Runnable[*ExecuteState, *ExecuteState]
    seqParser    schema.MessageParser[StepResult] // 从Content解析（fallback）
    branchParser schema.MessageParser[StepResult] // 从Content解析（fallback）
    toolParser   schema.MessageParser[StepResult] // 从ToolCall解析（优先）
}
```

### 2. 修改NewExecutor

```go
func NewExecutor(ctx context.Context) (*Executor, error) {
    // ... 现有代码 ...

    // 创建输出工具
    seqResultTool := itool.NewStructuredOutputTool(itool.StructuredOutputConfig{
        Name:        "output_seq_result",
        Description: "输出顺序步骤的执行结果，必须使用此工具输出结果",
        WrapData:    false,
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
                Description: "步骤执行中发现的信息、执行操作的结果等",
                Required:    true,
                Example:     "执行成功，未发现问题",
            },
        },
    })

    branchResultTool := itool.NewStructuredOutputTool(itool.StructuredOutputConfig{
        Name:        "output_branch_result",
        Description: "输出分支选择的结果，必须使用此工具输出结果",
        WrapData:    false,
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
                Description: "选择该分支的原因和依据",
                Required:    true,
                Example:     "根据日志分析，问题出在数据库连接",
            },
            {
                Name:        "selected_case",
                Type:        "number",
                Description: "选中的分支索引（从0开始），若没有符合条件的分支则设置为-1",
                Required:    true,
                Example:     0,
            },
        },
    })

    // 创建ReAct Agent，添加新工具
    agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        // ... 现有配置 ...
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: []tool.BaseTool{
                    itool.NewExecTool(targetRepo),
                    itool.NewCopyTool(targetRepo),
                    seqResultTool,     // 新增
                    branchResultTool,   // 新增
                },
            },
        },
    })

    // 创建Tool调用解析器
    toolParser := schema.NewMessageJSONParser[StepResult](&schema.MessageJSONParseConfig{
        ParseFrom: schema.MessageParseFromToolCall, // 从ToolCall解析
    })

    // ... 现有代码 ...
}
```

### 3. 修改executeSeqStep

```go
func (e *Executor) executeSeqStep(ctx context.Context, state *ExecuteState, currentStep playbook.Step, currentContext *StepContext) (*ExecuteState, error) {
    // 渲染提示词模板
    prompt, err := e.renderPrompt(state, currentStep)
    if err != nil {
        return e.handleStepError(state, currentStep, fmt.Sprintf("渲染提示词失败: %v", err))
    }

    // 使用ReAct Agent执行步骤
    runner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent:           e.recAgent,
        EnableStreaming: false,
    })

    iter := runner.Query(ctx, prompt)

    var lastMessage *schema.Message
    var toolResult *schema.Message // 记录Tool调用后的结果消息

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
                // 记录Tool调用结果
                if msg.Role == schema.Tool && msg.Content != "" {
                    toolResult = msg
                }
            }
        }
    }

    // 优先从ToolCall解析结果
    if toolResult != nil {
        result, err := e.toolParser.Parse(ctx, toolResult)
        if err == nil {
            return e.handleStepResult(state, currentStep, currentContext, result)
        }
        // Tool解析失败，降级到Content解析
        log.Warnf("Tool解析失败，降级到Content解析: %v", err)
    }

    // Fallback: 从Content解析
    if lastMessage == nil {
        return e.handleStepError(state, currentStep, "agent未返回响应")
    }

    // 清理消息内容
    lastMessage.Content = cleanJSONContent(lastMessage.Content)
    repairedJson, err := jsonrepair.JSONRepair(lastMessage.Content)
    if err != nil {
        return e.handleStepError(state, currentStep, fmt.Sprintf("解析结果失败:%s, cause:%v", lastMessage.Content, err))
    }
    lastMessage.Content = repairedJson

    result, err := e.seqParser.Parse(ctx, lastMessage)
    if err != nil {
        return e.handleStepError(state, currentStep, fmt.Sprintf("解析结果失败: %v", err))
    }

    return e.handleStepResult(state, currentStep, currentContext, result)
}
```

### 4. 修改executeBranchStep

类似 `executeSeqStep` 的修改，优先从ToolCall解析，降级到Content解析。

### 5. 更新提示词

**renderPrompt** - 添加工具使用说明：
```
请执行当前步骤，必须使用 `output_seq_result` 工具输出结果。

可用的工具：
- output_seq_result: 输出顺序步骤的执行结果
  - status (number, 必填): 0表示未完成，1表示已完成
  - result (string, 必填): 步骤执行的详细结果

重要提示：
1. 必须使用 output_seq_result 工具输出结果
2. 不要直接输出JSON文本
3. 确保status字段正确设置
4. result字段应包含详细的执行信息
```

**renderBranchPrompt** - 添加工具使用说明：
```
请根据当前情况选择最合适的分支，必须使用 `output_branch_result` 工具输出结果。

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

### 6. 改进cleanJSONContent

```go
func cleanJSONContent(content string) string {
    // 1. 移除成对的 </think> 标签及其内容
    for {
        startIdx := strings.Index(content, thinkTagStart)
        if startIdx == -1 {
            break
        }
        endIdx := strings.Index(content[startIdx:], thinkTagEnd)
        if endIdx == -1 {
            content = content[:startIdx] + content[startIdx+len(thinkTagStart):]
            continue
        }
        content = content[:startIdx] + content[startIdx+endIdx+thinkTagEndLen:]
    }

    // 2. 移除单独出现的 </think> 标签
    content = strings.ReplaceAll(content, thinkTagEnd, "")

    // 3. 移除markdown代码块标记
    content = strings.TrimSpace(content)
    if strings.HasPrefix(content, "```json") {
        content = content[7:]
        if idx := strings.Index(content, "```"); idx != -1 {
            content = content[:idx]
        }
    } else if strings.HasPrefix(content, "```") {
        content = content[3:]
        if idx := strings.Index(content, "```"); idx != -1 {
            content = content[:idx]
        }
    }

    // 4. 移除XML-like标签
    content = removeXMLTags(content)

    // 5. 提取JSON内容
    startIdx := strings.Index(content, "{")
    if startIdx == -1 {
        return strings.TrimSpace(content)
    }

    endIdx := strings.LastIndex(content, "}")
    if endIdx == -1 || endIdx < startIdx {
        return strings.TrimSpace(content)
    }

    // 6. 验证括号匹配
    jsonContent := content[startIdx : endIdx+1]
    if !validateBrackets(jsonContent) {
        // 括号不匹配，尝试修复
        jsonContent = fixBrackets(jsonContent)
    }

    return strings.TrimSpace(jsonContent)
}

func removeXMLTags(content string) string {
    // 移除 <output>...</output> 等XML-like标签
    // 实现略
    return content
}

func validateBrackets(content string) bool {
    // 验证{}是否匹配
    balance := 0
    for _, c := range content {
        switch c {
        case '{':
            balance++
        case '}':
            balance--
            if balance < 0 {
                return false
            }
        }
    }
    return balance == 0
}

func fixBrackets(content string) string {
    // 修复不匹配的括号
    balance := 0
    for _, c := range content {
        switch c {
        case '{':
            balance++
        case '}':
            balance--
        }
    }
    if balance > 0 {
        content = content + strings.Repeat("}", balance)
    }
    return content
}
```

## 预期效果

1. **小模型兼容性提升** - Tool调用方式比直接JSON输出更稳定
2. **错误率降低** - 预计从当前的30-50%降低到10%以下
3. **调试更容易** - Tool调用路径清晰，易于追踪问题
4. **代码更清晰** - 明确的解析路径，减少hack代码

## 实施检查清单

- [x] 优化 `internal/tool/structured_output.go`，支持自定义名称和直接输出模式
- [ ] 修改 `Executor` 结构体，添加 `toolParser`
- [ ] 修改 `NewExecutor`，创建 `output_seq_result` 和 `output_branch_result` 工具
- [ ] 修改 `executeSeqStep`，优先从ToolCall解析，降级到Content解析
- [ ] 修改 `executeBranchStep`，优先从ToolCall解析，降级到Content解析
- [ ] 更新 `renderPrompt`，明确要求使用 `output_seq_result` 工具
- [ ] 更新 `renderBranchPrompt`，明确要求使用 `output_branch_result` 工具
- [ ] 改进 `cleanJSONContent` 函数
- [ ] 添加错误日志记录
- [ ] 测试小模型（如Qwen-7B、DeepSeek-7B等）
- [ ] 性能测试
