package match

import (
	"fmt"
	"strings"

	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
)

// DescriptionBuilder 描述构建器
type DescriptionBuilder struct{}

// NewDescriptionBuilder 创建描述构建器
func NewDescriptionBuilder() *DescriptionBuilder {
	return &DescriptionBuilder{}
}

// BuildPlaybooksDescription 构建 playbooks 的描述
func (b *DescriptionBuilder) BuildPlaybooksDescription(playbooks []playbook.Playbook) string {
	var sb strings.Builder
	for i, pb := range playbooks {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, pb.Name))
		sb.WriteString(fmt.Sprintf("   描述: %s\n", pb.Desc))
		if len(pb.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("   标签: %s\n", strings.Join(pb.Tags, ", ")))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// BuildRefsDescription 构建 refs 的描述
func (b *DescriptionBuilder) BuildRefsDescription(refs []playbook.Ref) string {
	var sb strings.Builder
	for i, ref := range refs {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, ref.Name))
		sb.WriteString(fmt.Sprintf("   描述: %s\n", ref.Desc))
		if ref.Log != "" {
			sb.WriteString(fmt.Sprintf("   日志: %s\n", ref.Log))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// FilterExcludedPlaybooks 过滤掉已排除的 playbooks
func (b *DescriptionBuilder) FilterExcludedPlaybooks(
	allPlaybooks []playbook.Playbook,
	excludedPlaybooks []string,
) []playbook.Playbook {
	available := make([]playbook.Playbook, 0, len(allPlaybooks))
	excludedMap := make(map[string]bool)
	for _, name := range excludedPlaybooks {
		excludedMap[name] = true
	}

	for _, pb := range allPlaybooks {
		if !excludedMap[pb.Name] {
			available = append(available, pb)
		}
	}
	return available
}
