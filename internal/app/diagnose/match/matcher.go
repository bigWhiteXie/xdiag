package match

import (
	"context"
	"fmt"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/bigWhiteXie/xdiag/internal/app/targets"
	"github.com/bigWhiteXie/xdiag/pkg/formatter"

	"github.com/cloudwego/eino/compose"
)

// Matcher 方案匹配器
type Matcher struct {
	repo               playbook.Repo
	graph              compose.Runnable[*MatchState, *MatchState]
	llmRunner          *LLMRunner
	promptBuilder      *PromptBuilder
	descriptionBuilder *DescriptionBuilder
	stateManager       *StateManager
}

// NewMatcher 创建新的方案匹配器
func NewMatcher(repo playbook.Repo, chatModel ChatModelInterface, showDetails bool) (*Matcher, error) {
	// 创建组件
	llmRunner := NewLLMRunner(chatModel, LLMRunnerConfig{
		MaxRetries: 3,
		Formatter:  formatter.NewAgentFormatter(showDetails),
	})

	m := &Matcher{
		repo:               repo,
		llmRunner:          llmRunner,
		promptBuilder:      NewPromptBuilder(),
		descriptionBuilder: NewDescriptionBuilder(),
		stateManager:       NewStateManager(),
	}

	// 构建 Graph
	graph, err := m.buildGraph()
	if err != nil {
		return nil, fmt.Errorf("构建graph失败: %w", err)
	}
	m.graph = graph

	return m, nil
}

// Match 执行匹配
func (m *Matcher) Match(ctx context.Context, target *targets.Target, question string) (*MatchResult, error) {
	// 加载所有playbooks
	allPlaybooks, err := m.repo.ListPlaybooks(nil)
	if err != nil {
		return nil, fmt.Errorf("加载playbooks失败: %w", err)
	}

	// 初始化状态
	state := &MatchState{
		Target:            target,
		Question:          question,
		AllPlaybooks:      allPlaybooks,
		ExcludedPlaybooks: []string{},
		RefStatus:         0,
	}

	// 执行graph
	finalState, err := m.graph.Invoke(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("执行匹配失败: %w", err)
	}

	// 构建结果
	return m.stateManager.BuildResult(finalState), nil
}

// buildGraph 构建状态机图
func (m *Matcher) buildGraph() (compose.Runnable[*MatchState, *MatchState], error) {
	graph := compose.NewGraph[*MatchState, *MatchState]()

	// 添加节点
	err := graph.AddLambdaNode("select_playbook", compose.InvokableLambda(m.selectPlaybookNode))
	if err != nil {
		return nil, fmt.Errorf("添加select_playbook节点失败: %w", err)
	}

	err = graph.AddLambdaNode("select_ref", compose.InvokableLambda(m.selectRefNode))
	if err != nil {
		return nil, fmt.Errorf("添加select_ref节点失败: %w", err)
	}

	err = graph.AddLambdaNode("finish", compose.InvokableLambda(m.finishNode))
	if err != nil {
		return nil, fmt.Errorf("添加finish节点失败: %w", err)
	}

	// 设置入口
	graph.AddEdge(compose.START, "select_playbook")

	// 添加边
	// select_playbook -> select_ref (找到playbook)
	graph.AddEdge("select_playbook", "select_ref")

	// select_ref -> finish (找到合适的ref)
	// select_ref -> select_playbook (未找到合适的ref，重新选择playbook)
	err = graph.AddBranch("select_ref", compose.NewGraphBranch(func(ctx context.Context, state *MatchState) (string, error) {
		if state.RefStatus == 1 {
			return "finish", nil
		}
		// 检查是否还有可选的playbook
		if !m.stateManager.HasMorePlaybooks(state) {
			return "finish", nil
		}
		return "select_playbook", nil
	}, map[string]bool{
		"finish":          true,
		"select_playbook": true,
	}))
	if err != nil {
		return nil, fmt.Errorf("添加分支失败: %w", err)
	}

	// finish -> END
	graph.AddEdge("finish", compose.END)

	// 编译graph
	compiled, err := graph.Compile(context.Background())
	if err != nil {
		return nil, fmt.Errorf("编译graph失败: %w", err)
	}

	return compiled, nil
}
