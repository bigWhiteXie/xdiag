package execute

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
)

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
		"Target":        state.Target,
		"Question":      state.Question,
		"CurrentStep":   step,
		"ExecutedSteps": state.ExecutedSteps,
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
