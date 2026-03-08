package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/bigWhiteXie/xdiag/internal/app/targets"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"golang.org/x/crypto/ssh"
)

const (
	execTemplate = `
执行shell命令工具，支持本地和远程执行。

参数说明:
- username: 目标用户名(可选),默认root
- address: 目标地址(可选，格式为ip:port，不传则本地执行)
- path: 命令执行路径(可选)
- cmd: 要执行的命令(必选)
- expire: 超时时间，单位秒(可选，默认30秒)
- ignore_err: 是否忽略错误(可选，默认false)
`
)

// ExecToolInput 定义ExecTool的输入
type ExecToolInput struct {
	Username  string `json:"username,omitempty"`
	Address   string `json:"address,omitempty"`    // 可选，格式为ip:port
	Path      string `json:"path,omitempty"`       // 可选，命令执行路径
	Cmd       string `json:"cmd"`                  // 必选，要执行的命令
	Expire    int    `json:"expire,omitempty"`     // 可选，超时时间(秒)，默认30
	IgnoreErr bool   `json:"ignore_err,omitempty"` // 可选，是否忽略错误，默认false
}

// ExecToolOutput 定义ExecTool的输出
type ExecToolOutput struct {
	Success bool   `json:"success"`
	Stdout  string `json:"stdout,omitempty"`
	Stderr  string `json:"stderr,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ExecTool 实现eino的InvokableTool接口，用于执行shell命令
type ExecTool struct {
	targetRepo targets.Repo
}

var _ tool.InvokableTool = (*ExecTool)(nil)

// NewExecTool 创建一个新的ExecTool实例
func NewExecTool(targetRepo targets.Repo) *ExecTool {
	return &ExecTool{
		targetRepo: targetRepo,
	}
}

// Info 返回工具信息
func (t *ExecTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "shell_exec",
		Desc: execTemplate,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"address": {
				Desc:     "目标地址，格式为ip:port，不传则本地执行",
				Type:     schema.String,
				Required: false,
			},
			"path": {
				Desc:     "命令执行的基础路径",
				Type:     schema.String,
				Required: false,
			},
			"cmd": {
				Desc:     "要执行的shell命令",
				Type:     schema.String,
				Required: true,
			},
			"expire": {
				Desc:     "超时时间(秒)，默认30秒",
				Type:     schema.Integer,
				Required: false,
			},
			"ignore_err": {
				Desc:     "是否忽略错误，默认false",
				Type:     schema.Boolean,
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行工具调用，实现InvokableTool接口
func (t *ExecTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var input ExecToolInput
	err := json.Unmarshal([]byte(argumentsInJSON), &input)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	// 验证必选参数
	if input.Cmd == "" {
		return "", fmt.Errorf("cmd is required")
	}

	// 设置默认超时时间
	if input.Expire <= 0 {
		input.Expire = 30
	}

	var output ExecToolOutput

	// 根据是否有address决定本地还是远程执行
	if input.Address == "" {
		// 本地执行
		output = t.executeLocal(ctx, input)
	} else {
		// 远程执行
		output = t.executeRemote(ctx, input)
	}

	// 如果ignore_err为true，即使有错误也返回成功
	if input.IgnoreErr && (output.Error != "" || output.Stderr != "") {
		output.Success = true
		output.Stderr = ""
		output.Error = ""
		output.Stdout = "完成执行，忽略报错信息"
	}

	jsonResult, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(jsonResult), nil
}

// executeLocal 本地执行命令
func (t *ExecTool) executeLocal(ctx context.Context, input ExecToolInput) ExecToolOutput {
	// 创建超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(input.Expire)*time.Second)
	defer cancel()

	// 构建命令
	var cmd *exec.Cmd
	if input.Path != "" {
		cmd = exec.CommandContext(timeoutCtx, "sh", "-c", fmt.Sprintf("cd %s && %s", input.Path, input.Cmd))
	} else {
		cmd = exec.CommandContext(timeoutCtx, "sh", "-c", input.Cmd)
	}

	// 执行命令
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return ExecToolOutput{
				Success: false,
				Stdout:  string(stdout),
				Stderr:  string(exitErr.Stderr),
				Error:   fmt.Sprintf("command failed with exit code %d: %v", exitErr.ExitCode(), err),
			}
		}
		return ExecToolOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to execute command: %v", err),
		}
	}

	return ExecToolOutput{
		Success: true,
		Stdout:  string(stdout),
	}
}

// executeRemote 远程执行命令
func (t *ExecTool) executeRemote(ctx context.Context, input ExecToolInput) ExecToolOutput {
	// 解析address
	parts := strings.Split(input.Address, ":")
	if len(parts) != 2 {
		return ExecToolOutput{
			Success: false,
			Error:   fmt.Sprintf("invalid address format: %s, expected ip:port", input.Address),
		}
	}

	ip := parts[0]
	port := parts[1]

	// 从target repo中查找对应的node类型target
	filters := map[string]targets.Op{
		"address": {Op: "eq", Val: ip},
		"kind":    {Op: "eq", Val: targets.TargetKindNode},
	}

	targetList, err := t.targetRepo.List(ctx, filters)
	if err != nil {
		return ExecToolOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to find target: %v", err),
		}
	}

	if len(targetList) == 0 {
		return ExecToolOutput{
			Success: false,
			Error:   fmt.Sprintf("no node target found for address %s", input.Address),
		}
	}

	target := targetList[0]

	// 建立SSH连接
	config := &ssh.ClientConfig{
		User:            target.Username,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境应该验证host key
		Timeout:         time.Duration(input.Expire) * time.Second,
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
		return ExecToolOutput{
			Success: false,
			Error:   "no authentication method available (no password or ssh key)",
		}
	}

	// 连接SSH
	addr := fmt.Sprintf("%s:%s", ip, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return ExecToolOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to connect to %s: %v", addr, err),
		}
	}
	defer client.Close()

	// 创建session
	session, err := client.NewSession()
	if err != nil {
		return ExecToolOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to create session: %v", err),
		}
	}
	defer session.Close()

	// 构建命令
	cmdStr := input.Cmd
	if input.Path != "" {
		cmdStr = fmt.Sprintf("cd %s && %s", input.Path, input.Cmd)
	}

	// 执行命令
	output, err := session.CombinedOutput(cmdStr)
	if err != nil {
		return ExecToolOutput{
			Success: false,
			Stdout:  string(output),
			Error:   fmt.Sprintf("command execution failed: %v", err),
		}
	}

	return ExecToolOutput{
		Success: true,
		Stdout:  string(output),
	}
}

// Name 返回工具名称
func (t *ExecTool) Name() string {
	return "shell_exec"
}

// Description 返回工具描述
func (t *ExecTool) Description() string {
	return "执行shell命令，支持本地和远程(SSH)执行"
}
