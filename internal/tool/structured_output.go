package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	StructOutputToolName         = "output_result"
	structuredOutputBaseTemplate = `
结构化数据生成工具，用于生成符合预定义字段的JSON格式数据。

请严格按照以下字段要求生成JSON数据：
%s

注意：
1. 必须生成包含所有必填字段的JSON对象
2. 字段名必须与要求完全一致
3. 字段类型必须符合要求
4. 可选字段可以不提供，但必填字段必须存在
`
)

// FieldDefinition 定义字段的元信息
type FieldDefinition struct {
	Name        string            `json:"name"`        // 字段名
	Type        string            `json:"type"`        // 字段类型: string, number, boolean, array, object
	Description string            `json:"description"` // 字段描述
	Required    bool              `json:"required"`    // 是否必填
	Example     interface{}       `json:"example"`     // 示例值
	Properties  []FieldDefinition `json:"properties"`  // 嵌套字段定义（当Type为object时使用）
}

// StructuredOutputConfig 配置结构化输出工具
type StructuredOutputConfig struct {
	Description string            `json:"description"` // 工具用途描述
	Fields      []FieldDefinition `json:"fields"`      // 字段定义列表
}

// StructuredOutputInput 工具输入
type StructuredOutputInput struct {
	Data map[string]interface{} `json:"data"` // LLM生成的结构化数据
}

// StructuredOutputOutput 工具输出
type StructuredOutputOutput struct {
	Status        int                    `json:"status"`                   // 1: 成功, 2: 缺少字段
	Data          map[string]interface{} `json:"data"`                     // 解析后的数据
	MissingFields []string               `json:"missing_fields,omitempty"` // 缺少的必填字段
	Message       string                 `json:"message,omitempty"`        // 提示信息
}

// StructuredOutputTool 实现eino的InvokableTool接口
type StructuredOutputTool struct {
	config StructuredOutputConfig
}

var _ tool.InvokableTool = (*StructuredOutputTool)(nil)

// NewStructuredOutputTool 创建结构化输出工具
func NewStructuredOutputTool(config StructuredOutputConfig) *StructuredOutputTool {
	return &StructuredOutputTool{
		config: config,
	}
}

// Info 返回工具信息，根据配置动态生成描述
func (t *StructuredOutputTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	// 构建字段描述和参数
	var fieldDescriptions []string
	params := make(map[string]*schema.ParameterInfo)

	for _, field := range t.config.Fields {
		// 构建字段描述文本
		fieldDesc := t.buildFieldDescription(field, 0)
		fieldDescriptions = append(fieldDescriptions, fieldDesc)

		// 构建参数信息
		paramInfo := t.buildParameterInfo(field)
		params[field.Name] = paramInfo
	}

	// 生成完整的工具描述
	fieldsDesc := strings.Join(fieldDescriptions, "\n")
	fullDesc := fmt.Sprintf(structuredOutputBaseTemplate, fieldsDesc)

	if t.config.Description != "" {
		fullDesc = t.config.Description + "\n\n" + fullDesc
	}

	return &schema.ToolInfo{
		Name:        StructOutputToolName,
		Desc:        fullDesc,
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}, nil
}

// buildFieldDescription 递归构建字段描述
func (t *StructuredOutputTool) buildFieldDescription(field FieldDefinition, indent int) string {
	requiredText := "可选"
	if field.Required {
		requiredText = "必填"
	}

	exampleText := ""
	if field.Example != nil {
		exampleJSON, _ := json.Marshal(field.Example)
		exampleText = fmt.Sprintf("，示例: %s", string(exampleJSON))
	}

	indentStr := strings.Repeat("  ", indent)
	fieldDesc := fmt.Sprintf("%s- %s (%s, %s): %s%s",
		indentStr, field.Name, field.Type, requiredText, field.Description, exampleText)

	// 如果有嵌套字段，递归构建
	if len(field.Properties) > 0 {
		fieldDesc += "\n" + indentStr + "  属性:"
		for _, prop := range field.Properties {
			fieldDesc += "\n" + t.buildFieldDescription(prop, indent+2)
		}
	}

	return fieldDesc
}

// buildParameterInfo 递归构建参数信息
func (t *StructuredOutputTool) buildParameterInfo(field FieldDefinition) *schema.ParameterInfo {
	paramType := t.mapFieldTypeToSchemaType(field.Type)
	paramInfo := &schema.ParameterInfo{
		Desc:     field.Description,
		Type:     paramType,
		Required: field.Required,
	}

	// 如果是对象类型且有嵌套属性，构建嵌套参数
	if field.Type == "object" && len(field.Properties) > 0 {
		subParams := make(map[string]*schema.ParameterInfo)
		for _, prop := range field.Properties {
			subParams[prop.Name] = t.buildParameterInfo(prop)
		}
		paramInfo.SubParams = subParams
	}

	return paramInfo
}

// InvokableRun 执行工具调用
func (t *StructuredOutputTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 解析输入数据
	var data map[string]interface{}
	err := json.Unmarshal([]byte(argumentsInJSON), &data)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	// 校验必填字段
	var missingFields []string
	for _, field := range t.config.Fields {
		if field.Required {
			if _, exists := data[field.Name]; !exists {
				missingFields = append(missingFields, field.Name)
			}
		}
	}

	// 构建输出
	var output StructuredOutputOutput

	if len(missingFields) > 0 {
		// 缺少必填字段
		output = StructuredOutputOutput{
			Status:        2,
			Data:          data,
			MissingFields: missingFields,
			Message: fmt.Sprintf("缺少必填字段，请重新生成包含以下字段的完整数据: %s",
				strings.Join(missingFields, ", ")),
		}
	} else {
		// 所有必填字段都存在
		output = StructuredOutputOutput{
			Status:  1,
			Data:    data,
			Message: "成功解析结构化数据",
		}
	}

	jsonResult, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(jsonResult), nil
}

// mapFieldTypeToSchemaType 将字段类型映射到schema类型
func (t *StructuredOutputTool) mapFieldTypeToSchemaType(fieldType string) schema.DataType {
	switch strings.ToLower(fieldType) {
	case "string":
		return schema.String
	case "number", "integer", "float":
		return schema.Number
	case "boolean", "bool":
		return schema.Boolean
	case "array":
		return schema.Array
	case "object":
		return schema.Object
	default:
		return schema.String
	}
}

// Name 返回工具名称
func (t *StructuredOutputTool) Name() string {
	return StructOutputToolName
}

// Description 返回工具描述
func (t *StructuredOutputTool) Description() string {
	if t.config.Description != "" {
		return t.config.Description
	}
	return "生成结构化JSON数据"
}
