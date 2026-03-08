# xdiag config - 配置管理

管理 xdiag 的配置，包括 LLM 模型参数、数据目录等。

## 子命令

- `model` - 配置 LLM 模型参数
- `show` - 显示当前配置
- `set` - 设置单个配置项
- `unset` - 删除配置项
- `test` - 测试 LLM 连接

---

## config model

配置 LLM 模型的 API Key、Base URL、协议类型和模型名称。

### 语法

```bash
xdiag config model --api-key <key> --model-name <name> [--base-url <url>] [--protocol <type>]
```

### 参数

- `--api-key` (必填): LLM API Key
- `--model-name` (必填): 模型名称，如 gpt-4o, claude-3-opus
- `--base-url` (可选): LLM Base URL，默认为 https://api.openai.com/v1
- `--protocol` (可选): 协议类型（openai/anthropic/custom），默认为 openai

### 示例

```bash
# 配置 OpenAI 模型
xdiag config model --api-key sk-xxx --model-name gpt-4o

# 配置自定义服务
xdiag config model \
  --api-key xxx \
  --base-url https://custom.ai.com/v1 \
  --protocol openai \
  --model-name custom-model

# 配置 Anthropic Claude
xdiag config model \
  --api-key sk-ant-xxx \
  --base-url https://api.anthropic.com \
  --protocol anthropic \
  --model-name claude-3-opus
```

### 输出

```
✅ 配置已保存到 ~/.github.com/bigWhiteXie/xdiag/config.yaml
```

---

## config show

显示当前所有配置项。

### 语法

```bash
xdiag config show
```

### 示例

```bash
xdiag config show
```

### 输出示例

```yaml
llm:
  api_key: sk-xxx
  base_url: https://api.openai.com/v1
  protocol: openai
  model_name: gpt-4o
data_dir: /root/.github.com/bigWhiteXie/xdiag/data
playbooks_dir: /root/.github.com/bigWhiteXie/xdiag/playbooks
```

---

## config set

设置单个配置项的值。

### 语法

```bash
xdiag config set <key> <value>
```

### 参数

- `key`: 配置项名称
- `value`: 配置项值

### 支持的配置项

- `model_name` - 模型名称
- `base_url` - API Base URL
- `api_key` - API Key
- `provider` - 提供商类型
- `data_dir` - 数据目录路径
- `book_path` - Playbook 目录路径

### 示例

```bash
# 设置模型名称
xdiag config set model_name gpt-4-turbo

# 设置 Base URL
xdiag config set base_url https://api.anthropic.com

# 设置 API Key
xdiag config set api_key sk-xxx

# 设置提供商
xdiag config set provider openai

# 设置数据目录
xdiag config set data_dir /root/.github.com/bigWhiteXie/xdiag/data

# 设置 Playbook 路径
xdiag config set book_path /root/.github.com/bigWhiteXie/xdiag/playbooks
```

### 输出

```
✅ model_name updated to gpt-4-turbo
```

---

## config unset

删除单个配置项。

### 语法

```bash
xdiag config unset <key>
```

### 参数

- `key`: 要删除的配置项名称

### 示例

```bash
# 删除 API Key
xdiag config unset api_key

# 删除 Base URL
xdiag config unset base_url
```

### 输出

```
✅ api_key removed from configuration
```

---

## config test

测试 LLM 配置是否有效，验证 API Key 和模型可用性。

### 语法

```bash
xdiag config test
```

### 示例

```bash
xdiag config test
```

### 输出示例

```
🔍 正在测试 LLM 连接...
✅ 连接成功！API Key 有效，模型可用：gpt-4o
   测试响应：Hello! Yes, I'm here and ready to help you...
```

### 错误示例

```
加载配置失败：配置文件不存在
```

或

```
API Key 未设置
```

或

```
测试 LLM 连接失败：invalid API key
```

---

## 配置文件位置

配置文件保存在：`~/.github.com/bigWhiteXie/xdiag/config.yaml`

## 使用流程

1. 首次使用时配置 LLM 模型：
```bash
xdiag config model --api-key sk-xxx --model-name gpt-4o
```

2. 测试配置是否有效：
```bash
xdiag config test
```

3. 查看当前配置：
```bash
xdiag config show
```

4. 根据需要调整配置：
```bash
xdiag config set data_dir /custom/path
```

## 注意事项

- 首次使用 xdiag 前必须配置 LLM 模型
- API Key 会保存在配置文件中，请确保文件权限安全
- 更改配置后建议运行 `config test` 验证
- 删除关键配置项（如 api_key）会导致诊断功能无法使用
