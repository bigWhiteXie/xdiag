package match

import (
	"context"
	"fmt"

	"github.com/bigWhiteXie/xdiag/internal/tool"
	"github.com/bigWhiteXie/xdiag/pkg/utils"
)

// selectPlaybookNode 步骤1: 选择 playbook
func (m *Matcher) selectPlaybookNode(ctx context.Context, state *MatchState) (*MatchState, error) {
	availablePlaybooks, err := m.stateManager.GetAvailablePlaybooks(state, m.descriptionBuilder)
	if err != nil {
		return state, err
	}

	playbooksDesc := m.descriptionBuilder.BuildPlaybooksDescription(availablePlaybooks)
	prompt := m.promptBuilder.BuildPlaybookSelectionPrompt(state.Target, state.Question, playbooksDesc)

	toolConfig := tool.StructuredOutputConfig{
		WrapData:    true,
		Description: "当确定playbook后，使用此工具输出结果",
		Fields: []tool.FieldDefinition{
			{
				Name:        "playbook_name",
				Type:        "string",
				Description: "选择的playbook名称",
				Required:    true,
				Example:     "mysql_diagnostics",
			},
			{
				Name:        "reason",
				Type:        "string",
				Description: "选择该playbook的理由",
				Required:    true,
				Example:     "该方案适用于MySQL数据库的性能诊断",
			},
		},
	}

	result, err := m.llmRunner.RunWithStructuredOutput(ctx, prompt, toolConfig, func(data map[string]interface{}) (interface{}, error) {
		var selection PlaybookSelection
		if err := utils.UnmarshalMap(data, &selection); err != nil {
			return nil, fmt.Errorf("转换数据失败: %w", err)
		}
		return &selection, nil
	})

	if err != nil {
		return state, err
	}

	selection := result.(*PlaybookSelection)
	if err := m.stateManager.SetSelectedPlaybook(state, availablePlaybooks, selection); err != nil {
		return state, err
	}

	return state, nil
}

// selectRefNode 步骤2: 选择 ref
func (m *Matcher) selectRefNode(ctx context.Context, state *MatchState) (*MatchState, error) {
	if state.SelectedPlaybook == nil {
		return state, fmt.Errorf("未选择playbook")
	}

	refsDesc := m.descriptionBuilder.BuildRefsDescription(state.SelectedPlaybook.Refs)
	prompt := m.promptBuilder.BuildRefSelectionPrompt(
		state.Target,
		state.Question,
		state.SelectedPlaybook,
		refsDesc,
	)

	toolConfig := tool.StructuredOutputConfig{
		WrapData:    true,
		Description: "当确定相关ref或当前无相关ref时使用此工具结构化输出",
		Fields: []tool.FieldDefinition{
			{
				Name:        "ref_name",
				Type:        "string",
				Description: "选择的ref名称(如果没有合适的则为空字符串)",
				Required:    false,
				Example:     "slow_query_analysis",
			},
			{
				Name:        "status",
				Type:        "number",
				Description: "1表示找到合适的ref, 0表示未找到",
				Required:    true,
				Example:     1,
			},
			{
				Name:        "reason",
				Type:        "string",
				Description: "选择理由或未找到的原因",
				Required:    true,
				Example:     "该ref适用于慢查询分析场景",
			},
		},
	}

	result, err := m.llmRunner.RunWithStructuredOutput(ctx, prompt, toolConfig, func(data map[string]interface{}) (interface{}, error) {
		var selection RefSelection
		if err := utils.UnmarshalMap(data, &selection); err != nil {
			return nil, fmt.Errorf("转换数据失败: %w", err)
		}
		return &selection, nil
	})

	if err != nil {
		return state, err
	}

	selection := result.(*RefSelection)
	if err := m.stateManager.SetSelectedRef(state, selection); err != nil {
		return state, err
	}

	return state, nil
}

// finishNode 完成节点
func (m *Matcher) finishNode(ctx context.Context, state *MatchState) (*MatchState, error) {
	return state, nil
}
