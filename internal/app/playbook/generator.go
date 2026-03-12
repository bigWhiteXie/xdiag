package playbook

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	itool "github.com/bigWhiteXie/xdiag/internal/tool"
	"github.com/bigWhiteXie/xdiag/pkg/formatter"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Generator 用于根据用户描述生成playbook和book
type Generator struct {
	repo                 Repo
	llmClient            model.ToolCallingChatModel
	maxRetries           int
	baseRetryWait        time.Duration
	playbookOutputTool   *itool.StructuredOutputTool
	bookAndRefOutputTool *itool.StructuredOutputTool
}

// NewGenerator 创建一个新的Generator实例
func NewGenerator(llmClient model.ToolCallingChatModel, playbooksDir string) *Generator {
	repo := NewRepo(playbooksDir)

	// 创建 playbook 生成工具
	playbookOutputTool := itool.NewStructuredOutputTool(itool.StructuredOutputConfig{
		Description: "用于输出生成的 playbook 信息",
		Fields: []itool.FieldDefinition{
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
		},
	})

	// 创建 book 和 ref 生成工具
	bookAndRefOutputTool := itool.NewStructuredOutputTool(itool.StructuredOutputConfig{
		Description: "用于输出生成的诊断方案 book 和 ref 信息",
		Fields: []itool.FieldDefinition{
			{
				Name:        "name",
				Type:        "string",
				Description: "book名称",
				Required:    true,
			},
			{
				Name:        "steps",
				Type:        "array",
				Description: "诊断步骤列表",
				Required:    true,
				Properties: []itool.FieldDefinition{
					{
						Name:        "kind",
						Type:        "string",
						Description: "步骤类型，支持: seq(顺序执行), branch(条件分支)",
						Required:    true,
					},
					{
						Name:        "desc",
						Type:        "string",
						Description: "步骤描述或执行命令",
						Required:    true,
					},
					{
						Name:        "cases",
						Type:        "array",
						Description: "条件分支列表（仅当kind为branch时使用）",
						Required:    false,
						Properties: []itool.FieldDefinition{
							{
								Name:        "case",
								Type:        "string",
								Description: "分支条件描述",
								Required:    true,
							},
							{
								Name:        "steps",
								Type:        "array",
								Description: "该分支下的步骤列表",
								Required:    true,
							},
						},
					},
				},
			},
			{
				Name:        "desc",
				Type:        "string",
				Description: "ref简短描述",
				Required:    true,
			},
			{
				Name:        "log",
				Type:        "string",
				Description: "相关日志路径或说明",
				Required:    false,
			},
		},
	})

	return &Generator{
		repo:                 repo,
		llmClient:            llmClient,
		maxRetries:           3,
		baseRetryWait:        1 * time.Second,
		playbookOutputTool:   playbookOutputTool,
		bookAndRefOutputTool: bookAndRefOutputTool,
	}
}

// GenerateBookRequest 生成book的请求参数
type GenerateBookRequest struct {
	Name         string
	PlaybookName string // playbook名称(大类)
	Description  string // 用户对诊断步骤的描述
}

// GeneratedPlaybook AI生成的playbook内容
type GeneratedPlaybook struct {
	Name string `json:"name"` // playbook名称
	Desc string `json:"desc"` // playbook描述
}

// GeneratedBookAndRef AI生成的book和ref内容
type GeneratedBookAndRef struct {
	Name  string `json:"name"`  // book名称
	Steps []Step `json:"steps"` // 诊断步骤列表
	Desc  string `json:"desc"`  // ref简短描述
	Log   string `json:"log"`   // 相关日志路径或说明
}

