---
name: xdiag
description: 基于 AI 的智能诊断工具，用于诊断系统问题、执行自动化操作、管理目标资产和诊断剧本
version: 1.0.0
icon: 🔍
metadata:
  clawdbot:
    emoji: 🔍
    os: ["linux", "darwin"]
    requires:
      bins: ["xdiag"]
    install:
      - id: go-install
        kind: go
        command: go install github.com/bigWhiteXie/xdiag@latest
        bins: ["xdiag"]
        label: 通过 go install 安装 xdiag
---

# xdiag - 智能诊断工具

**xdiag** 是一个基于 AI 的智能诊断 CLI 工具，帮助用户诊断系统问题、管理目标资产和诊断剧本。它集成了四大核心功能模块，覆盖了从配置管理、资产管理到智能诊断的全生命周期。

## 📦 包含模块

### 1. 配置管理 (Config)
管理 xdiag 的配置，包括 LLM 模型参数、数据目录等。
*   **指令**: `xdiag config [show|set|unset|test]`
*   **详情**: [查看文档](references/config.md)

### 2. 目标资产管理 (Target)
管理诊断目标资产，包括服务器节点、数据库等。
*   **指令**: `xdiag target [add|list|get|update|delete|test]`
*   **详情**: [查看文档](references/target.md)

### 3. 诊断剧本管理 (Playbook)
管理诊断剧本（Playbook）和诊断方案（Book）。
*   **指令**: `xdiag playbook [list|show|generate]`
*   **详情**: [查看文档](references/playbook.md)

### 4. 智能诊断执行 (Diag)
根据用户描述执行智能诊断，自动匹配目标、检索方案并执行。
*   **指令**: `xdiag diag`
*   **详情**: [查看文档](references/diag.md)

## 🔄 保持更新

CLI 工具会持续迭代以修复问题和添加新功能，建议定期更新到最新版本：

```bash
go install github.com/bigWhiteXie/xdiag@latest
```


## 注意事项

- 首次使用前必须配置 LLM 模型
- 添加目标时确保网络连通性和认证信息正确
- 诊断方案生成依赖 LLM，需要有效的 API Key
- 执行诊断前建议先测试目标连通性
