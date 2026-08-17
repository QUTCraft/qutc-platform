# API 可观测性与审计规范

> 状态：已实现
> 适用范围：Go API、Docker Compose、Admin 审计页、部署与故障排查
> 契约源：[openapi.yaml](openapi.yaml)

## 1. Request ID

每个 HTTP 请求都必须具有一个关联 ID。客户端可以发送 `X-Request-ID`，但服务端只保留符合 `^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$` 的值；缺失、超长或包含空白/控制字符的值会被 UUID 替换。

最终 ID 会同时出现在：

- 响应头 `X-Request-ID`；
- 成功响应 `meta.request_id` 或错误响应 `error.request_id`；
- API 结构化访问日志的 `request_id`；
- 写操作生成的 `audit_events.request_id`。

因此排错时应先从浏览器响应或 Admin 审计页取得 Request ID，再到容器日志中精确检索。Request ID 只用于关联，不是认证凭据，也不能据此授权或推导组织范围。

## 2. 结构化访问日志

API 将每次请求输出为单行 JSON。固定字段如下：

| 字段 | 说明 |
| --- | --- |
| `event` | 固定为 `http_request`。 |
| `request_id` | 经服务端校验后的关联 ID。 |
| `method`、`path`、`route` | HTTP 方法、无查询串路径和 Gin 路由模板。 |
| `status`、`latency_ms` | 响应状态和处理耗时。 |
| `client_ip` | 由受信代理配置确定的客户端地址；当前不信任任意转发头。 |
| `user_id`、`organization_id` | 认证成功后才记录的内部 ID。 |

日志禁止记录：

- `Authorization`、Cookie、Access/Refresh Token 或邀请明文 Token；
- 查询串、请求体、密码、邮箱、QQ、SMTP/MinIO 凭据；
- 数据库 DSN、对象键或外部适配器的原始认证错误。

`2xx/3xx` 使用 info，`4xx` 使用 warn，`5xx` 使用 error。业务审计不能由访问日志替代：访问日志说明“请求发生了”，审计事件说明“哪个组织中的谁对什么对象执行了什么结果”。

## 3. 存活与就绪探针

| 路径 | 含义 | 依赖检查 | 用途 |
| --- | --- | --- | --- |
| `GET /healthz` | API 进程仍能响应 | 无 | 容器存活探针。 |
| `GET /readyz` | API 可以接收业务流量 | MySQL、Redis | 反向代理、部署和运行态冒烟。 |

就绪成功返回：

```json
{
  "status": "ready",
  "checks": {
    "mysql": "ok",
    "redis": "ok"
  }
}
```

任一必要依赖不可用时返回 `503`、`status=unavailable`。响应只暴露组件级状态，不返回连接地址、凭据或底层错误。媒体存储驱动在 API 启动时完成初始化，读写故障由资产接口以 `503` 和 Request ID 表达；它暂不作为全局就绪阻断项。

## 4. 审计查询

`GET /api/v1/admin/audit` 需要 Bearer JWT 与 `audit:read`。服务端从当前认证主体取得 `organization_id` 并强制加入数据库条件；请求不接受组织参数，不能查询其他组织。

支持分页与下列精确筛选：

- `action`、`target_type`、`result`；
- `actor_user_id`、`request_id`；
- `date_from`、`date_to`，格式 `YYYY-MM-DD`，按 UTC 自然日且包含边界日期。

结果按 `created_at DESC, id DESC` 返回，并包含操作者内部 ID/显示名、动作、对象、结果、Request ID 和时间。接口不返回操作者邮箱、请求体、命令原文或任何凭据。后台入口为 `/admin/audit`。

## 5. 验证

```powershell
# 不依赖 Compose
.\scripts\run-quality-gate.ps1

# Compose 启动后，包含 readiness、组织隔离和 Request ID 集成验证
.\scripts\run-s5-observability-integration.ps1

# 连同 S1—S6 业务集成一起执行
.\scripts\run-quality-gate.ps1 -Integration
```

专项测试还会创建同 Request ID 的跨组织审计记录，确认当前管理员只能看到本组织事件；测试结束后自动清理临时记录和组织。