// GenerateAndSave 根据用户描述生成book并落盘
func (g *Generator) GenerateAndSave(ctx context.Context, req GenerateBookRequest, showDetails bool) (*Book, error) {
	// 先检查playbook是否存在
	playbookExists := g.repo.PlaybookExists(req.PlaybookName)

	var existingPlaybook *Playbook
	var err error

	// 如果playbook不存在，先生成playbook
	if !playbookExists {
		generatedPlaybook, err := g.generatePlaybook(ctx, req, showDetails)
		if err != nil {
			return nil, fmt.Errorf("生成playbook失败: %w", err)
		}

		// 构建完整的playbook对象
		existingPlaybook = &Playbook{
			Name: generatedPlaybook.Name,
			Desc: generatedPlaybook.Desc,
		}

		// 保存playbook
		if err := g.repo.SavePlaybook(existingPlaybook); err != nil {
			return nil, fmt.Errorf("保存playbook失败: %w", err)
		}
	} else {
		existingPlaybook, err = g.repo.LoadPlaybook(req.PlaybookName)
		if err != nil {
			return nil, fmt.Errorf("加载playbook失败: %w", err)
		}
	}

	// 生成book和ref
	bookAndRef, err := g.generateBookAndRef(ctx, req, showDetails)
	if err != nil {
		return nil, fmt.Errorf("生成book和ref失败: %w", err)
	}

	// 构建完整的book和ref对象
	book := &Book{
		Name:  bookAndRef.Name,
		Steps: bookAndRef.Steps,
	}

	ref := Ref{
		Name: bookAndRef.Name,
		Desc: bookAndRef.Desc,
		Log:  bookAndRef.Log,
	}

	// 保存book文件
	if err := g.repo.SaveBook(req.PlaybookName, book); err != nil {
		return nil, fmt.Errorf("保存book失败: %w", err)
	}

	// 更新introduction.yaml中的refs列表
	if err := g.repo.UpdatePlaybookRef(req.PlaybookName, ref); err != nil {
		return nil, fmt.Errorf("更新playbook ref失败: %w", err)
	}

	return book, nil
}

// generatePlaybook 生成playbook信息
func (g *Generator) generatePlaybook(ctx context.Context, req GenerateBookRequest, showDetails bool) (*GeneratedPlaybook, error) {
	prompt := fmt.Sprintf(`你是一个诊断专家，需要为一个新的诊断大类生成playbook描述。

用户描述: %s
Playbook名称: %s

请根据playbook名称生成一个简短的描述（一句话即可）。注意：
1. desc应该是一个大的诊断方向，根据名称进行联想，而不是当前具体方案的描述
2. 直接输出描述文本，不要使用任何标签或格式

示例：
- 如果playbook名称是"network"，输出："网络相关的诊断方案"
- 如果playbook名称是"database"，输出："数据库相关的诊断方案"
- 如果playbook名称是"performance"，输出："性能相关的诊断方案"
`, req.Description, req.PlaybookName)

	agentFormatter := formatter.NewAgentFormatter(showDetails)
	agentFormatter.FormatLLMCall(prompt)

	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	// 调用AI生成描述（不使用工具）
	var desc string
	var lastErr error
	for attempt := 0; attempt < g.maxRetries; attempt++ {
		resp, err := g.llmClient.Generate(ctx, messages)
		if err != nil {
			lastErr = fmt.Errorf("AI调用失败: %w", err)
			if attempt < g.maxRetries-1 {
				waitTime := g.baseRetryWait * time.Duration(1<<uint(attempt))
				time.Sleep(waitTime)
			}
			continue
		}

		agentFormatter.FormatLLMResponse(resp.Content, false)

		// 过滤掉 <think> 等标签，提取纯文本
		desc = filterThinkTags(resp.Content)
		if desc == "" {
			lastErr = fmt.Errorf("AI返回空描述")
			if attempt < g.maxRetries-1 {
				waitTime := g.baseRetryWait * time.Duration(1<<uint(attempt))
				time.Sleep(waitTime)
			}
			continue
		}

		lastErr = nil
		break
	}

	if lastErr != nil {
		return nil, fmt.Errorf("AI生成playbook失败，已重试%d次: %w", g.maxRetries, lastErr)
	}

	return &GeneratedPlaybook{
		Name: req.PlaybookName,
		Desc: desc,
	}, nil
}

