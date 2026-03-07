package match

import (
	"context"
	"fmt"
	"log"

	"xdiag/internal/app/playbook"
	"xdiag/internal/llm"
	"xdiag/internal/targets"
)

// IntegrationExample 展示完整的集成使用示例
func IntegrationExample() {
	ctx := context.Background()

	// 1. 创建 LLM 客户端 (使用项目中的 client_factory)
	chatModel, err := llm.NewClient(ctx, &llm.ClientConfig{
		Provider:  "openai",
		ModelName: "gpt-4",
		APIKey:    "your-api-key",
		BaseURL:   "https://api.openai.com/v1",
	})
	if err != nil {
		log.Fatalf("创建LLM客户端失败: %v", err)
	}

	// 2. 创建 playbook 仓库
	repo := playbook.NewRepo("/path/to/playbooks")

	// 3. 创建匹配器
	matcher, err := NewMatcher(repo, chatModel)
	if err != nil {
		log.Fatalf("创建匹配器失败: %v", err)
	}

	// 4. 准备目标资产
	target := &targets.Target{
		ID:       1,
		Name:     "web-server-01",
		Kind:     "node",
		Address:  "192.168.1.100",
		Port:     22,
		Username: "admin",
		Tags:     "web,production,linux",
	}

	// 5. 执行匹配
	question := "服务器CPU使用率过高，需要诊断性能问题"
	result, err := matcher.Match(ctx, target, question)
	if err != nil {
		log.Fatalf("匹配失败: %v", err)
	}

	// 6. 处理结果
	if result.Success {
		fmt.Printf("✓ 匹配成功!\n")
		fmt.Printf("  Playbook: %s\n", result.Playbook.Name)
		fmt.Printf("  描述: %s\n", result.Playbook.Desc)
		fmt.Printf("  Ref: %s\n", result.Ref.Name)
		fmt.Printf("  Ref描述: %s\n", result.Ref.Desc)

		// 7. 获取完整的诊断方案
		book, err := repo.GetBook(result.Playbook.Name, result.Ref.Name)
		if err != nil {
			log.Fatalf("获取Book失败: %v", err)
		}

		fmt.Printf("\n诊断步骤:\n")
		for i, step := range book.Steps {
			fmt.Printf("  %d. [%s] %s\n", i+1, step.Kind, step.Desc)
		}
	} else {
		fmt.Printf("✗ 匹配失败: %s\n", result.Message)
	}
}

// BatchMatchExample 展示批量匹配的示例
func BatchMatchExample() {
	ctx := context.Background()

	// 创建匹配器
	chatModel, _ := llm.NewClient(ctx, &llm.ClientConfig{
		Provider:  "openai",
		ModelName: "gpt-4",
		APIKey:    "your-api-key",
	})
	repo := playbook.NewRepo("/path/to/playbooks")
	matcher, _ := NewMatcher(repo, chatModel)

	// 准备多个目标和问题
	cases := []struct {
		target   *targets.Target
		question string
	}{
		{
			target: &targets.Target{
				Name: "db-server-01",
				Kind: "postgres",
				Tags: "database,production",
			},
			question: "数据库查询响应缓慢",
		},
		{
			target: &targets.Target{
				Name: "web-server-01",
				Kind: "node",
				Tags: "web,production",
			},
			question: "网站访问超时",
		},
		{
			target: &targets.Target{
				Name: "cache-server-01",
				Kind: "redis",
				Tags: "cache,production",
			},
			question: "缓存命中率低",
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

		if result.Success {
			fmt.Printf("%d. [%s] ✓ %s -> %s\n",
				i+1, c.target.Name, result.Playbook.Name, result.Ref.Name)
		} else {
			fmt.Printf("%d. [%s] ✗ %s\n",
				i+1, c.target.Name, result.Message)
		}
	}
}
