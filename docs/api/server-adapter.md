# 申请审批与 ServerAdapter API 规范

> 状态：已实现基线  
> 版本：v1  
> 更新日期：2026-07-28  
> 机器可读契约：[openapi.yaml](openapi.yaml)

本文补充说明申请审批、服务器同步、失败重试和受限命令的状态语义。字段与路径以 OpenAPI 为准；本文负责解释跨接口约束、事务边界和安全要求。

## 1. 设计原则

申请审批和外部服务器同步是两个不同事实：

1. `Application.status` 表示组织是否批准申请。
2. `ApplicationServerSync.status` 表示批准结果是否已同步到外部服务器。
3. 外部适配器失败不得回滚已经提交的审批决定。
4. 生产未接入 RCON 时必须返回 `mode: disabled` 并拒绝外部执行；Mock 只供本地开发和自动化测试。
5. 浏览器不接触 RCON 地址、端口、密码、原始认证错误或可任意拼接的命令。

## 2. 权限和响应封装

| 能力 | 权限 |
| --- | --- |
| 查看申请 | `application:read` |
| 通过、拒绝、重试同步 | `application:approve` |
| 查看服务器后台状态 | `server:read_status` |
| 执行受限命令 | `server:command` |

单对象统一响应：

```json
{
  "data": {},
  "meta": {
    "request_id": "c5a22d03-b1b4-4c03-9939-c9ccdd98773e"
  }
}
```

错误统一响应：

```json
{
  "error": {
    "code": "application.server_sync_not_retryable",
    "message": "当前服务器同步状态不能重试。",
    "request_id": "c5a22d03-b1b4-4c03-9939-c9ccdd98773e"
  }
}
```

## 3. ApplicationServerSync

后台申请列表 `GET /api/v1/admin/applications` 可使用 `server_sync_status=none|pending|succeeded|failed` 筛选同步结果，并与 `status`、`type`、`query` 和分页参数组合。筛选始终受当前组织隔离约束；Admin 页面应展示尝试次数、请求时间以及脱敏后的 `message` 或 `last_error`。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 同步记录 ID。 |
| `operation` | enum | 当前为 `whitelist.add`。 |
| `adapter` | string | 服务端适配器名称，例如 `minecraft-mock`。 |
| `mode` | enum | `disabled`、`mock` 或 `rcon`。 |
| `status` | enum | `pending`、`succeeded`、`failed`。 |
| `attempts` | integer | 已完成的适配尝试次数，从 0 开始。 |
| `message` | string | 可向管理员展示的脱敏结果。 |
| `last_error` | string | 稳定、脱敏的失败摘要，不包含外部原始错误。 |
| `requested_at` | date-time | 首次请求或最近一次重试时间。 |
| `completed_at` | date-time / null | 最近一次尝试结束时间。 |

状态转换：

```text
创建审批任务 ──> pending ──> succeeded
                       └──> failed ──重试──> pending
```

- `succeeded` 不可重试。
- `pending` 不可并发重试。
- `failed` 可由具备审批权限的成员重试。
- 每次尝试完成后 `attempts + 1`。
- 重试不会新建另一条业务审批记录，也不会改变 `Application.status`。

## 4. 通过申请

`POST /api/v1/admin/applications/{application_id}/approve`

可选请求体为 `{ "reason": "资料完整，符合要求。" }`。通过备注最多 500 字符，随申请决定持久化，仅在 Admin API 展示。

对白名单申请，服务端按以下顺序处理：

1. 在数据库事务中将 `Application.status` 从 `pending` 改为 `approved`。
2. 在同一事务中写入审批审计和 `pending` 同步记录。
3. 提交事务。
4. 使用 ServerAdapter 执行白名单同步。
5. 独立写入 `succeeded` 或 `failed`。

成功且 Mock 模拟同步完成：

```json
{
  "data": {
    "id": "application-id",
    "status": "approved",
    "type": "whitelist",
    "game_id": "PlayerOne",
    "server_sync": {
      "id": "sync-id",
      "operation": "whitelist.add",
      "adapter": "minecraft-mock",
      "mode": "mock",
      "status": "succeeded",
      "attempts": 1,
      "message": "Mock 适配器已模拟白名单同步，未连接真实 RCON。",
      "last_error": "",
      "requested_at": "2026-07-28T08:00:00Z",
      "completed_at": "2026-07-28T08:00:00Z"
    }
  },
  "meta": {
    "request_id": "request-id"
  }
}
```

适配失败时 HTTP 仍为 `200`，申请仍是 `approved`，但同步状态为 `failed`：

```json
{
  "data": {
    "id": "application-id",
    "status": "approved",
    "server_sync": {
      "id": "sync-id",
      "mode": "rcon",
      "status": "failed",
      "attempts": 1,
      "message": "",
      "last_error": "服务器适配器执行失败；详细原因仅记录于受控服务日志。",
      "requested_at": "2026-07-28T08:00:00Z",
      "completed_at": "2026-07-28T08:00:05Z"
    }
  },
  "meta": {
    "request_id": "request-id"
  }
}
```

