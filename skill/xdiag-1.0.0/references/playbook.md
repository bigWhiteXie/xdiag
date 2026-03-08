# xdiag playbook - 诊断剧本管理

管理诊断剧本（Playbook）和诊断方案（Book）。Playbook 是一组相关诊断方案的集合，每个诊断方案包含具体的诊断步骤和逻辑。

## 子命令

- `list` - 列出所有可用的 Playbook
- `show` - 显示指定 Playbook 的详细信息
- `generate` - 生成新的诊断方案

---

## playbook list

列出所有在系统中注册的 Playbook 及其基本信息。

### 语法

```bash
xdiag playbook list
```

### 示例

```bash
xdiag playbook list
```

### 输出示例

```
Available Playbooks:
NAME                    DESC                            REQUIRED_TAGS
system-diagnostics      系统性能诊断剧本                 node
database-health         数据库健康检查剧本               postgres,mysql
network-troubleshoot    网络故障排查剧本                 node
redis-performance       Redis 性能分析剧本               redis
```

### 输出说明

- **NAME**: Playbook 名称（目录名）
- **DESC**: Playbook 描述
- **REQUIRED_TAGS**: 执行此 Playbook 所需的目标标签，如果为 `none` 表示无特殊要求

---

## playbook show

显示指定名称的 Playbook 的详细信息，包括其所有诊断方案。

### 语法

```bash
xdiag playbook show [playbook-name]
```

### 参数

- `playbook-name` (必填): Playbook 名称

### 示例

```bash
# 显示系统诊断 Playbook 的详情
xdiag playbook show system-diagnostics

# 显示数据库健康检查 Playbook 的详情
xdiag playbook show database-health
```

### 输出示例

```
Playbook: system-diagnostics
Description: 系统性能诊断剧本
Required Tags: node

Diagnosis Refs:
  1. cpu-high-usage: 诊断 CPU 使用率过高问题
  2. memory-leak: 检测内存泄漏问题
  3. disk-full: 诊断磁盘空间不足问题
  4. network-latency: 分析网络延迟问题
```

### 输出说明

- **Playbook**: Playbook 名称
- **Description**: Playbook 描述
- **Required Tags**: 所需的目标标签
- **Diagnosis Refs**: 该 Playbook 包含的所有诊断方案列表

---

## playbook generate

根据描述生成新的诊断/自动化方案（Book），使用 AI 自动生成诊断步骤和逻辑。

### 语法

```bash
xdiag playbook generate --playbook <name> --name <book-name> --desc <description>
```

### 参数

- `--playbook` (必填): 要添加到的 Playbook 名称
- `--name` (必填): 诊断方案名称
- `--desc` (必填): 诊断方案描述

### 示例

```bash
# 生成磁盘检查方案
xdiag playbook generate \
  --playbook system-diagnostics \
  --name disk-usage-check \
  --desc "检查磁盘使用率是否超过 80% 阈值"

# 生成数据库连接池方案
xdiag playbook generate \
  --playbook database-health \
  --name connection-pool-check \
  --desc "检查数据库连接池是否耗尽"

# 生成内存泄漏检测方案
xdiag playbook generate \
  --playbook system-diagnostics \
  --name memory-leak-detection \
  --desc "检测应用程序是否存在内存泄漏，分析内存增长趋势"

# 生成 Redis 慢查询方案
xdiag playbook generate \
  --playbook redis-performance \
  --name slow-query-analysis \
  --desc "分析 Redis 慢查询日志，找出性能瓶颈"
```

### 输出示例

```
诊断方案已成功生成到Playbook 'system-diagnostics'
{
  "name": "disk-usage-check",
  "desc": "检查磁盘使用率是否超过 80% 阈值",
  "steps": [
    {
      "name": "获取磁盘使用情况",
      "command": "df -h",
      "type": "shell"
    },
    {
      "name": "分析磁盘使用率",
      "type": "analysis"
    }
  ]
}
```

