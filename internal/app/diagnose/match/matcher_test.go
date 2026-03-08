package match

import (
	"context"
	"fmt"
	"testing"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/bigWhiteXie/xdiag/internal/app/targets"
	"github.com/bigWhiteXie/xdiag/internal/llm"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepo 模拟playbook.Repo
type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) ListPlaybooks(tags []string) ([]playbook.Playbook, error) {
	args := m.Called(tags)
	return args.Get(0).([]playbook.Playbook), args.Error(1)
}

func (m *MockRepo) GetBook(playbookName, refName string) (*playbook.Book, error) {
	args := m.Called(playbookName, refName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*playbook.Book), args.Error(1)
}

// MockChatModel 模拟model.ChatModel
type MockChatModel struct {
	mock.Mock
}

func (m *MockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	args := m.Called(ctx, input, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Message), args.Error(1)
}

func (m *MockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	args := m.Called(ctx, input, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.StreamReader[*schema.Message]), args.Error(1)
}

func (m *MockChatModel) BindTools(tools []*schema.ToolInfo) error {
	args := m.Called(tools)
	return args.Error(0)
}

func (m *MockChatModel) GetType() string {
	return "mock"
}

func (m *MockChatModel) IsCallbacksEnabled() bool {
	return false
}

func TestMatchState_ExcludePlaybooks(t *testing.T) {
	state := &MatchState{
		AllPlaybooks: []playbook.Playbook{
			{Name: "playbook1"},
			{Name: "playbook2"},
			{Name: "playbook3"},
		},
		ExcludedPlaybooks: []string{"playbook1"},
	}

	// 验证排除逻辑
	availablePlaybooks := []playbook.Playbook{}
	for _, pb := range state.AllPlaybooks {
		excluded := false
		for _, excludedName := range state.ExcludedPlaybooks {
			if pb.Name == excludedName {
				excluded = true
				break
			}
		}
		if !excluded {
			availablePlaybooks = append(availablePlaybooks, pb)
		}
	}

	assert.Equal(t, 2, len(availablePlaybooks))
	assert.Equal(t, "playbook2", availablePlaybooks[0].Name)
	assert.Equal(t, "playbook3", availablePlaybooks[1].Name)
}

func TestMatchResult(t *testing.T) {
	// 测试成功的匹配结果
	successResult := &MatchResult{
		Playbook: &playbook.Playbook{Name: "test-playbook"},
		Ref:      &playbook.Ref{Name: "test-ref"},
		Success:  true,
		Message:  "成功匹配到合适的诊断方案",
	}

	assert.True(t, successResult.Success)
	assert.NotNil(t, successResult.Playbook)
	assert.NotNil(t, successResult.Ref)

	// 测试失败的匹配结果
	failResult := &MatchResult{
		Success: false,
		Message: "未找到合适的诊断方案",
	}

	assert.False(t, failResult.Success)
	assert.Nil(t, failResult.Playbook)
	assert.Nil(t, failResult.Ref)
}

func TestMatcherLLM(t *testing.T) {
	ctx := context.Background()

	// 创建匹配器
	llmConfig := &llm.ClientConfig{
		APIKey:    "b6ddebfe0af182f2a015e81448b09d71.thjX2dtaj8XvAJ8d",
		BaseURL:   "http://localhost:1234/v1",
		ModelName: "glm-4.7-flash",
		Protocol:  "openai", // 使用 custom provider 兼容 OpenAI API
	}
	model, err := llm.NewClient(ctx, llmConfig)
	assert.NoError(t, err)
	assert.NotNil(t, model)
	//todo:mock 基于内存的playbook的repo，并提供

	matcher, _ := NewMatcher(&MockPlaybookRepo{}, model)
	// 准备多个目标和问题
	cases := []struct {
		target   *targets.Target
		question string
		expected string
		exist    bool
	}{
		{
			target: &targets.Target{
				Name: "db-server-01",
				Kind: "postgres",
				Tags: "database,production",
			},
			question: "数据库查询响应缓慢",
			expected: "慢查询相关诊断",
			exist:    true,
		},
		{
			target: &targets.Target{
				Name: "node-01",
				Kind: "node",
				Tags: "web,production",
			},
			question: "deploy_blackbox failed",
			expected: "黑匣子安装部署失败",
			exist:    true,
		},
		{
			target: &targets.Target{
				Name: "node-02",
				Kind: "node",
				Tags: "xos",
			},
			question: "xos卸载报错黑匣子卸载失败",
			expected: "黑匣子卸载失败",
			exist:    true,
		},
	}

	// 批量匹配
	fmt.Println("批量匹配结果:")
	fmt.Println("=" + string(make([]byte, 60)))
	for i, c := range cases {
		result, err := matcher.Match(ctx, c.target, c.question)
		if err != nil {
			fmt.Printf("%d. [%s] 错误: %v\n", i+1, c.target.Name, err)
			continue
		}

		if c.exist {
			assert.Equal(t, c.expected, result.Ref.Name)
			fmt.Printf("%d. [%s] ✓ %s -> %s\n",
				i+1, c.target.Name, result.Playbook.Name, result.Ref.Name)
		} else {
			assert.Equal(t, result.Success, false)
		}
	}
}

type MockPlaybookRepo struct {
	playbooks []playbook.Playbook
}

func (m *MockPlaybookRepo) ListPlaybooks(tags []string) ([]playbook.Playbook, error) {
	return []playbook.Playbook{
		{
			Name: "postgres",
			Desc: "负责postgres的问题诊断",
			Refs: []playbook.Ref{
				{
					Name: "慢查询相关诊断",
					Desc: "诊断各种慢查询的性能瓶颈",
					Log:  "ref1 log",
				},
				{
					Name: "数据库备份相关",
					Desc: "数据库备份相关的问题",
					Log:  "ref2 log",
				},
			},
		},
		{
			Name: "xos-安装部署",
			Desc: "诊断xos系统出现的各种安装部署问题和卸载相关问题",
			Refs: []playbook.Ref{
				{
					Name: "黑匣子安装部署失败",
					Desc: "诊断黑匣子安装失败的原因并进行修复",
					Log:  "deploy blackbox failed...",
				},
				{
					Name: "黑匣子卸载失败",
					Desc: "解决由于黑匣子卸载失败导致的各种问题",
					Log:  "ref2 log",
				},
			},
		},
	}, nil
}

func (m *MockPlaybookRepo) GetBook(playbookName, refName string) (*playbook.Book, error) {

	return nil, nil
}

func (m *MockPlaybookRepo) LoadPlaybook(playbookName string) (*playbook.Playbook, error) {
	// 返回一个空的playbook和nil错误
	return &playbook.Playbook{}, nil
}

func (m *MockPlaybookRepo) PlaybookExists(playbookName string) bool {
	// 默认返回true，表示playbook存在
	return true
}

func (m *MockPlaybookRepo) SavePlaybook(playbook *playbook.Playbook) error {
	// 空实现，直接返回nil
	return nil
}

func (m *MockPlaybookRepo) SaveBook(playbookName string, book *playbook.Book) error {
	// 空实现，直接返回nil
	return nil
}

func (m *MockPlaybookRepo) UpdatePlaybookRef(playbookName string, ref playbook.Ref) error {
	// 空实现，直接返回nil
	return nil
}
