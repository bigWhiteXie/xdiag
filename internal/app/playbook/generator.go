package playbook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	itool "github.com/bigWhiteXie/xdiag/internal/tool"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Generator 用于根据用户描述生成playbook和book
type Generator struct {
	repo             Repo
	llmClient        model.ToolCallingChatModel
	maxRetries       int
	baseRetryWait    time.Duration
	structOutputTool *itool.StructuredOutputTool
}

// NewGenerator 创建一个新的Generator实例
func NewGenerator(llmClient model.ToolCallingChatModel, playbooksDir string) *Generator {
	repo := NewRepo(playbooksDir)

	// 创建结构化输出工具
	structOutputTool := itool.NewStructuredOutputTool(itool.StructuredOutputConfig{
		Description: "用于输出生成的诊断方案内容",
		Fields: []itool.FieldDefinition{
			{
				Name:        "playbook",
				Type:        "object",
				Description: "playbook信息（仅在playbook不存在时需要提供）",
				Required:    false,
				Properties: []itool.FieldDefinition{
					{
						Name:        "name",
						Type:        "string",
						Description: "playbook名称",
						Required:    true,
					},
					{
						Name:        "desc",
						Type:        "string",
						Description: "playbook描述",
						Required:    true,
					},
					{
						Name:        "required_tags",
						Type:        "array",
						Description: "所需的标签列表",
						Required:    false,
					},
				},
			},
			{
				Name:        "book",
				Type:        "object",
				Description: "诊断方案book的详细步骤",
				Required:    true,
				Properties: []itool.FieldDefinition{
					{
						Name:        "steps",
						Type:        "array",
						Description: "诊断步骤列表",
						Required:    true,
					},
				},
			},
			{
				Name:        "ref",
				Type:        "object",
				Description: "诊断方案的引用信息",
				Required:    true,
				Properties: []itool.FieldDefinition{
					{
						Name:        "desc",
						Type:        "string",
						Description: "简短描述",
						Required:    true,
					},
					{
						Name:        "log",
						Type:        "string",
						Description: "相关日志路径或说明",
						Required:    false,
					},
				},
			},
		},
	})

	return &Generator{
		repo:             repo,
		llmClient:        llmClient,
		maxRetries:       3,
		baseRetryWait:    1 * time.Second,
		structOutputTool: structOutputTool,
	}
}

// GenerateBookRequest 生成book的请求参数
type GenerateBookRequest struct {
	Name         string
	PlaybookName string // playbook名称(大类)
	Description  string // 用户对诊断步骤的描述
}

// GeneratedContent AI生成的内容
type GeneratedContent struct {
	Playbook *Playbook `json:"playbook,omitempty"` // 如果playbook不存在，AI生成的playbook信息
	Book     Book      `json:"book"`               // 生成的book实例
	Ref      Ref       `json:"ref"`                // 生成的ref实例
}