### 工作原理

1. 接收用户提供的诊断方案描述
2. 调用 LLM 模型分析描述，生成诊断步骤
3. 将生成的诊断方案保存到指定的 Playbook 目录
4. 返回生成的诊断方案详情

---

## Playbook 目录结构

Playbook 存储在：`~/.github.com/bigWhiteXie/xdiag/playbooks/`

典型的目录结构：

```
playbooks/
├── system-diagnostics/
│   ├── playbook.yaml          # Playbook 元数据
│   ├── cpu-high-usage.yaml    # 诊断方案 1
│   ├── memory-leak.yaml       # 诊断方案 2
│   └── disk-full.yaml         # 诊断方案 3
├── database-health/
│   ├── playbook.yaml
│   ├── connection-check.yaml
│   └── slow-query.yaml
└── network-troubleshoot/
    ├── playbook.yaml
    └── latency-check.yaml
```

## 使用流程

1. 查看可用的 Playbook：
```bash
xdiag playbook list
```

2. 查看具体 Playbook 的详情：
```bash
xdiag playbook show system-diagnostics
```

3. 生成新的诊断方案：
```bash
xdiag playbook generate \
  --playbook system-diagnostics \
  --name process-check \
  --desc "检查占用 CPU 最高的进程"
```

4. 再次查看 Playbook 确认方案已添加：
```bash
xdiag playbook show system-diagnostics
```

5. 使用诊断命令执行方案：
```bash
xdiag diag --question "服务器 CPU 使用率过高"
```

## Playbook 与 Target 的关系

- Playbook 的 `Required Tags` 字段定义了适用的目标类型
- 执行诊断时，系统会根据目标的标签匹配合适的 Playbook
- 例如：
  - `system-diagnostics` 需要 `node` 标签，适用于服务器节点
  - `database-health` 需要 `postgres` 或 `mysql` 标签，适用于数据库
  - `redis-performance` 需要 `redis` 标签，适用于 Redis 实例

## 诊断方案生成最佳实践

### 描述要清晰具体

❌ 不好的描述：
```bash
--desc "检查系统"
```

✅ 好的描述：
```bash
--desc "检查系统 CPU 使用率是否超过 80%，并找出占用最高的前 5 个进程"
```

### 包含诊断目标

❌ 不好的描述：
```bash
--desc "性能问题"
```

✅ 好的描述：
```bash
--desc "诊断数据库查询性能问题，分析慢查询日志并给出优化建议"
```

### 指定阈值和条件

❌ 不好的描述：
```bash
--desc "内存使用"
```

✅ 好的描述：
```bash
--desc "检查内存使用率是否超过 90%，分析内存占用最高的进程"
```

## 注意事项

- 生成诊断方案需要有效的 LLM 配置
- Playbook 名称必须已存在，不能生成到不存在的 Playbook
- 诊断方案名称在同一个 Playbook 中必须唯一
- 生成的方案质量取决于描述的清晰度和 LLM 模型的能力
- 建议生成后测试方案的有效性
- 复杂的诊断逻辑可能需要手动调整生成的方案

## 高级用法

### 创建自定义 Playbook

可以手动在 playbooks 目录下创建新的 Playbook：

1. 创建 Playbook 目录：
```bash
mkdir -p ~/.github.com/bigWhiteXie/xdiag/playbooks/my-custom-playbook
```

2. 创建 playbook.yaml：
```yaml
name: my-custom-playbook
desc: 我的自定义诊断剧本
tags:
  - node
  - custom
```

3. 生成诊断方案：
```bash
xdiag playbook generate \
  --playbook my-custom-playbook \
  --name my-check \
  --desc "自定义检查逻辑"
```

### 方案命名建议

- 使用小写字母和连字符
- 名称要能反映方案的功能
- 例如：`cpu-high-usage`, `memory-leak-detection`, `disk-space-check`
