package route

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bigWhiteXie/xdiag/internal/app/targets"
	"github.com/bigWhiteXie/xdiag/internal/llm"
	"github.com/bigWhiteXie/xdiag/internal/svc"
	innertool "github.com/bigWhiteXie/xdiag/internal/tool"

	"github.com/stretchr/testify/assert"
)

// TestRouteTargetAgent_Integration 测试完整的 RouteTargetAgent 集成
// 这个测试需要实际的数据库和模拟的 LLM 响应
func TestRouteTargetAgent_Integration(t *testing.T) {
	// 创建临时数据库文件
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_targets.db")

	// 初始化 SQLite 仓库
	repo, err := targets.NewSQLiteRepo(dbPath)
	assert.NoError(t, err)
	defer repo.Close()

	// 添加测试目标
	testTargets := []*targets.Target{
		{
			ID:      1,
			Name:    "web-server-prod",
			Kind:    "server",
			Address: "192.168.1.100",
			Port:    22,
			Tags:    "production,web,nginx",
		},
		{
			ID:      2,
			Name:    "mysql-db-prod",
			Kind:    "mysql",
			Address: "192.168.1.101",
			Port:    3306,
			Tags:    "production,database,mysql",
		},
		{
			ID:      3,
			Name:    "redis-cache",
			Kind:    "redis",
			Address: "192.168.1.102",
			Port:    6379,
			Tags:    "production,cache,redis",
		},
	}

	for _, target := range testTargets {
		err = repo.Create(context.Background(), target)
		assert.NoError(t, err)
	}

	// 设置服务上下文
	svc.SetTargetsRepo(repo)

	// 注意：由于 Eino 框架需要真实的 LLM 模型，这个测试在没有配置 LLM 的情况下会失败
	// 在实际使用中，需要配置适当的 LLM 模型

	t.Skip("Skipping integration test - requires actual LLM configuration")
}

// TestTargetFinderTool_Integration 测试 TargetFinderTool 与 SQLite 仓库的集成
func TestTargetFinderTool_Integration(t *testing.T) {
	// 创建临时数据库文件
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_targets_tool.db")

	// 初始化 SQLite 仓库
	repo, err := targets.NewSQLiteRepo(dbPath)
	assert.NoError(t, err)
	defer repo.Close()

	// 添加测试目标
	testTargets := []*targets.Target{
		{
			ID:      1,
			Name:    "test-web-server",
			Kind:    "server",
			Address: "10.0.0.100",
			Tags:    "test,web",
		},
		{
			ID:      2,
			Name:    "test-mysql-db",
			Kind:    "mysql",
			Address: "10.0.0.101",
			Tags:    "test,database",
		},
	}

	for _, target := range testTargets {
		err = repo.Create(context.Background(), target)
		assert.NoError(t, err)
	}

	// 创建 TargetFinderTool
	tool := innertool.NewTargetFinderTool(repo)

	// 测试 name eq filter
	inputJSON := `{"filters": "name eq test-web-server"}`
	result, err := tool.InvokableRun(context.Background(), inputJSON)
	assert.NoError(t, err)

	var output struct {
		Targets []targets.Target `json:"targets"`
		Error   string           `json:"error,omitempty"`
	}
	err = json.Unmarshal([]byte(result), &output)
	assert.NoError(t, err)
	assert.Len(t, output.Targets, 1)
	assert.Equal(t, uint(1), output.Targets[0].ID)
	assert.Equal(t, "test-web-server", output.Targets[0].Name)

	// 测试 address eq filter (使用 address 而不是 ip)
	inputJSON = `{"filters": "address eq 10.0.0.101"}`
	result, err = tool.InvokableRun(context.Background(), inputJSON)
	assert.NoError(t, err)

	err = json.Unmarshal([]byte(result), &output)
	assert.NoError(t, err)
	assert.Len(t, output.Targets, 1)
	assert.Equal(t, uint(2), output.Targets[0].ID)
	assert.Equal(t, "10.0.0.101", output.Targets[0].Address)

	// 测试 tag like filter
	inputJSON = `{"filters": "tag like database"}`
	result, err = tool.InvokableRun(context.Background(), inputJSON)
	assert.NoError(t, err)

	err = json.Unmarshal([]byte(result), &output)
	assert.NoError(t, err)
	assert.Len(t, output.Targets, 1)
	assert.Equal(t, "test,database", output.Targets[0].Tags)
}

// setupTestDB 创建并初始化测试数据库
func setupTestDB(t *testing.T) (string, targets.Repo) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "integration_test.db")

	repo, err := targets.NewSQLiteRepo(dbPath)
	assert.NoError(t, err)

	// 添加一些测试数据
	testData := []*targets.Target{
		{ID: 1, Name: "server-1", Kind: "server", Address: "192.168.1.10", Tags: "prod,web"},
		{ID: 2, Name: "db-1", Kind: "mysql", Address: "192.168.1.11", Tags: "prod,db"},
		{ID: 3, Name: "cache-1", Kind: "redis", Address: "192.168.1.12", Tags: "prod,cache"},
	}

	for _, target := range testData {
		err = repo.Create(context.Background(), target)
		assert.NoError(t, err)
	}

	return dbPath, repo
}

// cleanupTestDB 清理测试数据库
func cleanupTestDB(t *testing.T, dbPath string, repo targets.Repo) {
	if repo != nil {
		repo.Close()
	}
	if dbPath != "" {
		os.Remove(dbPath)
	}
}