// generateBookAndRef 生成book和ref信息
func (g *Generator) generateBookAndRef(ctx context.Context, req GenerateBookRequest, showDetails bool) (*GeneratedBookAndRef, error) {
	prompt := fmt.Sprintf(`你是一个诊断专家，需要根据用户描述生成具体的诊断步骤。

用户描述: %s

你需要生成诊断方案(book)和引用(ref)。使用 output_result 工具即可输出结构化数据。

# 注意:
1. name字段是book和ref共用的名称
2. steps是具体的诊断步骤列表，每个步骤都要具体到执行命令，不能只有语言描述
3. desc是对该诊断方案的简短描述
4. log是相关日志路径或说明（可选）
5. kind字段当前仅支持: seq, branch
6. cases字段是可选的，只在有条件分支时使用

# 工具调用示例:

示例1 - 简单顺序步骤（无分支）:
{
  "name": "check_disk_space",
  "steps": [
    {
      "kind": "seq",
      "desc": "df -h"
    },
    {
      "kind": "seq",
      "desc": "du -sh /var/log"
    }
  ],
  "desc": "检查磁盘空间使用情况",
  "log": "/var/log/disk.log"
}

示例2 - 包含条件分支:
{
  "name": "diagnose_network",
  "steps": [
    {
      "kind": "seq",
      "desc": "ping -c 4 8.8.8.8"
    },
    {
      "kind": "branch",
      "desc": "根据ping结果判断",
      "cases": [
        {
          "case": "ping通",
          "steps": [
            {
              "kind": "seq",
              "desc": "curl -I https://www.google.com"
            }
          ]
        },
        {
          "case": "ping不通",
          "steps": [
            {
              "kind": "seq",
              "desc": "ip route show"
            },
            {
              "kind": "seq",
              "desc": "cat /etc/resolv.conf"
            }
          ]
        }
      ]
    }
  ],
  "desc": "网络连通性诊断",
  "log": "/var/log/network.log"
}

示例3 - 复杂嵌套分支:
{
  "name": "service_health_check",
  "steps": [
    {
      "kind": "seq",
      "desc": "systemctl status nginx"
    },
    {
      "kind": "branch",
      "desc": "根据服务状态判断",
      "cases": [
        {
          "case": "服务运行中",
          "steps": [
            {
              "kind": "seq",
              "desc": "curl -I http://localhost:80"
            },
            {
              "kind": "branch",
              "desc": "根据HTTP响应判断",
              "cases": [
                {
                  "case": "返回200",
                  "steps": [
                    {
                      "kind": "seq",
                      "desc": "tail -n 50 /var/log/nginx/access.log"
                    }
                  ]
                },
                {
                  "case": "返回错误",
                  "steps": [
                    {
                      "kind": "seq",
                      "desc": "tail -n 50 /var/log/nginx/error.log"
                    },
                    {
                      "kind": "seq",
                      "desc": "nginx -t"
                    }
                  ]
                }
              ]
            }
          ]
        },
        {
          "case": "服务未运行",
          "steps": [
            {
              "kind": "seq",
              "desc": "journalctl -u nginx -n 50"
            },
            {
              "kind": "seq",
              "desc": "systemctl start nginx"
            }
          ]
        }
      ]
    }
  ],
  "desc": "Nginx服务健康检查",
  "log": "/var/log/nginx/error.log"
}

请严格按照以上示例格式调用 output_result 工具。
`, req.Description)

	// 调用AI生成book和ref
	var result *GeneratedBookAndRef
	var lastErr error
	for attempt := 0; attempt < g.maxRetries; attempt++ {
		result, lastErr = g.callAIForBookAndRef(ctx, prompt, attempt, showDetails)
		if lastErr == nil {
			break
		}

		if attempt < g.maxRetries-1 {
			waitTime := g.baseRetryWait * time.Duration(1<<uint(attempt))
			time.Sleep(waitTime)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("AI生成book和ref失败，已重试%d次: %w", g.maxRetries, lastErr)
	}

	return result, nil
}

// callAIForPlaybook 调用AI生成playbook
func (g *Generator) callAIForPlaybook(ctx context.Context, prompt string, attempt int, showDetails bool) (*GeneratedPlaybook, error) {
	agentFormatter := formatter.NewAgentFormatter(showDetails)

	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	toolInfo, err := g.playbookOutputTool.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工具信息失败: %w", err)
	}

	maxRounds := 5
	for round := 0; round < maxRounds; round++ {
		if round == 0 {
			agentFormatter.FormatLLMCall(prompt)
		}

		resp, err := g.llmClient.Generate(ctx, messages, model.WithTools([]*schema.ToolInfo{toolInfo}))
		if err != nil {
			return nil, fmt.Errorf("AI调用失败: %w", err)
		}

		agentFormatter.FormatLLMResponse(resp.Content, len(resp.ToolCalls) > 0)

		if len(resp.ToolCalls) == 0 {
			agentFormatter.FormatThinking(resp.Content)

			if round < maxRounds-1 {
				messages = append(messages, schema.AssistantMessage(resp.Content, nil))
				messages = append(messages, schema.UserMessage("请使用 output_result 工具输出结构化数据，不要只进行文字描述。"))
				continue
			}
			return nil, fmt.Errorf("AI在%d轮对话后仍未调用工具(尝试%d/%d)", maxRounds, attempt+1, g.maxRetries)
		}

		var toolCall *schema.ToolCall
		for i := range resp.ToolCalls {
			if resp.ToolCalls[i].Function.Name == itool.StructOutputToolName {
				toolCall = &resp.ToolCalls[i]
				break
			}
		}

		if toolCall == nil {
			return nil, fmt.Errorf("未找到 output_result 工具调用(尝试%d/%d)", attempt+1, g.maxRetries)
		}

		agentFormatter.FormatToolCall(toolCall.Function.Name, toolCall.Function.Arguments)

		toolResult, err := g.playbookOutputTool.InvokableRun(ctx, toolCall.Function.Arguments)
		if err != nil {
			return nil, fmt.Errorf("工具执行失败(尝试%d/%d): %w", attempt+1, g.maxRetries, err)
		}

		agentFormatter.FormatToolResult(toolResult)

		var toolOutput itool.StructuredOutputOutput
		if err := json.Unmarshal([]byte(toolResult), &toolOutput); err != nil {
			return nil, fmt.Errorf("解析工具输出失败(尝试%d/%d): %w", attempt+1, g.maxRetries, err)
		}

		if toolOutput.Status != 1 {
			messages = append(messages,
				schema.AssistantMessage(resp.Content, resp.ToolCalls),
				schema.ToolMessage(toolResult, toolCall.ID))
			continue
		}

		// 解析playbook数据
		var playbook GeneratedPlaybook
		nameData, hasName := toolOutput.Data["name"]
		descData, hasDesc := toolOutput.Data["desc"]

		if !hasName || !hasDesc {
			return nil, fmt.Errorf("缺少必需的name或desc字段")
		}

		if name, ok := nameData.(string); ok {
			playbook.Name = name
		} else {
			return nil, fmt.Errorf("name字段类型错误")
		}

		if desc, ok := descData.(string); ok {
			playbook.Desc = desc
		} else {
			return nil, fmt.Errorf("desc字段类型错误")
		}

		return &playbook, nil
	}

	return nil, fmt.Errorf("超过最大循环次数%d(尝试%d/%d)", maxRounds, attempt+1, g.maxRetries)
}

