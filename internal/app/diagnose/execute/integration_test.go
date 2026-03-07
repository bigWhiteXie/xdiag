package execute_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"xdiag/internal/app/diagnose/execute"
	"xdiag/internal/app/playbook"
	"xdiag/internal/app/targets"
	"xdiag/internal/llm"
	"xdiag/internal/svc"

	"github.com/stretchr/testify/assert"
)

// TestExecutorWithMockData 使用模拟数据测试执行器
func TestExecutorWithMockData(t *testing.T) {
	ctx := context.Background()

	// 创建模拟数据
	book := &playbook.Book{
		Name: "磁盘状态检查",
		Steps: []playbook.Step{
			{
				Kind: "branch",
				Desc: "使用`df -h /` 查看根路径磁盘使用情况",
				Cases: []playbook.CaseBlock{
					{
						Case: "容量大的磁盘空间还充足则直接返回，告诉用户当前磁盘容量充足",
						Steps: []playbook.Step{
							{
								Kind: "seq",
								Desc: "通过du命令找到挂载在该磁盘路径下的占用磁盘最大的前5个目录，并展示它们的大小",
							},
						},
					},
					{
						Case: "若存在超过10g大小的磁盘，可用容量小于500m时执行该方案",
					},
				},
			},
		},
	}
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_route_agent.db")

	// 初始化 SQLite 仓库
	repo, err := targets.NewSQLiteRepo(dbPath)
	assert.NoError(t, err)
	svc.SetTargetsRepo(repo)
	defer repo.Close()
	target := &targets.Target{
		ID:       1,
		Name:     "节点1",
		Kind:     "node",
		Address:  "192.168.1.5",
		Username: "xielei",
		Password: "j3391111!",
		Port:     22,
		Tags:     "production,database",
	}
	repo.Create(ctx, target)
	svc.SetTargetsRepo(repo)
	llmConfig := &llm.ClientConfig{
		APIKey:    "b6ddebfe0af182f2a015e81448b09d71.thjX2dtaj8XvAJ8d",
		BaseURL:   "http://localhost:1234/v1",
		ModelName: "glm-4.7-flash",
		Provider:  "openai", // 使用 custom provider 兼容 OpenAI API
	}

	model, err := llm.NewClient(ctx, llmConfig)
	svc.SetModel(model)

	question := "检查磁盘空间占用情况"
	// 创建执行器
	executor, err := execute.NewExecutor(ctx)
	if err != nil {
		t.Fatalf("创建执行器失败: %v", err)
	}

	// 执行诊断
	evtChan, err := executor.Execute(ctx, book, target, question)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	var lastEvt execute.ExecuteEvent
	for evt := range evtChan {
		lastEvt = evt
		bytes, _ := json.Marshal(evt)
		t.Logf("事件: %s", bytes)
	}

	t.Log(lastEvt.Message)

}
