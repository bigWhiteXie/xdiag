package match

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"xdiag/internal/app/playbook"
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

func TestMatcher_BuildDescriptions(t *testing.T) {
	mockRepo := new(MockRepo)
	mockModel := new(MockChatModel)

	matcher := &Matcher{
		repo:      mockRepo,
		chatModel: mockModel,
	}

	// 测试 buildPlaybooksDescription
	playbooks := []playbook.Playbook{
		{
			Name: "网络诊断",
			Desc: "诊断网络连接问题",
			Tags: []string{"network", "connectivity"},
		},
		{
			Name: "性能诊断",
			Desc: "诊断系统性能问题",
			Tags: []string{"performance", "cpu"},
		},
	}

	desc := matcher.buildPlaybooksDescription(playbooks)
	assert.Contains(t, desc, "网络诊断")
	assert.Contains(t, desc, "性能诊断")
	assert.Contains(t, desc, "network")

	// 测试 buildRefsDescription
	refs := []playbook.Ref{
		{
			Name: "ping测试",
			Desc: "使用ping测试网络连通性",
			Log:  "/var/log/ping.log",
		},
		{
			Name: "traceroute测试",
			Desc: "使用traceroute追踪网络路径",
		},
	}

	refDesc := matcher.buildRefsDescription(refs)
	assert.Contains(t, refDesc, "ping测试")
	assert.Contains(t, refDesc, "traceroute测试")
	assert.Contains(t, refDesc, "/var/log/ping.log")
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
