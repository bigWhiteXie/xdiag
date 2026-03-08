package playbook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bigWhiteXie/xdiag/pkg/utils"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Generator 用于根据用户描述生成playbook和book
type Generator struct {
	repo          Repo
	llmClient     model.ToolCallingChatModel
	maxRetries    int
	baseRetryWait time.Duration
}

// NewGenerator 创建一个新的Generator实例
func NewGenerator(llmClient model.ToolCallingChatModel, playbooksDir string) *Generator {
	repo := NewRepo(playbooksDir)
	return &Generator{
		repo:          repo,
		llmClient:     llmClient,
		maxRetries:    3,
		baseRetryWait: 1 * time.Second,
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

你需要生成新的诊断方案(book)和引用(ref)。

请按如下格式进行返回:
<output>
{
  "book": {
    "steps": [
      {
        "kind": "步骤类型",
        "desc": "步骤描述，需要包含具体操作(如shell命令、工具调用等等)不能只有语言描述",
        "cases": [
          {
            "case": "条件描述",
            "steps": [...]
          }
        ]
      }
    ]
  },
  "ref": {
    "desc": "简短描述",
    "log": "相关日志路径或说明，如果用户没有声明日志则不写"
  }
}
</output>

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

请按如下格式进行返回:
<output>
{
  "playbook": {
    "name": "%s",
    "desc": "playbook描述",
    "required_tags": ["tag1", "tag2"]
  },
  "book": {
    "steps": [
      {
        "kind": "步骤类型",
        "desc": "步骤描述",
        "cases": [  // 当kind为branch时才声明cases字段
          {
            "case": "条件描述",
            "steps": [...]
          }
        ]
      }
    ]
  },
  "ref": {
    "desc": "简短描述",
    "log": "相关日志路径或说明，若用户未描述方案相关日志则不写"
  }
}
</output>

# 注意:
1. playbook.name必须是"%s",并且playbook.Desc是一个大的诊断方向，应该根据它的名称进行联想而不是当前方案的描述
2. book.name 和 ref.name 必须相同
3. 步骤要具体、可执行
4. kind字段当前仅支持: seq, branch
5. cases字段是可选的，只在有条件分支时使用
`, req.Description, req.PlaybookName, req.PlaybookName, req.PlaybookName)
}

// callAIWithRetry 调用AI并处理JSON反序列化
func (g *Generator) callAIWithRetry(ctx context.Context, prompt string, attempt int) (*GeneratedContent, error) {
	// 构建消息
	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	// 调用AI
	resp, err := g.llmClient.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("AI调用失败: %w", err)
	}

	// 提取响应内容
	responseText := resp.Content

	// 尝试解析JSON
	var content GeneratedContent
	responseText = utils.ParseJsonByLabel("output", responseText)
	err = json.Unmarshal([]byte(responseText), &content)
	if err != nil {
		return nil, fmt.Errorf("JSON反序列化失败(尝试%d/%d): %w, 响应内容: %s",
			attempt+1, g.maxRetries, err, responseText)
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
