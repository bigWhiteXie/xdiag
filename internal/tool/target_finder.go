package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bigWhiteXie/xdiag/internal/app/targets"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
)

const (
	finderTemplate = `
根据过滤器查找系统中的目标资产，如服务器、数据库等。过滤器格式为 "字段 操作符 值[, 字段 操作符 值]"。
		
支持的字段包括:
- name: 目标名称
- ip/address: IP地址
- kind/type: 目标类型(如node, mysql, redis等)
- tag: 标签

支持的操作符包括:
- eq: 等于
- like: 包含	
`
)

// TargetFinderToolInput 定义TargetFinderTool的输入
type TargetFinderToolInput struct {
	Filters string `json:"filters"`
}

// TargetFinderToolOutput 定义TargetFinderTool的输出
type TargetFinderToolOutput struct {
	Targets []targets.Target `json:"targets"`
	Error   string           `json:"error,omitempty"`
}

// TargetFinderTool 实现eino的Tool接口，用于根据用户查询查找目标
type TargetFinderTool struct {
	targetRepo targets.Repo
}

var _ tool.InvokableTool = (*TargetFinderTool)(nil)

// NewTargetFinderTool 创建一个新的TargetFinderTool实例
func NewTargetFinderTool(targetRepo targets.Repo) *TargetFinderTool {
	return &TargetFinderTool{
		targetRepo: targetRepo,
	}
}

// parseFilters 解析过滤器字符串
func (t *TargetFinderTool) parseFilters(filtersStr string) map[string]targets.Op {
	filters := make(map[string]targets.Op)

	// 按逗号分割多个过滤器
	parts := strings.Split(filtersStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// 查找操作符
		var op string
		var found bool

		// 支持的操作符
		operators := []string{" eq ", " ne ", " like ", " gt ", " lt ", " ge ", " le "}

		for _, operator := range operators {
			if strings.Contains(part, operator) {
				op = operator
				found = true
				break
			}
		}

		if !found {
			continue
		}

		// 分割字段名和值
		fieldAndValue := strings.Split(part, op)
		if len(fieldAndValue) != 2 {
			continue
		}

		field := strings.TrimSpace(fieldAndValue[0])
		value := strings.TrimSpace(fieldAndValue[1])

		// 移除值两端的引号（如果有的话）
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		// 根据字段和操作符处理过滤条件
		switch field {
		case "name":
			fallthrough
		case "ip":
			fallthrough
		case "address":
			fallthrough
		case "kind":
			fallthrough
		case "type":
			fallthrough
		case "tag":
			opType := strings.TrimSpace(op)
			filters[field] = targets.Op{Op: opType, Val: value}
		}
	}

	return filters
}

// Info 返回工具信息
func (t *TargetFinderTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "target_finder",
		Desc: finderTemplate,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"filters": &schema.ParameterInfo{
				Desc: `
目标过滤条件,示例:
- "name eq myserver" - 查找名称为myserver的目标
- "ip eq 10.131.135.26" - 查找IP为10.131.135.26的目标
- "name like xos-no, kind eq server" - 查找名称包含xos-no且类型为server的目标
- "tag eq production" - 查找标记为production的目标`,
				Type: schema.String,
			},
		}),
	}, nil
}

// InvokableRun 执行工具调用，实现InvokableTool接口
func (t *TargetFinderTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var input TargetFinderToolInput
	err := json.Unmarshal([]byte(argumentsInJSON), &input)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	// 解析过滤器
	filters := t.parseFilters(input.Filters)

	// 获取匹配的目标
	results, err := t.targetRepo.List(ctx, filters)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "暂无数据", nil
		}
		return "", fmt.Errorf("failed to list targets: %w", err)
	}

	output := TargetFinderToolOutput{
		Targets: make([]targets.Target, len(results)),
	}

	for i, target := range results {
		output.Targets[i] = *target
	}

	jsonResult, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(jsonResult), nil
}

// Name 返回工具名称
func (t *TargetFinderTool) Name() string {
	return "target_finder"
}
