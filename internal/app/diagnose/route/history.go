package route

import (
	"fmt"
	"strings"
)

// executionHistory 管理执行历史
type executionHistory struct {
	history []string
}

// newExecutionHistory 创建新的历史记录管理器
func newExecutionHistory() *executionHistory {
	return &executionHistory{
		history: make([]string, 0),
	}
}

// addSuccess 记录成功找到目标
func (h *executionHistory) addSuccess(targetId uint) {
	h.history = append(h.history, fmt.Sprintf("- 成功找到目标: target_id=%d", targetId))
}

// addNotFound 记录未找到目标
func (h *executionHistory) addNotFound() {
	h.history = append(h.history, "- 未找到相关目标")
}

// addFailure 记录失败原因
func (h *executionHistory) addFailure(reason string) {
	h.history = append(h.history, fmt.Sprintf("- 工具调用失败: %s", reason))
}

// buildContext 构建历史上下文字符串
func (h *executionHistory) buildContext() string {
	if len(h.history) == 0 {
		return ""
	}
	return "# 历史执行记录\n" + strings.Join(h.history, "\n")
}
