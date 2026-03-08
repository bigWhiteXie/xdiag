# xdiag - 智能诊断 CLI 工具

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

xdiag 是一个基于 AI 的智能诊断命令行工具，帮助用户自动化诊断系统问题。通过集成大型语言模型（LLM），xdiag 能够理解用户的自然语言描述，自动匹配合适的诊断方案，并执行相应的诊断步骤。

## 功能特性

- **智能目标识别**: 根据用户描述自动识别和定位目标资产
- **多协议支持**: 支持 SSH 节点、PostgreSQL、MySQL、Redis 等多种目标类型
- **AI 驱动的诊断**: 利用 LLM 理解用户意图，智能匹配诊断方案
- **灵活的 Playbook 系统**: 可扩展的诊断剧本管理系统
- **连通性测试**: 内置目标连通性和认证测试功能
- **配置管理**: 完整的 LLM 配置和参数管理

## 安装

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/bigWhiteXie/xdiag.git
cd xdiag

# 构建可执行文件
make build

# 或者直接使用 go build
go build -o xdiag ./main.go
```

### 开发模式运行

```bash
# 直接运行（无需编译）
go run main.go --help
```

## 快速开始

### 1. 配置 LLM

首先需要配置 LLM API 密钥和模型参数：

```bash
# 配置 OpenAI 模型
xdiag config model \
  --api-key sk-your-api-key-here \
  --model-name gpt-4o

# 配置自定义 LLM 服务
xdiag config model \
  --api-key your-custom-key \
  --base-url https://custom.ai.com/v1 \
  --protocol openai \
  --model-name custom-model
```

### 2. 添加目标资产

添加需要诊断的目标资产：

```bash
# 添加 SSH 节点
xdiag target add \
  --name prod-server \
  --kind node \
  --address 192.168.1.100 \
  --port 22 \
  --username admin \
  --ssh-key ~/.ssh/id_rsa \
  --tags production,web

# 添加 PostgreSQL 数据库
xdiag target add \
  --name prod-db \
  --kind postgres \
  --address db.example.com \
  --port 5432 \
  --username postgres \
  --password your-password \
  --tags production,database
```

### 3. 执行智能诊断

使用自然语言描述问题，让 xdiag 自动诊断：

```bash
# 诊断服务器 CPU 使用率过高问题
xdiag diag --question "我的生产服务器 CPU 使用率很高，帮我诊断一下"

# 显示详细诊断过程
xdiag diag --question "数据库连接很慢" --show
```

## 命令参考

### 配置管理 (`config`)

```bash
# 配置 LLM 模型参数
xdiag config model --api-key <key> --model-name <model> [--base-url <url>] [--protocol <protocol>]

# 设置单个配置项
xdiag config set <key> <value>

# 显示当前配置
xdiag config show

# 删除配置项
xdiag config unset <key>

# 测试 LLM 配置有效性
xdiag config test
```

### 目标资产管理 (`target`)

```bash
# 添加目标
xdiag target add --kind <type> --address <addr> [其他参数]

# 列出所有目标（支持过滤）
xdiag target list [--kind <type>] [--tag <tag>]

# 获取目标详情
xdiag target get --name <name> | --id <id>

# 更新目标信息
xdiag target update --id <id> [要更新的字段]

# 删除目标
xdiag target delete --id <id>

# 测试目标连通性
xdiag target test --name <name> | --id <id>
```

### Playbook 管理 (`playbook`)

```bash
# 列出所有可用的诊断剧本
xdiag playbook list

# 显示指定 Playbook 的详细信息
xdiag playbook show <playbook-name>
```

### 智能诊断 (`diag`)

```bash
# 执行智能诊断
xdiag diag --question "<问题描述>" [--show]
```

## 支持的目标类型

| 类型 | 描述 | 认证方式 |
|------|------|----------|
| `node` | Linux/Unix 服务器节点 | SSH 密钥或密码 |
| `postgres` | PostgreSQL 数据库 | 用户名密码 |
| `mysql` | MySQL 数据库 | 用户名密码 |
| `redis` | Redis 缓存服务 | 密码（可选） |

## 配置文件

xdiag 的配置文件存储在 `~/.xdiag/config.yaml`，包含以下配置项：

```yaml
llm:
  api_key: "your-api-key"
  base_url: "https://api.openai.com/v1"
  protocol: "openai"
  model_name: "gpt-4o"
data_dir: "/home/user/.xdiag/data"
playbooks_dir: "/home/user/.xdiag/playbooks"
```

## 目录结构

```
~/.xdiag/
├── config.yaml          # 主配置文件
├── data/
│   └── targets.db       # 目标资产数据库
└── playbooks/           # 诊断剧本目录
    ├── node-diagnosis/
    │   ├── introduction.yaml
    │   ├── metadata.yaml
    │   └── refs/
    │       └── cpu-high.yaml
    └── db-diagnosis/
        ├── introduction.yaml
        ├── metadata.yaml
        └── refs/
            └── slow-query.yaml
```

## 环境变量

xdiag 支持通过环境变量覆盖配置：

- `XDIAG_API_KEY`: LLM API 密钥
- `XDIAG_BASE_URL`: LLM Base URL
- `XDIAG_MODEL_NAME`: 模型名称
- `XDIAG_DATA_DIR`: 数据目录路径
- `XDIAG_PLAYBOOKS_DIR`: Playbook 目录路径

## 依赖

- Go 1.25+
- SQLite3 (通过 CGO 集成)
- 支持的 LLM 服务 (OpenAI, Anthropic, 或兼容 OpenAI 协议的自定义服务)

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！请确保遵循项目的代码规范和测试要求。

## 技术栈

- **核心框架**: [Cobra](https://github.com/spf13/cobra) - CLI 框架
- **配置管理**: [Viper](https://github.com/spf13/viper) - 配置解析
- **AI 集成**: [Eino](https://github.com/cloudwego/eino) - LLM 框架
- **数据库**: SQLite3 + GORM - 目标资产管理
- **YAML 解析**: gopkg.in/yaml.v3 - Playbook 配置解析

## 示例场景

### 场景 1: 服务器性能问题诊断

```bash
# 添加服务器目标
xdiag target add --name web-server --kind node --address 10.0.0.10 --port 22 --username ubuntu --ssh-key ~/.ssh/prod-key --tags web,production

# 执行诊断
xdiag diag --question "我的 Web 服务器响应很慢，CPU 使用率经常达到 90% 以上"
```

### 场景 2: 数据库连接问题

```bash
# 添加数据库目标
xdiag target add --name mysql-master --kind mysql --address 10.0.0.20 --port 3306 --username admin --password secret --tags database,master

# 执行诊断
xdiag diag --question "应用无法连接到 MySQL 主库，连接超时"
```

### 场景 3: Redis 缓存问题

```bash
# 添加 Redis 目标
xdiag target add --name redis-cache --kind redis --address 10.0.0.30 --port 6379 --tags cache,production

# 执行诊断
xdiag diag --question "10.101.135.25上的redis 缓存命中率很低，内存使用率很高"
```