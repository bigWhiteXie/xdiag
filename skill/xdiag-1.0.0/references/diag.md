# xdiag diag - 执行智能诊断

根据用户描述执行智能诊断，自动完成目标匹配、方案检索和执行等步骤，最终生成诊断报告。

## 语法

```bash
xdiag diag --question <description> [--show]
```

## 参数

- `--question` (必填): 问题描述，描述需要诊断的问题
- `--show` (可选): 是否显示诊断过程的详细信息

## 工作流程

执行 `xdiag diag` 命令时，系统会自动完成以下步骤：

1. **分析问题** - 使用 AI 分析用户的问题描述
2. **定位目标** - 根据问题描述匹配合适的目标资产
3. **测试连通性** - 验证目标资产的连通性和认证
4. **匹配方案** - 从 Playbook 中检索最合适的诊断方案
5. **执行诊断** - 在目标上执行诊断方案
6. **生成报告** - 汇总诊断结果并生成报告

## 示例

### 基本诊断

```bash
# 诊断数据库连接问题
xdiag diag --question "生产环境数据库连接缓慢"

# 诊断服务器性能问题
xdiag diag --question "服务器 CPU 使用率过高"

# 诊断磁盘空间问题
xdiag diag --question "web 服务器磁盘空间不足"

# 诊断 Redis 性能问题
xdiag diag --question "Redis 缓存响应时间变长"
```

### 显示详细过程

```bash
# 显示诊断过程的详细信息
xdiag diag --question "生产环境 web 服务器响应缓慢" --show
```

使用 `--show` 参数可以查看：
- 每个诊断步骤的执行过程
- 中间结果和分析
- 命令执行的详细输出

## 输出示例

### 成功诊断

```
正在分析您的问题: 生产环境数据库连接缓慢......
✅ 找到目标, ip:192.168.1.50 port:5432 kind:postgres tags:production,db
✅ 匹配成功 方案:connection-pool-check 描述:检查数据库连接池状态
诊断报告:
===========================================
诊断目标: postgres-192.168.1.50:5432
诊断方案: connection-pool-check
执行时间: 2026-03-08 14:30:00

诊断结果:
- 连接池使用率: 95%
- 活跃连接数: 190/200
- 等待连接数: 15

问题分析:
数据库连接池接近饱和，大量请求在等待可用连接。

建议措施:
1. 增加连接池最大连接数
2. 优化慢查询，减少连接占用时间
3. 检查是否存在连接泄漏
===========================================
```

### 未找到目标

```
正在分析您的问题: 测试环境服务器问题......
未找到匹配的目标
```

### 未匹配到方案

```
正在分析您的问题: 服务器硬件故障......
✅ 找到目标, ip:192.168.1.100 port:22 kind:node tags:production,web
未匹配到playbook, 具体信息: 没有找到适合硬件故障诊断的方案
```

### 连通性测试失败

```
正在分析您的问题: 数据库连接问题......
✅ 找到目标, ip:192.168.1.50 port:5432 kind:postgres tags:production,db
测试目标连通性失败: connection timeout
```

## 问题描述最佳实践

### 包含关键信息

好的问题描述应该包含：
- 目标环境（生产/测试/开发）
- 目标类型（服务器/数据库/缓存）
- 具体症状（CPU 高/响应慢/连接失败）

### 示例对比

❌ 不好的描述：
```bash
xdiag diag --question "有问题"
xdiag diag --question "系统慢"
xdiag diag --question "出错了"
```

✅ 好的描述：
```bash
xdiag diag --question "生产环境 web 服务器 CPU 使用率持续在 90% 以上"
xdiag diag --question "PostgreSQL 数据库查询响应时间从 100ms 增加到 5s"
xdiag diag --question "Redis 缓存命中率从 95% 下降到 60%"
```

## 使用场景

### 1. 性能问题诊断

```bash
# CPU 使用率高
xdiag diag --question "生产服务器 CPU 使用率达到 95%"

# 内存不足
xdiag diag --question "应用服务器内存使用率超过 90%"

# 磁盘 I/O 高
xdiag diag --question "数据库服务器磁盘 I/O 等待时间过长"
```

### 2. 连接问题诊断

