package match

import (
	"fmt"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/bigWhiteXie/xdiag/internal/app/targets"
)

// PromptBuilder 提示词构建器
type PromptBuilder struct{}

// NewPromptBuilder 创建提示词构建器
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// BuildPlaybookSelectionPrompt 构建选择 playbook 的提示词
func (b *PromptBuilder) BuildPlaybookSelectionPrompt(
	target *targets.Target,
	question string,
	playbooksDesc string,
) string {
	return fmt.Sprintf(`你是一个智能诊断助手。请根据以下信息选择最合适的诊断方案(playbook)。

目标资产信息:
- 名称: %s
- 类型: %s
- 地址: %s:%d
- 标签: %s

用户问题: %s

可用的诊断方案:
%s

请分析目标资产的类型、标签和用户问题，选择最合适的诊断方案。`,
		target.Name,
		target.Kind,
		target.Address,
		target.Port,
		target.Tags,
		question,
		playbooksDesc,
	)
}

// BuildRefSelectionPrompt 构建选择 ref 的提示词
func (b *PromptBuilder) BuildRefSelectionPrompt(
	target *targets.Target,
	question string,
	selectedPlaybook *playbook.Playbook,
	refsDesc string,
) string {
	return fmt.Sprintf(`你是一个智能诊断助手。请根据以下信息从当前诊断方案中选择最合适的具体诊断参考(ref)。

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
		target.Name,
		target.Kind,
		target.Address,
		target.Port,
		target.Tags,
		question,
		selectedPlaybook.Name,
		selectedPlaybook.Desc,
		refsDesc,
	)
}
