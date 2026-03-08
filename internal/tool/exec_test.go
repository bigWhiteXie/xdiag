package tool

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bigWhiteXie/xdiag/internal/app/targets"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTargetRepo 是targets.Repo的mock实现
type MockTargetRepo struct {
	mock.Mock
}

func (m *MockTargetRepo) Create(ctx context.Context, target *targets.Target) error {
	args := m.Called(ctx, target)
	return args.Error(0)
}

func (m *MockTargetRepo) GetByID(ctx context.Context, id uint) (*targets.Target, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*targets.Target), args.Error(1)
}

func (m *MockTargetRepo) GetByName(ctx context.Context, name string) (*targets.Target, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*targets.Target), args.Error(1)
}

func (m *MockTargetRepo) List(ctx context.Context, filters map[string]targets.Op) ([]*targets.Target, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*targets.Target), args.Error(1)
}

func (m *MockTargetRepo) Update(ctx context.Context, target *targets.Target) error {
	args := m.Called(ctx, target)
	return args.Error(0)
}

func (m *MockTargetRepo) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTargetRepo) GetAllKinds() ([]string, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockTargetRepo) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestExecTool_Info(t *testing.T) {
	mockRepo := new(MockTargetRepo)
	tool := NewExecTool(mockRepo)

	ctx := context.Background()
	info, err := tool.Info(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "shell_exec", info.Name)
	assert.Contains(t, info.Desc, "执行shell命令")
}

func TestExecTool_LocalExecution(t *testing.T) {
	mockRepo := new(MockTargetRepo)
	tool := NewExecTool(mockRepo)

	ctx := context.Background()

	// 测试本地执行简单命令
	input := ExecToolInput{
		Cmd:    "echo 'hello world'",
		Expire: 5,
	}

	inputJSON, _ := json.Marshal(input)
	result, err := tool.InvokableRun(ctx, string(inputJSON))

	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	var output ExecToolOutput
	err = json.Unmarshal([]byte(result), &output)
	assert.NoError(t, err)
	assert.True(t, output.Success)
	assert.Contains(t, output.Stdout, "hello world")
}

func TestExecTool_LocalExecutionWithPath(t *testing.T) {
	mockRepo := new(MockTargetRepo)
	tool := NewExecTool(mockRepo)

	ctx := context.Background()

	// 测试带路径的本地执行
	input := ExecToolInput{
		Path:   "/tmp",
		Cmd:    "pwd",
		Expire: 5,
	}

	inputJSON, _ := json.Marshal(input)
	result, err := tool.InvokableRun(ctx, string(inputJSON))

	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	var output ExecToolOutput
	err = json.Unmarshal([]byte(result), &output)
	assert.NoError(t, err)
	assert.True(t, output.Success)
	assert.Contains(t, output.Stdout, "/tmp")
}

func TestExecTool_LocalExecutionWithError(t *testing.T) {
	mockRepo := new(MockTargetRepo)
	tool := NewExecTool(mockRepo)

	ctx := context.Background()

	// 测试执行失败的命令
	input := ExecToolInput{
		Cmd:    "exit 1",
		Expire: 5,
	}

	inputJSON, _ := json.Marshal(input)
	result, err := tool.InvokableRun(ctx, string(inputJSON))

	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	var output ExecToolOutput
	err = json.Unmarshal([]byte(result), &output)
	assert.NoError(t, err)
	assert.False(t, output.Success)
	assert.NotEmpty(t, output.Error)
}

func TestExecTool_LocalExecutionIgnoreError(t *testing.T) {
	mockRepo := new(MockTargetRepo)
	tool := NewExecTool(mockRepo)

	ctx := context.Background()

	// 测试忽略错误
	input := ExecToolInput{
		Cmd:       "exit 1",
		Expire:    5,
		IgnoreErr: true,
	}

	inputJSON, _ := json.Marshal(input)
	result, err := tool.InvokableRun(ctx, string(inputJSON))

	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	var output ExecToolOutput
	err = json.Unmarshal([]byte(result), &output)
	assert.NoError(t, err)
	assert.True(t, output.Success) // 应该成功，因为ignore_err=true
	assert.Empty(t, output.Error)  // 错误应该被移到stderr
}

func TestExecTool_MissingCmd(t *testing.T) {
	mockRepo := new(MockTargetRepo)
	tool := NewExecTool(mockRepo)

	ctx := context.Background()

	// 测试缺少cmd参数
	input := ExecToolInput{
		Expire: 5,
	}

	inputJSON, _ := json.Marshal(input)
	_, err := tool.InvokableRun(ctx, string(inputJSON))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cmd is required")
}

func TestExecTool_Name(t *testing.T) {
	mockRepo := new(MockTargetRepo)
	tool := NewExecTool(mockRepo)

	assert.Equal(t, "shell_exec", tool.Name())
}

func TestExecTool_Description(t *testing.T) {
	mockRepo := new(MockTargetRepo)
	tool := NewExecTool(mockRepo)

	desc := tool.Description()
	assert.Contains(t, desc, "shell命令")
	assert.Contains(t, desc, "SSH")
}