```bash
# 数据库连接
xdiag diag --question "应用无法连接到 PostgreSQL 数据库"

# SSH 连接
xdiag diag --question "无法 SSH 登录到生产服务器"

# Redis 连接
xdiag diag --question "Redis 连接超时"
```

### 3. 资源问题诊断

```bash
# 磁盘空间
xdiag diag --question "web 服务器磁盘使用率达到 95%"

# 连接池
xdiag diag --question "数据库连接池耗尽"

# 文件描述符
xdiag diag --question "应用程序打开文件数达到上限"
```

### 4. 应用问题诊断

```bash
# 响应慢
xdiag diag --question "API 接口响应时间从 200ms 增加到 3s"

# 错误率高
xdiag diag --question "应用错误率突然增加到 10%"

# 内存泄漏
xdiag diag --question "Java 应用内存持续增长不释放"
```

## 诊断报告说明

诊断报告通常包含以下部分：

1. **诊断目标** - 执行诊断的目标资产信息
2. **诊断方案** - 使用的诊断方案名称和描述
3. **执行时间** - 诊断执行的时间
4. **诊断结果** - 收集到的数据和指标
5. **问题分析** - AI 对问题的分析和判断
6. **建议措施** - 针对问题的解决建议

## 前置条件

执行诊断前需要确保：

1. **LLM 配置完成**
```bash
xdiag config model --api-key sk-xxx --model-name gpt-4o
xdiag config test
```

2. **目标资产已添加**
```bash
xdiag target add \
  --kind node \
  --address 192.168.1.100 \
  --port 22 \
  --username admin \
  --ssh-key ~/.ssh/id_rsa \
  --tags production,web
```

3. **Playbook 已配置**
```bash
xdiag playbook list
```

## 故障排查

### 问题：未找到匹配的目标

**原因：**
- 问题描述不够明确
- 没有添加相关的目标资产
- 目标标签与问题不匹配

**解决方法：**
```bash
# 1. 查看现有目标
xdiag target list

# 2. 添加缺失的目标
xdiag target add --kind node --address 192.168.1.100 --port 22 --username admin --tags production,web

# 3. 使用更明确的问题描述
xdiag diag --question "生产环境 192.168.1.100 服务器 CPU 使用率过高"
```

### 问题：未匹配到 Playbook

**原因：**
- 没有适合该问题的诊断方案
- Playbook 的标签要求与目标不匹配

**解决方法：**
```bash
# 1. 查看可用的 Playbook
xdiag playbook list

# 2. 生成新的诊断方案
xdiag playbook generate \
  --playbook system-diagnostics \
  --name custom-check \
  --desc "针对当前问题的诊断方案"

# 3. 重新执行诊断
xdiag diag --question "原问题描述"
```

### 问题：连通性测试失败

**原因：**
- 网络不通
- 认证信息错误
- 目标服务未启动

**解决方法：**
```bash
# 1. 单独测试连通性
xdiag target test --id 1

# 2. 更新认证信息
xdiag target update --id 1 --username newuser --password newpass

# 3. 检查网络和防火墙配置
```

## 高级用法

### 批量诊断

可以编写脚本批量执行诊断：

```bash
#!/bin/bash

# 定义问题列表
questions=(
  "生产服务器 CPU 使用率过高"
  "数据库连接池接近饱和"
  "Redis 缓存命中率下降"
)

# 批量执行诊断
for question in "${questions[@]}"; do
  echo "诊断: $question"
  xdiag diag --question "$question"
  echo "---"
done
```

### 定时诊断

使用 cron 定时执行诊断：

```bash
# 每小时执行一次系统健康检查
0 * * * * /usr/local/bin/xdiag diag --question "生产环境系统健康检查" >> /var/log/xdiag.log 2>&1
```

### 结合监控告警

在监控告警触发时自动执行诊断：

```bash
#!/bin/bash
# 当 CPU 告警触发时执行

ALERT_MESSAGE="$1"
xdiag diag --question "生产服务器 $ALERT_MESSAGE" --show
```

## 注意事项

- 诊断会在目标系统上执行命令，确保有足够的权限
- 某些诊断操作可能会影响系统性能，建议在低峰期执行
- 诊断结果依赖 AI 分析，建议结合人工判断
- 复杂问题可能需要多次诊断和调整
- 保存诊断报告以便后续分析和对比
- 定期更新和优化诊断方案以提高准确性
