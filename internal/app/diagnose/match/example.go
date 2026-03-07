package match

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"xdiag/internal/app/playbook"
	"xdiag/internal/targets"
)

// Example 展示如何使用Matcher
func Example() {
	// 1. 创建playbook仓库
	repo := playbook.NewRepo("/path/to/playbooks")

	// 2. 创建LLM客户端
	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		Model:  "gpt-4",
		APIKey: "your-api-key",
	})
	if err != nil {
		log.Fatalf("创建LLM客户端失败: %v", err)
	}

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
	result, err := matcher.Match(context.Background(), target, question)
	if err != nil {
		log.Fatalf("匹配失败: %v", err)
	}

	// 6. 处理结果
	if result.Success {
		fmt.Printf("匹配成功!\n")
		fmt.Printf("选中的Playbook: %s\n", result.Playbook.Name)
		fmt.Printf("选中的Ref: %s\n", result.Ref.Name)
		fmt.Printf("消息: %s\n", result.Message)
	} else {
		fmt.Printf("匹配失败: %s\n", result.Message)
	}
}

// ExampleWithCustomConfig 展示如何使用自定义配置
func ExampleWithCustomConfig() {
	ctx := context.Background()

	// 使用自定义的OpenAI配置
	temp := float32(0.7)
	maxTokens := 2000
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:       "gpt-4",
		APIKey:      "your-api-key",
		BaseURL:     "https://api.openai.com/v1",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	})
	if err != nil {
		log.Fatalf("创建LLM客户端失败: %v", err)
	}

	repo := playbook.NewRepo("/path/to/playbooks")
	matcher, err := NewMatcher(repo, chatModel)
	if err != nil {
		log.Fatalf("创建匹配器失败: %v", err)
	}

	target := &targets.Target{
		Name:    "db-server-01",
		Kind:    "postgres",
		Address: "192.168.1.200",
		Port:    5432,
		Tags:    "database,production,postgres",
	}

	result, err := matcher.Match(ctx, target, "数据库查询响应缓慢")
	if err != nil {
		log.Fatalf("匹配失败: %v", err)
	}

	if result.Success {
		// 获取完整的Book信息
		book, err := repo.GetBook(result.Playbook.Name, result.Ref.Name)
		if err != nil {
			log.Fatalf("获取Book失败: %v", err)
		}
		fmt.Printf("诊断方案: %s\n", book.Name)
		fmt.Printf("步骤数: %d\n", len(book.Steps))
	}
}
