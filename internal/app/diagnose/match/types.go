package match

import (
	"context"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/bigWhiteXie/xdiag/internal/app/targets"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// MatchState 定义状态机的状态
type MatchState struct {
	Target            *targets.Target
	Question          string
	AllPlaybooks      []playbook.Playbook
	ExcludedPlaybooks []string
	SelectedPlaybook  *playbook.Playbook
	SelectedRef       *playbook.Ref
	RefStatus         int // 0: 未找到合适的ref, 1: 找到合适的ref
}

// PlaybookSelection LLM 步骤1的输出
type PlaybookSelection struct {
	PlaybookName string `json:"playbook_name"`
	Reason       string `json:"reason"`
}

// RefSelection LLM 步骤2的输出
type RefSelection struct {
	RefName string `json:"ref_name"`
	Status  int    `json:"status"` // 0: 未找到合适的ref, 1: 找到合适的ref
	Reason  string `json:"reason"`
}

// MatchResult 匹配结果
type MatchResult struct {
	Playbook *playbook.Playbook
	Ref      *playbook.Ref
	Success  bool
	Message  string
}

// ChatModelInterface 定义匹配器需要的最小 LLM 接口
type ChatModelInterface interface {
	Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error)
}
