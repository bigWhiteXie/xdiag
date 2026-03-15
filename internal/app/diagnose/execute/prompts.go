package execute

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
)

const (
	// agentInstruction 是诊断执行器的 Agent 指令
	agentInstruction = `你是一个专业的系统诊断专家，能够根据给定的诊断步骤执行相应的操作。

# 工作方式
1. 仔细阅读当前需要执行的步骤描述
2. 根据步骤类型和描述，使用可用的工具执行相应的操作
3. 收集执行过程中的信息和结果
4. 必须使用指定的结构化输出工具来输出你的执行结果

# 重要提示
- 对于顺序步骤（seq类型），必须使用 output_seq_result 工具输出结果
- 对于分支步骤（branch类型），必须使用 output_branch_result 工具输出结果
- 不要直接输出 JSON 文本，必须通过工具调用输出
- 确保工具调用中包含所有必填字段
- 不要在工具调用之外输出任何结果，所有结果必须通过工具返回
`

	// seqPromptTemplate 是顺序步骤的提示词模板（使用工具调用方式）
	seqPromptTemplate = `你是一个专业的系统诊断专家。请根据以下信息执行当前诊断步骤。

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

请执行当前步骤，并使用 ` + "`output_seq_result`" + ` 工具输出结果。

可用的工具：
- output_seq_result: 输出顺序步骤的执行结果
  - status (number, 必填): 0表示未完成，1表示已完成
  - result (string, 必填): 步骤执行的详细结果

重要提示：
1. 必须使用 output_seq_result 工具输出结果
2. 不要直接输出JSON文本
3. 确保status字段正确设置
4. result字段应包含详细的执行信息
5. 在工具调用之外不要输出任何结果或结论
6. 如果需要执行诊断操作，应该先调用相关工具，然后使用 output_seq_result 输出结果`

	// branchPromptTemplate 是分支选择的提示词模板（使用工具调用方式）
	branchPromptTemplate = `你是一个专业的系统诊断专家。请根据以下信息选择合适的诊断分支。若没有分支符合条件，则将selected_case设置为-1

## 当前环境
{{with .Target}}
- 目标名称: {{.Name}}
- 目标类型: {{.Kind}}
- 目标地址: {{.Address}}:{{.Port}}
- 目标标签: {{.Tags}}
{{end}}

{{if .CurrentContext}}
## 执行上下文
{{.CurrentContext}}
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

请根据当前情况和已执行步骤的结果，选择最合适的分支，必须使用 ` + "`output_branch_result`" + ` 工具输出结果。

工具使用：
- output_branch_result: 输出分支选择的结果
  - status (number, 必填): 0表示未完成，1表示已完成
  - result (string, 必填): 选择该分支的原因和依据
  - selected_case (number, 必填): 选中的分支索引（从0开始），若没有符合条件的分支则设置为-1

重要：
1. 必须使用 output_branch_result 工具输出结果
2. 不要直接输出JSON文本
3. selected_case必须是有效的分支索引或-1
4. result字段应说明选择该分支的原因
5. 在工具调用之外不要输出任何结果或结论
6. 必须在最后调用 output_branch_result 工具输出你的选择`
)

// getAgentInstruction 获取Agent指令
func getAgentInstruction() string {
	return agentInstruction
}

// getSeqPromptTemplate 获取顺序步骤提示词模板
func getSeqPromptTemplate() string {
	return seqPromptTemplate
}

// getBranchPromptTemplate 获取分支选择提示词模板
func getBranchPromptTemplate() string {
	return branchPromptTemplate
}

// add 模板辅助函数
func add(a, b int) int {
	return a + b
}

// renderSeqPrompt 渲染顺序步骤提示词
func renderSeqPrompt(state *ExecuteState, step playbook.Step) (string, error) {
	data := map[string]any{
		"Target":         state.Target,
		"Question":       state.Question,
		"CurrentStep":    step,
		"CurrentContext": state.CurrentContext,
		"ExecutedSteps":  state.ExecutedSteps,
	}

	tmpl := template.Must(template.New("seq_prompt").Funcs(template.FuncMap{
		"add": add,
	}).Parse(getSeqPromptTemplate()))

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("模板执行失败: %w", err)
	}

	return sb.String(), nil
}

// renderBranchPrompt 渲染分支选择提示词
func renderBranchPrompt(state *ExecuteState, step playbook.Step) (string, error) {
	if state.Target == nil {
		return "", fmt.Errorf("target 不能为 nil")
	}

	data := map[string]any{
		"Target":         state.Target,
		"Question":       state.Question,
		"CurrentStep":    step,
		"ExecutedSteps":  state.ExecutedSteps,
		"CurrentContext": state.CurrentContext,
	}

	tmpl := template.Must(template.New("branch_prompt").Funcs(template.FuncMap{
		"add": add,
	}).Parse(getBranchPromptTemplate()))

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("模板执行失败: %w", err)
	}

	return sb.String(), nil
}
