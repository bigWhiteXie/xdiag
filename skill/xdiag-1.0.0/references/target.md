# xdiag target - 目标资产管理

管理诊断目标资产，包括服务器节点、数据库等。

## 子命令

- `add` - 添加新的目标资产
- `list` - 列出所有目标资产
- `get` - 获取目标详情
- `update` - 更新目标信息
- `delete` - 删除目标
- `test` - 测试目标连通性

---

## target add

添加一个新的目标资产。

### 语法

```bash
xdiag target add --kind <type> --address <addr> [options]
```

### 参数

- `--kind` (必填): 目标类型 - node/postgres/mysql/redis
- `--address` (必填): 目标地址（IP 或域名）
- `--port` (可选): 目标端口
- `--username` (可选): 用户名
- `--password` (可选): 密码
- `--tags` (可选): 逗号分隔的标签

### 示例

```bash
# 添加一个服务器节点
xdiag target add \
  --kind node \
  --address 192.168.1.100 \
  --port 22 \
  --username admin \
  --password 123456 \
  --tags production,web

# 添加一个 PostgreSQL 数据库
xdiag target add \
  --kind postgres \
  --address db.example.com \
  --port 5432 \
  --username postgres \
  --password secret \
  --tags production,db

# 添加 MySQL 数据库
xdiag target add \
  --kind mysql \
  --address 10.0.0.50 \
  --port 3306 \
  --username root \
  --password mypass \
  --tags staging,mysql

# 添加 Redis 实例
xdiag target add \
  --kind redis \
  --address redis.example.com \
  --port 6379 \
  --password redispass \
  --tags cache,production
```

### 输出

```
✅ Target 'node-192.168.1.100:22' added successfully with ID 1
```

---

## target list

列出所有目标资产，支持按类型和标签过滤。

### 语法

```bash
xdiag target list [--kind <type>] [--tag <tag>]
```

### 参数

- `--kind` (可选): 按类型过滤（node/postgres/mysql/redis）
- `--tag` (可选): 按标签过滤

### 示例

```bash
# 列出所有目标
xdiag target list

# 列出所有节点类型的目标
xdiag target list --kind node

# 列出所有带 production 标签的目标
xdiag target list --tag production

# 列出所有 PostgreSQL 数据库
xdiag target list --kind postgres
```

### 输出示例

```
Found 3 target(s):
- ID: 1, Name: node-192.168.1.100:22, Kind: node, Address: 192.168.1.100:22, Tags: production,web
- ID: 2, Name: postgres-db.example.com:5432, Kind: postgres, Address: db.example.com:5432, Tags: production,db
- ID: 3, Name: redis-redis.example.com:6379, Kind: redis, Address: redis.example.com:6379, Tags: cache,production
```

---

## target get

根据名称或 ID 获取目标资产的详细信息。

### 语法

```bash
xdiag target get --name <name>
xdiag target get --id <id>
```

### 参数

- `--name` (二选一): 目标名称
- `--id` (二选一): 目标 ID

### 示例

```bash
# 通过名称获取
xdiag target get --name myserver

# 通过 ID 获取
xdiag target get --id 1
```

### 输出示例

```
Target Details:
  ID: 1
  Name: node-192.168.1.100:22
  Kind: node
  Address: 192.168.1.100:22
  Username: admin
  Tags: production,web
  Created At: 2026-03-08 10:30:00
  Updated At: 2026-03-08 10:30:00
```

---

## target update

更新现有目标资产的信息。

### 语法

```bash
xdiag target update --id <id> [options]
```

### 参数

- `--id` (必填): 目标 ID
- `--name` (可选): 新的目标名称
- `--kind` (可选): 新的目标类型
- `--address` (可选): 新的目标地址
- `--port` (可选): 新的目标端口
- `--username` (可选): 新的用户名
- `--password` (可选): 新的密码
- `--tags` (可选): 新的标签

### 示例

```bash
# 更新目标地址和端口
xdiag target update \
  --id 1 \
  --address 192.168.1.101 \
  --port 22

# 更新标签
xdiag target update \
  --id 1 \
  --tags production,web,updated

# 更新用户名和密码
xdiag target update \
  --id 2 \
  --username newuser \
  --password newpass

# 更新多个字段
xdiag target update \
  --id 1 \
  --name new-server \
  --address 10.0.0.100 \
  --tags production,critical
```

### 输出

```
✅ Target 'new-server' (ID: 1) updated successfully
```

---

## target delete

根据 ID 删除目标资产。

### 语法

```bash
xdiag target delete --id <id>
```

### 参数

- `--id` (必填): 要删除的目标 ID

### 示例

```bash
# 删除 ID 为 1 的目标
xdiag target delete --id 1

# 删除 ID 为 5 的目标
xdiag target delete --id 5
```

### 输出

```
✅ Target with ID 1 deleted successfully
```

---

## target test

测试目标资产的连通性和认证状态。

### 语法

```bash
xdiag target test --name <name>
xdiag target test --id <id>
```

### 参数

- `--name` (二选一): 目标名称
- `--id` (二选一): 目标 ID

### 示例

```bash
# 通过名称测试
xdiag target test --name myserver

# 通过 ID 测试
xdiag target test --id 1
```

### 输出示例

```
Connectivity Test Result for 'node-192.168.1.100:22' (ID: 1):
  Status: success
  Ping Status: reachable
  Auth Status: authenticated
  Message: Connection successful
  Extra Details:
    ssh_version: OpenSSH_8.9
    os_type: Linux
```

### 失败示例

```
Connectivity Test Result for 'node-192.168.1.100:22' (ID: 1):
  Status: failed
  Ping Status: unreachable
  Auth Status: unknown
  Message: Connection timeout
```

---

## 数据存储

目标资产数据保存在：`~/.github.com/bigWhiteXie/xdiag/data/targets.db`（SQLite 数据库）

## 使用流程

1. 添加目标资产：
```bash
xdiag target add \
  --kind node \
  --address 192.168.1.100 \
  --port 22 \
  --username admin \
  --password 123456 \
  --tags production,web
```

2. 测试连通性：
```bash
xdiag target test --id 1
```

3. 查看所有目标：
```bash
xdiag target list
```

4. 查看详细信息：
```bash
xdiag target get --id 1
```

5. 更新目标信息：
```bash
xdiag target update --id 1 --tags production,web,critical
```

6. 删除不需要的目标：
```bash
xdiag target delete --id 1
```

## 支持的目标类型

- **node**: 服务器节点（通过 SSH 连接）
- **postgres**: PostgreSQL 数据库
- **mysql**: MySQL 数据库
- **redis**: Redis 缓存

## 标签使用建议

使用标签可以更好地组织和过滤目标资产：

- 环境标签：`production`, `staging`, `development`
- 功能标签：`web`, `db`, `cache`, `api`
- 区域标签：`us-east`, `eu-west`, `asia`
- 状态标签：`critical`, `monitoring`, `deprecated`

## 注意事项

- 添加目标前确保网络连通性
- 删除目标是不可逆操作，请谨慎使用
- 测试连通性会实际尝试连接目标，可能会触发安全告警