重复审批或并发状态已变化返回：

- HTTP `409`
- `application.already_decided`

## 5. 拒绝申请

`POST /api/v1/admin/applications/{application_id}/reject`

- 将待处理申请改为 `rejected`。
- 不创建服务器同步记录。
- `server_sync` 返回 `null`。
- 重复决定返回 `409 application.already_decided`。

## 6. 重试服务器同步

`POST /api/v1/admin/applications/{application_id}/server-sync/retry`

无请求体。服务端只允许重试当前组织、已批准、白名单类型且最新同步状态为 `failed` 的记录。

重试成功：

```json
{
  "data": {
    "id": "sync-id",
    "operation": "whitelist.add",
    "adapter": "minecraft-mock",
    "mode": "mock",
    "status": "succeeded",
    "attempts": 2,
    "message": "Mock 适配器已模拟白名单同步，未连接真实 RCON。",
    "last_error": "",
    "requested_at": "2026-07-28T08:05:00Z",
    "completed_at": "2026-07-28T08:05:00Z"
  },
  "meta": {
    "request_id": "request-id"
  }
}
```

外部服务仍失败时接口仍返回 `200`，但 `data.status` 为 `failed`，`attempts` 已递增。客户端必须读取业务状态，不能只根据 HTTP `200` 显示“同步成功”。

| HTTP | 错误码 | 场景 |
| --- | --- | --- |
| `401` | `auth.token_missing` / token 类错误 | 未登录或会话无效。 |
| `403` | `auth.permission_denied` | 缺少 `application:approve`。 |
| `404` | `application.server_sync_not_found` | 申请或同步记录不属于当前组织/不存在。 |
| `409` | `application.server_sync_not_retryable` | 状态不是 `failed`、申请未批准或不是白名单申请。 |
| `500` | `application.server_sync_retry_failed` | 数据库或内部状态更新失败。 |

重试审计：

- `application.server_sync_retry`：记录重试已被原子接受。
- `application.server_sync_retry_result`：记录最终 `succeeded` 或 `failed`。
- 两条记录使用相同 `request_id`。

## 7. 获取服务器状态

`GET /api/v1/admin/server/status`

Mock 状态示例：

```json
{
  "data": {
    "enabled": true,
    "adapter": "minecraft-mock",
    "mode": "mock",
    "label": "QUTCraft Minecraft Mock",
    "state": "maintenance",
    "online_players": 0,
    "max_players": 60,
    "updated_at": "2026-07-28T08:00:00Z",
    "last_command_at": null
  },
  "meta": {
    "request_id": "request-id"
  }
}
```

`mode` 是判断真实/模拟执行的权威字段。不得仅凭 `enabled` 或 `state` 推断已连接真实服务器。

## 8. 执行受限命令

`POST /api/v1/admin/server/commands`

允许的固定命令：

- `list`
- `save-all`
- `time set day`
- `weather clear`
- 非空的 `say <message>`

以下输入必须拒绝：

- 超过 256 字符；
- 包含 `\r` 或 `\n`；
- 任意不在白名单内的命令；
- 空 `say` 消息。

Mock 响应中的 `accepted: true` 只表示模拟适配器接受了请求；`executed: false` 表示没有在真实 Minecraft 服务器执行。

| HTTP | 错误码 | 场景 |
| --- | --- | --- |
| `400` | `server.command_invalid` | 请求体无效或命令为空。 |
| `403` | `server.command_not_allowed` | 命令或参数不在服务端白名单。 |
| `502` | `server.adapter_failed` | 适配器超时、离线或执行失败。 |

## 9. 超时和配置

- 所有 `Status`、`Execute`、`AddWhitelist` 调用统一受 `SERVER_ADAPTER_TIMEOUT` 限制。
- 默认超时为 5 秒。
- 非法或非正持续时间回退到默认值。
- 超时属于同步失败，不改变已经提交的审批决定。
- RCON 暂时搁置；生产默认适配器为 `disabled`，Mock 只用于测试和本地开发。

## 10. 安全边界

- Adapter 只存在于 Go 服务端，不作为 Portal API 或前端插件能力暴露。
- `last_error` 只保存稳定摘要；原始网络/RCON 错误不得进入 API、数据库审计正文或浏览器日志。
- `game_id` 在进入 Minecraft 白名单适配器时必须符合 Java 用户名规则：3–16 位 ASCII 字母、数字或下划线。
- 所有组织级查询必须同时包含 `organization_id`。
- 公开 Portal 只能读取脱敏服务器状态，不得读取审批、同步记录、命令或审计。
- AI 智能体不得直接获得 `server:command` 或 ServerAdapter 调用能力；相关规划见 [AI 智能体集成设计](../architecture/ai-agent-integration.md)。