// GenerateAndSave 根据用户描述生成book并落盘
func (g *Generator) GenerateAndSave(ctx context.Context, req GenerateBookRequest) (*Book, error) {
	// 先检查playbook是否存在
	playbookExists := g.repo.PlaybookExists(req.PlaybookName)

	var existingPlaybook *Playbook
	var err error
	if playbookExists {
		existingPlaybook, err = g.repo.LoadPlaybook(req.PlaybookName)
		if err != nil {
			return nil, fmt.Errorf("加载playbook失败: %w", err)
		}
	}

	// 根据playbook是否存在构建不同的AI提示词
	prompt := g.buildPrompt(req, playbookExists, existingPlaybook)

	// 调用AI生成内容，带重试机制
	var content *GeneratedContent
	var lastErr error
	for attempt := 0; attempt < g.maxRetries; attempt++ {
		content, lastErr = g.callAIWithRetry(ctx, prompt, attempt)
		if lastErr == nil {
			break
		}

		if attempt < g.maxRetries-1 {
			// 指数退避
			waitTime := g.baseRetryWait * time.Duration(1<<uint(attempt))
			time.Sleep(waitTime)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("AI生成失败，已重试%d次: %w", g.maxRetries, lastErr)
	}
	content.Book.Name = req.Name
	content.Ref.Name = req.Name
	// 保存到磁盘
	return &content.Book, g.saveContent(req, content, existingPlaybook)
}

// buildPrompt 构建AI提示词
func (g *Generator) buildPrompt(req GenerateBookRequest, playbookExists bool, _ *Playbook) string {
	if playbookExists {
		// Playbook已存在，只需生成book和ref
		return fmt.Sprintf(`你是一个诊断专家，需要根据用户描述生成诊断步骤。

用户描述: %s

你需要生成新的诊断方案(book)和引用(ref)，并使用 output_result 工具输出结果。

# 注意:
1. book.name 和 ref.name 必须相同
2. 每个步骤都要具体到执行命令，不能只有语言描述
3. kind字段当前仅支持: seq, branch
4. cases字段是可选的，只在有条件分支时使用
`, req.Description)
	}

	// Playbook不存在，需要生成playbook、book和ref
	return fmt.Sprintf(`你是一个诊断专家，需要根据用户描述生成诊断步骤。

用户描述: %s
Playbook名称: %s

该Playbook不存在，你需要:
1. 为这个诊断大类生成合适的描述和标签
2. 生成对应的诊断方案(book)和引用(ref)

使用 output_result 工具输出结果。

# 注意:
1. playbook.name必须是"%s",并且playbook.Desc是一个大的诊断方向，应该根据它的名称进行联想而不是当前方案的描述
2. book.name 和 ref.name 必须相同
3. 步骤要具体、可执行
4. kind字段当前仅支持: seq, branch
5. cases字段是可选的，只在有条件分支时使用
`, req.Description, req.PlaybookName, req.PlaybookName)
}

// callAIWithRetry 调用AI并处理工具调用
func (g *Generator) callAIWithRetry(ctx context.Context, prompt string, attempt int) (*GeneratedContent, error) {
	// 构建消息
	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	// 获取工具信息
	toolInfo, err := g.structOutputTool.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工具信息失败: %w", err)
	}

	// 调用AI，传入工具
	resp, err := g.llmClient.Generate(ctx, messages, model.WithTools([]*schema.ToolInfo{toolInfo}))
	if err != nil {
		return nil, fmt.Errorf("AI调用失败: %w", err)
	}

	// 检查是否有工具调用
	if len(resp.ToolCalls) == 0 {
		return nil, fmt.Errorf("AI未调用工具(尝试%d/%d), 响应内容: %s",
			attempt+1, g.maxRetries, resp.Content)
	}

	// 查找 output_result 工具调用
	var toolCall *schema.ToolCall
	for i := range resp.ToolCalls {
		if resp.ToolCalls[i].Function.Name == itool.StructOutputToolName {
			toolCall = &resp.ToolCalls[i]
			break
		}
	}

	if toolCall == nil {
		return nil, fmt.Errorf("未找到 output_result 工具调用(尝试%d/%d)",
			attempt+1, g.maxRetries)
	}

	// 执行工具调用
	toolResult, err := g.structOutputTool.InvokableRun(ctx, toolCall.Function.Arguments)
	if err != nil {
		return nil, fmt.Errorf("工具执行失败(尝试%d/%d): %w",
			attempt+1, g.maxRetries, err)
	}

	// 解析工具输出
	var toolOutput itool.StructuredOutputOutput
	if err := json.Unmarshal([]byte(toolResult), &toolOutput); err != nil {
		return nil, fmt.Errorf("解析工具输出失败(尝试%d/%d): %w",
			attempt+1, g.maxRetries, err)
	}

	// 检查工具调用状态
	if toolOutput.Status != 1 {
		return nil, fmt.Errorf("工具调用失败(尝试%d/%d): %s",
			attempt+1, g.maxRetries, toolOutput.Message)
	}

	// 将工具输出转换为 GeneratedContent
	var content GeneratedContent

	// 解析 book
	if bookData, ok := toolOutput.Data["book"]; ok {
		bookJSON, err := json.Marshal(bookData)
		if err != nil {
			return nil, fmt.Errorf("序列化book失败: %w", err)
		}
		if err := json.Unmarshal(bookJSON, &content.Book); err != nil {
			return nil, fmt.Errorf("解析book失败: %w", err)
		}
	} else {
		return nil, fmt.Errorf("缺少必需的book字段")
	}

	// 解析 ref
	if refData, ok := toolOutput.Data["ref"]; ok {
		refJSON, err := json.Marshal(refData)
		if err != nil {
			return nil, fmt.Errorf("序列化ref失败: %w", err)
		}
		if err := json.Unmarshal(refJSON, &content.Ref); err != nil {
			return nil, fmt.Errorf("解析ref失败: %w", err)
		}
	} else {
		return nil, fmt.Errorf("缺少必需的ref字段")
	}

	// 解析 playbook（可选）
	if playbookData, ok := toolOutput.Data["playbook"]; ok {
		playbookJSON, err := json.Marshal(playbookData)
		if err != nil {
			return nil, fmt.Errorf("序列化playbook失败: %w", err)
		}
		var playbook Playbook
		if err := json.Unmarshal(playbookJSON, &playbook); err != nil {
			return nil, fmt.Errorf("解析playbook失败: %w", err)
		}
		content.Playbook = &playbook
	}

	return &content, nil
}

// saveContent 保存生成的内容到磁盘
func (g *Generator) saveContent(req GenerateBookRequest, content *GeneratedContent, playbook *Playbook) error {
	// 如果playbook不存在，创建目录和introduction.yaml
	if playbook == nil {
		if content.Playbook == nil || content.Playbook.Name == "" || content.Playbook.Desc == "" {
			return fmt.Errorf("playbook不存在但AI未生成必需的playbook信息")
		}

		// 保存playbook
		if err := g.repo.SavePlaybook(content.Playbook); err != nil {
			return err
		}
	}

	// 保存book文件
	if err := g.repo.SaveBook(req.PlaybookName, &content.Book); err != nil {
		return err
	}

	// 更新introduction.yaml中的refs列表
	return g.repo.UpdatePlaybookRef(req.PlaybookName, content.Ref)
}
