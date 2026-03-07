# Match 包 - 智能方案匹配器

## 概述

Match 包提供了一个基于 eino 框架的智能方案匹配器，用于根据目标资产和用户问题自动选择最合适的诊断方案(Playbook)和具体参考(Ref)。

## 核心特性

- **基于 eino Graph**: 使用状态机实现多步骤匹配流程
- **智能排除机制**: 当某个 Playbook 下没有合适的 Ref 时，自动排除该 Playbook 并重新选择
- **LLM 驱动**: 利用大语言模型进行智能决策
- **结构化输出**: 使用 JSON 格式确保输出可解析

## 工作流程

```
┌─────────────────┐
│  开始           │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 步骤1:          │
│ 选择 Playbook   │◄──────┐
│ (排除已失败的)  │       │
└────────┬────────┘       │
         │                │
         ▼                │
┌─────────────────┐       │
│ 步骤2:          │       │
│ 选择 Ref        │       │
└────────┬────────┘       │
         │                │
         ▼                │
    ┌────────┐            │
    │找到合适│            │
    │的 Ref? │            │
    └───┬─┬──┘            │
        │ │               │
     是 │ │ 否            │
        │ └───────────────┘
        │   (排除当前Playbook)
        ▼
┌─────────────────┐
│  完成           │
└─────────────────┘
```

## 使用方法

### 1. 创建匹配器

```go
import (
    "context"
    "github.com/cloudwego/eino-ext/components/model/openai"
    "xdiag/internal/app/match"
    "xdiag/internal/app/playbook"
)

// 创建 playbook 仓库
repo := playbook.NewRepo("/path/to/playbooks")

// 创建 LLM 客户端
chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
    Model:  "gpt-4",
    APIKey: "your-api-key",
})

// 创建匹配器
matcher, err := match.NewMatcher(repo, chatModel)
```

### 2. 执行匹配

```go
// 准备目标资产
target := &targets.Target{
    Name:    "web-server-01",
    Kind:    "node",
    Address: "192.168.1.100",
    Port:    22,
    Tags:    "web,production,linux",
}

// 执行匹配
result, err := matcher.Match(context.Background(), target, "CPU使用率过高")

// 处理结果
if result.Success {
    fmt.Printf("Playbook: %s\n", result.Playbook.Name)
    fmt.Printf("Ref: %s\n", result.Ref.Name)
}
```

## 数据结构

### MatchState

状态机的状态，包含匹配过程中的所有信息:

```go
type MatchState struct {
    Target            *targets.Target      // 目标资产
    Question          string               // 用户问题
    AllPlaybooks      []playbook.Playbook  // 所有可用的 Playbook
    ExcludedPlaybooks []string             // 已排除的 Playbook 名称
    SelectedPlaybook  *playbook.Playbook   // 当前选中的 Playbook
    SelectedRef       *playbook.Ref        // 选中的 Ref
    RefStatus         int                  // 0: 未找到, 1: 找到
}
```

### MatchResult

匹配结果:

```go
type MatchResult struct {
    Playbook *playbook.Playbook  // 匹配到的 Playbook
    Ref      *playbook.Ref       // 匹配到的 Ref
    Success  bool                // 是否成功
    Message  string              // 结果消息
}
```

## LLM 提示词设计

### 步骤1: 选择 Playbook

输入信息:
- 目标资产信息(名称、类型、地址、标签)
- 用户问题
- 可用的 Playbook 列表(排除已失败的)

输出格式:
```json
{
  "playbook_name": "选择的playbook名称",
  "reason": "选择理由"
}
```

### 步骤2: 选择 Ref

输入信息:
- 目标资产信息
- 用户问题
- 当前 Playbook 信息
- 可用的 Ref 列表

输出格式:
```json
{
  "ref_name": "选择的ref名称",
  "status": 1,
  "reason": "选择理由"
}
```

如果没有合适的 Ref:
```json
{
  "ref_name": "",
  "status": 0,
  "reason": "未找到合适的原因"
}
```

## 错误处理

匹配器会在以下情况返回错误:
- 加载 Playbook 失败
- LLM 调用失败
- JSON 解析失败
- 找不到 LLM 选择的 Playbook/Ref

如果所有 Playbook 都被排除仍未找到合适的方案，会返回 `Success: false` 的结果。

## 扩展性

### 自定义 LLM 配置

```go
chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    Model:       "gpt-4",
    Temperature: 0.7,
    MaxTokens:   2000,
})
```

### 添加更多匹配策略

可以通过修改提示词或添加新的节点来扩展匹配逻辑。

## 依赖

- `github.com/cloudwego/eino`: Graph 和状态机框架
- `github.com/cloudwego/eino-ext/components/model/openai`: OpenAI LLM 客户端
- `xdiag/internal/app/playbook`: Playbook 数据结构和仓库
- `xdiag/internal/targets`: 目标资产数据结构