// TestRouteTargetAgent_WithRealLLM 测试 RouteTargetAgent 与真实 LLM 的集成
func TestRouteTargetAgent_WithRealLLM(t *testing.T) {
	// 创建临时数据库文件
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_route_agent.db")

	// 初始化 SQLite 仓库
	repo, err := targets.NewSQLiteRepo(dbPath)
	assert.NoError(t, err)
	defer repo.Close()

	// 添加测试目标：2个node、1个redis、1个pg
	testTargets := []*targets.Target{
		{
			ID:      1,
			Name:    "web-node-01",
			Kind:    "node",
			Address: "192.168.1.10",
			Port:    22,
			Tags:    "production,web,frontend",
		},
		{
			ID:      2,
			Name:    "api-node-02",
			Kind:    "node",
			Address: "192.168.1.11",
			Port:    22,
			Tags:    "production,api,backend",
		},
		{
			ID:      3,
			Name:    "redis-cache-01",
			Kind:    "redis",
			Address: "192.168.1.20",
			Port:    6379,
			Tags:    "production,cache,redis",
		},
		{
			ID:      4,
			Name:    "postgres-db-01",
			Kind:    "pg",
			Address: "192.168.1.30",
			Port:    5432,
			Tags:    "production,database,postgresql",
		},
	}

	for _, target := range testTargets {
		err = repo.Create(context.Background(), target)
		assert.NoError(t, err)
	}

	// 配置 LLM 客户端（智谱 AI）
	ctx := context.Background()
	llmConfig := &llm.ClientConfig{
		APIKey:    "b6ddebfe0af182f2a015e81448b09d71.thjX2dtaj8XvAJ8d",
		BaseURL:   "http://localhost:1234/v1",
		ModelName: "glm-4.7-flash",
		Protocol:  "openai", // 使用 custom provider 兼容 OpenAI API
	}

	model, err := llm.NewClient(ctx, llmConfig)
	assert.NoError(t, err)
	assert.NotNil(t, model)

	// 设置服务上下文
	svc.SetTargetsRepo(repo)
	svc.SetModel(model)

	// 创建 RouteTargetAgent
	agent, err := NewTargetRouteAgent(ctx, true)
	assert.NoError(t, err)
	assert.NotNil(t, agent)

	// 测试场景1：查找 web 前端节点
	t.Run("Find web frontend node", func(t *testing.T) {
		question := "我的前端网站访问很慢，需要检查 web 服务器"
		targetID, err := agent.Run(ctx, question)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), targetID, "应该找到 web-node-01")

		// 验证找到的目标
		target, err := repo.GetByID(ctx, targetID)
		assert.NoError(t, err)
		assert.Equal(t, "web-node-01", target.Name)
		t.Logf("✓ 成功找到目标: %s (ID: %d)", target.Name, target.ID)
	})

	// 测试场景2：查找 API 后端节点
	t.Run("Find API backend node", func(t *testing.T) {
		question := "API 接口响应超时，需要排查后端服务"
		targetID, err := agent.Run(ctx, question)
		assert.NoError(t, err)
		assert.Equal(t, uint(2), targetID, "应该找到 api-node-02")

		target, err := repo.GetByID(ctx, targetID)
		assert.NoError(t, err)
		assert.Equal(t, "api-node-02", target.Name)
		t.Logf("✓ 成功找到目标: %s (ID: %d)", target.Name, target.ID)
	})

	// 测试场景3：查找 Redis 缓存
	t.Run("Find Redis cache", func(t *testing.T) {
		question := "缓存命中率很低，需要检查 Redis 服务"
		targetID, err := agent.Run(ctx, question)
		assert.NoError(t, err)
		assert.Equal(t, uint(3), targetID, "应该找到 redis-cache-01")

		target, err := repo.GetByID(ctx, targetID)
		assert.NoError(t, err)
		assert.Equal(t, "redis-cache-01", target.Name)
		t.Logf("✓ 成功找到目标: %s (ID: %d)", target.Name, target.ID)
	})

	// 测试场景4：查找 PostgreSQL 数据库
	t.Run("Find PostgreSQL database", func(t *testing.T) {
		question := "数据库查询很慢，需要检查 PostgreSQL"
		targetID, err := agent.Run(ctx, question)
		assert.NoError(t, err)
		assert.Equal(t, uint(4), targetID, "应该找到 postgres-db-01")

		target, err := repo.GetByID(ctx, targetID)
		assert.NoError(t, err)
		assert.Equal(t, "postgres-db-01", target.Name)
		t.Logf("✓ 成功找到目标: %s (ID: %d)", target.Name, target.ID)
	})

	// 测试场景5：通过 IP 地址查找
	t.Run("Find by IP address", func(t *testing.T) {
		question := "192.168.1.20 这台机器的连接数异常"
		targetID, err := agent.Run(ctx, question)
		assert.NoError(t, err)
		assert.Equal(t, uint(3), targetID, "应该通过 IP 找到 redis-cache-01")

		target, err := repo.GetByID(ctx, targetID)
		assert.NoError(t, err)
		assert.Equal(t, "192.168.1.20", target.Address)
		t.Logf("✓ 成功通过 IP 找到目标: %s (IP: %s)", target.Name, target.Address)
	})

	// 测试场景6：找不到相关目标
	t.Run("No matching target", func(t *testing.T) {
		question := "MongoDB 数据库连接失败"
		targetID, err := agent.Run(ctx, question)
		assert.NoError(t, err)
		assert.Equal(t, uint(0), targetID, "应该返回 0 表示没有找到目标")
		t.Logf("✓ 正确识别出没有匹配的目标")
	})
}
