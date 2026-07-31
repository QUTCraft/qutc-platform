# 邀请邮件适配器规范

## 1. 目标与边界

邀请邮件是成员邀请流程的可选外部适配器，不是邀请事务的组成部分：

- 邀请记录和 token 哈希先写入数据库，提交成功后才尝试邮件投递。
- SMTP 连接、认证或收件失败不会删除邀请，也不会回滚成员邀请业务状态。
- 未启用 SMTP 时，Admin 仍会取得一次性 `invite_url`，可复制链接完成完整加入流程。
- 浏览器只读取适配器状态和单次投递结果，不读取 SMTP 主机、用户名、密码或授权码。
- 服务端始终只持久化邀请 token 的 SHA-256 哈希，不为邮件重试保存明文 token。

## 2. 服务端配置

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PUBLIC_WEB_BASE_URL` | `http://localhost:8082` | 邮件中邀请链接的公开 Web 根地址。 |
| `EMAIL_DRIVER` | `disabled` | `disabled` 或 `smtp`。 |
| `SMTP_HOST` | 空 | SMTP 主机；启用 `smtp` 时必填。 |
| `SMTP_PORT` | `587` | SMTP 端口，范围 `1..65535`。 |
| `SMTP_USERNAME` | 空 | 可选认证用户名。 |
| `SMTP_PASSWORD` | 空 | 设置用户名时必填，只允许部署环境注入。 |
| `SMTP_FROM_ADDRESS` | 空 | 发件邮箱，启用 `smtp` 时必填。 |
| `SMTP_FROM_NAME` | `QUTCraft Commons` | 邮件显示名称。 |
| `SMTP_SECURITY` | `starttls` | `starttls`、`tls`（隐式 TLS）或 `none`。生产环境拒绝 `none`。 |
| `SMTP_TIMEOUT` | `8s` | 单次连接、握手与投递总超时。 |

配置在 API 启动时完成校验。缺少 SMTP 必填字段、端口非法、发件地址非法或安全模式未知时，API 拒绝启动，避免运行中把错误配置误报为可用。

## 3. 数据模型

`invitation_deliveries` 为每个邀请保存一条邮件投递记录：

| 字段 | 语义 |
| --- | --- |
| `invitation_id` | 与邀请一对一，唯一索引。 |
| `adapter` | `disabled` 或 `smtp`。 |
| `status` | `disabled`、`pending`、`sent`、`failed`。 |
| `attempts` | 实际调用外部适配器的次数；禁用时为 `0`。 |
| `last_attempt_at` | 最近一次真实投递尝试时间。 |
| `sent_at` | 最近一次成功时间。 |
| `last_error` | 最长 500 字符的服务端诊断信息，仅在受保护 Admin 响应中出现。 |

邀请接受不依赖投递状态。即使状态是 `failed` 或 `disabled`，有效链接仍可完成注册或已有账户接受。

## 4. API

### 4.1 创建邀请

`POST /api/v1/admin/invitations`

响应新增 `delivery`：

```json
{
  "data": {
    "id": "invitation-id",
    "email": "member@example.com",
    "role": "editor",
    "status": "pending",
    "invite_url": "/invite/only-returned-once",
    "delivery": {
      "status": "sent",
      "adapter": "smtp",
      "attempts": 1,
      "last_attempt_at": "2026-08-04T08:00:00Z",
      "sent_at": "2026-08-04T08:00:00Z"
    }
  },
  "meta": { "request_id": "..." }
}
```

投递失败时仍返回 `201`，`delivery.status=failed`，且 `invite_url` 可直接使用。未启用邮件时返回 `delivery.status=disabled`、`adapter=disabled`、`attempts=0`，不能在 UI 中显示“已发送”。

### 4.2 轮换链接并重试

`POST /api/v1/admin/invitations/{invitation_id}/email/retry`

- 要求 `membership:manage`。
- 邮件驱动禁用时返回 `409 notification.email_disabled`，不会修改 token。
- 只允许对当前组织仍为 `pending` 的邀请执行。
- 服务端在事务中生成新 token、替换哈希并使旧链接立即失效；事务提交后再尝试邮件。
- 响应返回本次唯一可见的新 `invite_url` 和新的 `delivery` 状态。
- 即使本次邮件仍失败，接口也返回 `200`，因为 token 轮换已经成功；调用方应依据 `delivery.status` 显示真实结果。

### 4.3 读取适配器状态

`GET /api/v1/admin/notifications/email/status`

要求 `organization:configure`，返回：

```json
{
  "data": {
    "driver": "smtp",
    "enabled": true,
    "configured": true,
    "from_address": "noreply@example.com",
    "from_name": "QUTCraft Commons",
    "security": "starttls"
  },
  "meta": { "request_id": "..." }
}
```

该接口永不返回 `SMTP_HOST`、端口、用户名、密码或授权码。配置修改只能通过受控部署变量完成并重启 API。

## 5. 审计与失败语义

- `membership.invite`：邀请核心记录创建成功。
- `membership.invite_email`：记录每次邮件阶段结果，`result` 为 `sent`、`failed` 或未启用时的 `skipped`。
- 适配器使用连接级总超时；SMTP 错误会截断到 500 字符，不包含请求中的邀请 token。
- 重试端点受敏感操作限流保护，避免对同一地址反复发送。
- SMTP 成功仅代表上游服务器接受邮件，不代表最终进入收件箱；退信与送达回执属于后续异步通知能力。

## 6. 本地验证

默认 Compose 使用 `EMAIL_DRIVER=disabled`，可验证链接降级路径：

```powershell
docker compose --env-file .env up -d --build api web
.\scripts\run-s2-integration.ps1
```

接入测试 SMTP 时，在 `deploy/compose/.env` 设置 SMTP 变量并重建 API。不要把真实密码提交到仓库，也不要将其写入任何 `VITE_*` 变量。
