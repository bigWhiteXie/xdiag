package match

import (
	"fmt"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
)

// StateManager 状态管理器
type StateManager struct{}

// NewStateManager 创建状态管理器
func NewStateManager() *StateManager {
	return &StateManager{}
}

// GetAvailablePlaybooks 获取可用的 playbooks
func (s *StateManager) GetAvailablePlaybooks(
	state *MatchState,
	descriptionBuilder *DescriptionBuilder,
) ([]playbook.Playbook, error) {
	playbooks := descriptionBuilder.FilterExcludedPlaybooks(
		state.AllPlaybooks,
		state.ExcludedPlaybooks,
	)

	if len(playbooks) == 0 {
		return nil, fmt.Errorf("没有可用的playbook")
	}

	return playbooks, nil
}

// SetSelectedPlaybook 设置选中的 playbook
func (s *StateManager) SetSelectedPlaybook(
	state *MatchState,
	playbooks []playbook.Playbook,
	selection *PlaybookSelection,
) error {
	for i := range playbooks {
		if playbooks[i].Name == selection.PlaybookName {
			state.SelectedPlaybook = &playbooks[i]
			return nil
		}
	}
	return fmt.Errorf("未找到名为 '%s' 的playbook", selection.PlaybookName)
}

// SetSelectedRef 设置选中的 ref
func (s *StateManager) SetSelectedRef(
	state *MatchState,
	selection *RefSelection,
) error {
	state.RefStatus = selection.Status

	if selection.Status == 1 {
		for i := range state.SelectedPlaybook.Refs {
			if state.SelectedPlaybook.Refs[i].Name == selection.RefName {
				state.SelectedRef = &state.SelectedPlaybook.Refs[i]
				return nil
			}
		}
		return fmt.Errorf("未找到名为 '%s' 的ref", selection.RefName)
	}

	// 未找到合适的ref，将当前playbook加入排除列表
	state.ExcludedPlaybooks = append(state.ExcludedPlaybooks, state.SelectedPlaybook.Name)
	state.SelectedPlaybook = nil
	return nil
}

// HasMorePlaybooks 检查是否还有更多可选的 playbook
func (s *StateManager) HasMorePlaybooks(state *MatchState) bool {
	return len(state.ExcludedPlaybooks) < len(state.AllPlaybooks)
}

// BuildResult 构建最终结果
func (s *StateManager) BuildResult(state *MatchState) *MatchResult {
	if state.SelectedPlaybook != nil && state.SelectedRef != nil {
		return &MatchResult{
			Playbook: state.SelectedPlaybook,
			Ref:      state.SelectedRef,
			Success:  true,
			Message:  "成功匹配到合适的诊断方案",
		}
	}

	return &MatchResult{
		Success: false,
		Message: "未找到合适的诊断方案",
	}
}
