package utils

import "strings"

func ParseJsonByLabel(label string, content string) string {
	startTag := "<" + label + ">"
	endTag := "</" + label + ">"
	
	startIdx := strings.Index(content, startTag)
	if startIdx == -1 {
		return content // 如果标签不存在，直接返回原字符串
	}
	
	startIdx += len(startTag)
	endIdx := strings.Index(content[startIdx:], endTag)
	if endIdx == -1 {
		return content // 如果只有开始标签没有结束标签，也返回原字符串
	}
	
	endIdx += startIdx
	return strings.TrimSpace(content[startIdx:endIdx])
}