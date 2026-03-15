package execute

import (
	"fmt"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
)

const (
	maxRetries       = 3
	statusIncomplete = 0
	statusComplete   = 1
	invalidCaseIndex = -1
)

// hasMoreSteps 检查是否还有步骤需要执行
func (s *ExecuteState) hasMoreSteps() bool {
	if len(s.StepStack) == 0 {
		return false
	}

	currentContext := &s.StepStack[len(s.StepStack)-1]
	return currentContext.CurrentIndex < len(currentContext.Steps)
}

// getCurrentStep 获取当前步骤和上下文
func (s *ExecuteState) getCurrentStep() (step playbook.Step, stepCtx *StepContext) {
	if len(s.StepStack) == 0 {
		return playbook.Step{}, nil
	}

	currentContext := &s.StepStack[len(s.StepStack)-1]
	return currentContext.Steps[currentContext.CurrentIndex], currentContext
}

// shouldFinish 检查是否应该结束执行
func (s *ExecuteState) shouldFinish() bool {
	// 检查是否有错误且超过重试次数
	if s.Error != "" && s.RetryCount >= maxRetries {
		return true
	}

	// 检查步骤栈是否为空
	if len(s.StepStack) == 0 {
		return true
	}

	// 获取当前栈顶
	currentContext := &s.StepStack[len(s.StepStack)-1]

	// 检查当前层级的步骤是否都已完成
	if currentContext.CurrentIndex >= len(currentContext.Steps) {
		// 弹出当前层级
		s.StepStack = s.StepStack[:len(s.StepStack)-1]

		// 如果栈为空，说明所有步骤都已完成
		if len(s.StepStack) == 0 {
			return true
		}
	}

	return false
}

// handleError 处理步骤错误
func (s *ExecuteState) handleError(step playbook.Step, errMsg string) *ExecuteState {
	s.Error = errMsg
	s.RetryCount++

	if s.EventChan != nil {
		s.EventChan <- ExecuteEvent{
			Type:  EventTypeStepError,
			Step:  &step,
			Error: errMsg,
		}
	}

	return s
}

// handleResult 处理步骤执行结果，成功则更新stepCtx到下一个步骤，失败则设置当前执行上下文
func (s *ExecuteState) handleResult(step playbook.Step, stepCtx *StepContext, result StepResult) *ExecuteState {
	if result.Status == statusComplete {
		// 步骤完成
		s.ExecutedSteps = append(s.ExecutedSteps, ExecutedStep{
			Step:   step,
			Result: result,
		})
		stepCtx.CurrentIndex++
		s.RetryCount = 0
		s.Error = ""
		s.CurrentContext = ""

		if s.EventChan != nil {
			s.EventChan <- ExecuteEvent{
				Type:   EventTypeStepComplete,
				Step:   &step,
				Result: &result,
			}
		}
	} else {
		// 步骤未完成，设置上下文并重试
		s.CurrentContext = result.Result
		s.RetryCount++
	}

	return s
}

// handleTextOutput 处理模型的文本输出（当模型返回文本而非工具调用时）
func (s *ExecuteState) handleTextOutput(step playbook.Step, stepCtx *StepContext, textOutput string) *ExecuteState {
	// 将文本输出添加到当前上下文中
	if s.CurrentContext == "" {
		s.CurrentContext = textOutput
	} else {
		s.CurrentContext = s.CurrentContext + "\n\n" + textOutput
	}

	// 增加重试次数，但不要立即失败
	s.RetryCount++

	if s.EventChan != nil {
		emiter := &eventEmitter{}
		emiter.sendAgentThinking(s.EventChan, step, textOutput)
	}

	return s
}

// handleBranchSelection 处理分支选择
func (s *ExecuteState) handleBranchSelection(step playbook.Step, stepCtx *StepContext, result StepResult) *ExecuteState {
	// 当所有分支都不满足条件时，跳到下一个步骤
	if result.SelectedCase < 0 || result.SelectedCase >= len(step.Cases) {
		return s.handleResult(step, stepCtx, result)
	}

	// 记录分支选择
	selectedCase := step.Cases[result.SelectedCase]
	s.ExecutedSteps = append(s.ExecutedSteps, ExecutedStep{
		Step:   step,
		Result: result,
	})

	if s.EventChan != nil {
		s.EventChan <- ExecuteEvent{
			Type:    EventTypeBranchSelect,
			Step:    &step,
			Result:  &result,
			Message: fmt.Sprintf("选择分支: %s", selectedCase.Case),
		}
	}

	// 移动到当前层级的下一个步骤
	stepCtx.CurrentIndex++

	// 将选中的分支步骤压入栈
	if len(selectedCase.Steps) > 0 {
		s.StepStack = append(s.StepStack, StepContext{
			Steps:        selectedCase.Steps,
			CurrentIndex: 0,
		})
	}

	s.RetryCount = 0
	s.Error = ""
	s.CurrentContext = ""

	return s
}