// callAIForBookAndRef 调用AI生成book和ref
func (g *Generator) callAIForBookAndRef(ctx context.Context, prompt string, attempt int, showDetails bool) (*GeneratedBookAndRef, error) {
	agentFormatter := formatter.NewAgentFormatter(showDetails)

	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	toolInfo, err := g.bookAndRefOutputTool.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工具信息失败: %w", err)
	}

	maxRounds := 5
	for round := 0; round < maxRounds; round++ {
		if round == 0 {
			agentFormatter.FormatLLMCall(prompt)
		}

		resp, err := g.llmClient.Generate(ctx, messages, model.WithTools([]*schema.ToolInfo{toolInfo}))
		if err != nil {
			return nil, fmt.Errorf("AI调用失败: %w", err)
		}

		agentFormatter.FormatLLMResponse(resp.Content, len(resp.ToolCalls) > 0)

		if len(resp.ToolCalls) == 0 {
			agentFormatter.FormatThinking(resp.Content)

			if round < maxRounds-1 {
				messages = append(messages, schema.AssistantMessage(resp.Content, nil))
				messages = append(messages, schema.UserMessage("请使用 output_result 工具输出结构化数据，不要只进行文字描述。"))
				continue
			}
			return nil, fmt.Errorf("AI在%d轮对话后仍未调用工具(尝试%d/%d)", maxRounds, attempt+1, g.maxRetries)
		}

		var toolCall *schema.ToolCall
		for i := range resp.ToolCalls {
			if resp.ToolCalls[i].Function.Name == itool.StructOutputToolName {
				toolCall = &resp.ToolCalls[i]
				break
			}
		}

		if toolCall == nil {
			return nil, fmt.Errorf("未找到 output_result 工具调用(尝试%d/%d)", attempt+1, g.maxRetries)
		}

		agentFormatter.FormatToolCall(toolCall.Function.Name, toolCall.Function.Arguments)

		toolResult, err := g.bookAndRefOutputTool.InvokableRun(ctx, toolCall.Function.Arguments)
		if err != nil {
			return nil, fmt.Errorf("工具执行失败(尝试%d/%d): %w", attempt+1, g.maxRetries, err)
		}

		agentFormatter.FormatToolResult(toolResult)

		var toolOutput itool.StructuredOutputOutput
		if err := json.Unmarshal([]byte(toolResult), &toolOutput); err != nil {
			return nil, fmt.Errorf("解析工具输出失败(尝试%d/%d): %w", attempt+1, g.maxRetries, err)
		}

		if toolOutput.Status != 1 {
			messages = append(messages,
				schema.AssistantMessage(resp.Content, resp.ToolCalls),
				schema.ToolMessage(toolResult, toolCall.ID))
			continue
		}

		// 解析book和ref数据
		var result GeneratedBookAndRef

		nameData, hasName := toolOutput.Data["name"]
		stepsData, hasSteps := toolOutput.Data["steps"]
		descData, hasDesc := toolOutput.Data["desc"]

		if !hasName || !hasSteps || !hasDesc {
			return nil, fmt.Errorf("缺少必需的字段")
		}

		if name, ok := nameData.(string); ok {
			result.Name = name
		} else {
			return nil, fmt.Errorf("name字段类型错误")
		}

		if desc, ok := descData.(string); ok {
			result.Desc = desc
		} else {
			return nil, fmt.Errorf("desc字段类型错误")
		}

		// 解析steps
		stepsJSON, err := json.Marshal(stepsData)
		if err != nil {
			return nil, fmt.Errorf("序列化steps失败: %w", err)
		}
		if err := json.Unmarshal(stepsJSON, &result.Steps); err != nil {
			return nil, fmt.Errorf("解析steps失败: %w", err)
		}

		// 解析log（可选）
		if logData, hasLog := toolOutput.Data["log"]; hasLog {
			if log, ok := logData.(string); ok {
				result.Log = log
			}
		}

		return &result, nil
	}

	return nil, fmt.Errorf("超过最大循环次数%d(尝试%d/%d)", maxRounds, attempt+1, g.maxRetries)
}

// filterThinkTags 过滤掉 <think> 等标签，提取纯文本内容
func filterThinkTags(content string) string {
	// 移除 <think>...</think> 标签及其内容
	re := regexp.MustCompile(`(?s)<think>.*?</think>`)
	result := re.ReplaceAllString(content, "")

	// 移除其他可能的 XML 标签
	re = regexp.MustCompile(`<[^>]+>`)
	result = re.ReplaceAllString(result, "")

	// 去除首尾空白字符
	result = strings.TrimSpace(result)

	return result
}
