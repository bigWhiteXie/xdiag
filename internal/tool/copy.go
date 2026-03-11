package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/bigWhiteXie/xdiag/internal/app/targets"
	"github.com/bigWhiteXie/xdiag/internal/config"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	copyTemplate = `
文件拷贝工具，将本地脚本文件拷贝到远程节点。

参数说明:
- ip: 目标节点IP地址(必选)
- script_name: 脚本文件名(必选)
- dest_path: 目标路径(可选，默认/root/)
`
)

// CopyToolInput 定义CopyTool的输入
type CopyToolInput struct {
	IP         string `json:"ip"`                  // 必选，目标节点IP
	ScriptName string `json:"script_name"`         // 必选，脚本文件名
	DestPath   string `json:"dest_path,omitempty"` // 可选，目标路径，默认/root/
}

// CopyToolOutput 定义CopyTool的输出
type CopyToolOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CopyTool 实现eino的InvokableTool接口，用于拷贝文件到远程节点
type CopyTool struct {
	targetRepo targets.Repo
}

var _ tool.InvokableTool = (*CopyTool)(nil)

// NewCopyTool 创建一个新的CopyTool实例
func NewCopyTool(targetRepo targets.Repo) *CopyTool {
	return &CopyTool{
		targetRepo: targetRepo,
	}
}

// Info 返回工具信息
func (t *CopyTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "file_copy",
		Desc: copyTemplate,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"ip": {
				Desc:     "目标节点IP地址",
				Type:     schema.String,
				Required: true,
			},
			"script_name": {
				Desc:     "脚本文件名",
				Type:     schema.String,
				Required: true,
			},
			"dest_path": {
				Desc:     "目标路径，默认/root/",
				Type:     schema.String,
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行工具调用，实现InvokableTool接口
func (t *CopyTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var input CopyToolInput
	err := json.Unmarshal([]byte(argumentsInJSON), &input)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	// 验证必选参数
	if input.IP == "" {
		return "", fmt.Errorf("ip is required")
	}
	if input.ScriptName == "" {
		return "", fmt.Errorf("script_name is required")
	}

	// 设置默认目标路径
	if input.DestPath == "" {
		input.DestPath = "/root/"
	}

	// 构建本地脚本路径
	configDir := config.GetConfigDir()
	scriptPath := filepath.Join(configDir, "scripts", input.ScriptName)

	// 检查脚本文件是否存在
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		output := CopyToolOutput{
			Success: false,
			Error:   fmt.Sprintf("script file not found: %s", scriptPath),
		}
		jsonResult, _ := json.Marshal(output)
		return string(jsonResult), nil
	}

	// 从target repo中查找对应的node类型target
	filters := map[string]targets.Op{
		"address": {Op: "eq", Val: input.IP},
		"kind":    {Op: "eq", Val: targets.TargetKindNode},
	}

	targetList, err := t.targetRepo.List(ctx, filters)
	if err != nil {
		output := CopyToolOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to find target: %v", err),
		}
		jsonResult, _ := json.Marshal(output)
		return string(jsonResult), nil
	}

	if len(targetList) == 0 {
		output := CopyToolOutput{
			Success: false,
			Error:   fmt.Sprintf("no node target found for IP %s", input.IP),
		}
		jsonResult, _ := json.Marshal(output)
		return string(jsonResult), nil
	}

	target := targetList[0]

	// 执行文件拷贝
	output := t.copyFile(ctx, target, scriptPath, input.DestPath)

	jsonResult, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(jsonResult), nil
}

// copyFile 通过SFTP拷贝文件到远程节点
func (t *CopyTool) copyFile(ctx context.Context, target *targets.Target, localPath, remotePath string) CopyToolOutput {
	// 建立SSH连接
	config := &ssh.ClientConfig{
		User:            target.Username,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         60 * time.Second,
	}

	// 优先使用密码认证
	if target.Password != "" {
		config.Auth = append(config.Auth, ssh.Password(target.Password))
	}

	// 如果有SSH key，也添加key认证
	if target.Node != "" {
		signer, err := ssh.ParsePrivateKey([]byte(target.Node))
		if err == nil {
			config.Auth = append(config.Auth, ssh.PublicKeys(signer))
		}
	}

	if len(config.Auth) == 0 {
		return CopyToolOutput{
			Success: false,
			Error:   "no authentication method available (no password or ssh key)",
		}
	}

	// 连接SSH
	addr := fmt.Sprintf("%s:%d", target.Address, target.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return CopyToolOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to connect to %s: %v", addr, err),
		}
	}
	defer client.Close()

	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return CopyToolOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to create sftp client: %v", err),
		}
	}
	defer sftpClient.Close()

	// 打开本地文件
	localFile, err := os.Open(localPath)
	if err != nil {
		return CopyToolOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to open local file: %v", err),
		}
	}
	defer localFile.Close()

	// 创建远程文件
	dstFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return CopyToolOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to create remote file %s: %v", remotePath, err),
		}
	}
	defer dstFile.Close()

	// 拷贝文件内容
	_, err = io.Copy(dstFile, localFile)
	if err != nil {
		return CopyToolOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to copy file content: %v", err),
		}
	}

	return CopyToolOutput{
		Success: true,
		Message: fmt.Sprintf("successfully copied %s to %s:%s", localPath, target.Address, remotePath),
	}
}

// Name 返回工具名称
func (t *CopyTool) Name() string {
	return "file_copy"
}

// Description 返回工具描述
func (t *CopyTool) Description() string {
	return "拷贝本地脚本文件到远程节点"
}
