# 邮件与通知适配器规范

## 1. 目标与边界

当前邮件能力包含两条失败语义不同的路径：成员邀请邮件，以及申请审批结果通知。

邀请邮件是成员邀请流程的可选外部适配器，不是邀请事务的组成部分：

- 邀请记录和 token 哈希先写入数据库，提交成功后才尝试邮件投递。
- SMTP 连接、认证或收件失败不会删除邀请，也不会回滚成员邀请业务状态。
- 未启用 SMTP 时，Admin 仍会取得一次性 `invite_url`，可复制链接完成完整加入流程。
- 浏览器只读取适配器状态和单次投递结果，不读取 SMTP 主机、用户名、密码或授权码。
- 服务端始终只持久化邀请 token 的 SHA-256 哈希，不为邮件重试保存明文 token。

申请审批通知使用持久化 Outbox：

- 审批事务只写入唯一通知事件，不在事务中连接 SMTP。
- 单机 worker 在事务提交后领取事件并发送；发送失败不会回滚已经生效的审批决定。
- SMTP 禁用、失败、重试次数和脱敏错误均可由具有组织配置权限的管理员查看。
- 当前没有申请人自助状态页、Webhook、企业微信、退信处理或多实例 worker，这些能力继续延期。

## 2. 服务端默认值与网页配置

环境变量只负责首次启动与未保存组织级配置时的默认值。拥有 `organization:configure` 权限的管理员可在“系统设置 → 服务接入”保存 SMTP，配置按组织即时生效，不需要重启 API。SMTP 密码使用服务端密钥派生的 AES-GCM 密钥加密保存；GET 响应只返回掩码提示。

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

### 3.1 邀请投递

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

### 3.2 审批通知 Outbox

`notification_outboxes` 为审批结果保存可重试事件：

| 字段 | 语义 |
| --- | --- |
| `event_type` | `application.approved` 或 `application.rejected`。 |
| `target_type` / `target_id` | 当前为 `application` 与申请 ID；与事件类型组成唯一约束，避免重复审批通知。 |
| `recipient_email` | 审批时申请记录中的邮箱，仅 Admin 可见。 |
| `status` | `pending`、`sending`、`sent`、`failed`、`disabled`。 |
| `attempts` | 已领取并尝试处理的次数；自动处理上限为 5。 |
| `available_at` | 允许下一次领取的时间；失败时按尝试次数退避。 |
| `last_attempt_at` / `sent_at` | 最近尝试和成功时间。 |
| `last_error` | 最长 500 字符的安全错误摘要，不保存 SMTP 原文或凭据。 |

API 进程重启不会丢失 `pending`/`failed` 事件。当前 worker 与 API 同进程、按单实例部署设计；多实例公平领取和独立消息中间件不在比赛版承诺内。

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

该状态接口永不返回主机、端口、用户名、密码或授权码。完整配置使用：

- `GET /api/v1/admin/integrations`
- `PATCH /api/v1/admin/integrations`
- `POST /api/v1/admin/integrations/test`，请求 `{ "section": "email" }`

测试连接验证 DNS/TCP、TLS/STARTTLS 与 SMTP AUTH，但不会发送邮件。网页读取只得到是否已配置与尾号提示，保存后浏览器表单立即清空密码。

### 4.4 组织级邀请模板

`GET /api/v1/admin/notifications/invitation-template` 读取当前组织模板，`PATCH` 更新主题与正文；两者均要求 `organization:configure`。

允许变量只有：

- `{{organization}}`
- `{{role}}`
- `{{invite_url}}`
- `{{expires_at}}`

主题最长 255 字符，正文最长 4000 字符。未知变量、残缺花括号和超长内容返回 `400 notification.template_invalid`；主题或正文留空表示使用服务端默认模板。保存动作写入 `notification.invitation_template_update` 审计。模板不允许读取申请材料、成员隐私或 SMTP 配置。

### 4.5 审批通知队列

- `GET /api/v1/admin/notifications/outbox`：要求 `organization:configure`，按当前组织分页，可用 `status` 精确筛选。
- `POST /api/v1/admin/notifications/outbox/{notification_id}/retry`：只允许 `failed` 或 `disabled` 事件；清空安全错误、重置尝试次数并重新置为 `pending`，写入 `notification.retry` 审计。

SMTP 未启用时，worker 将事件标记为 `disabled`，而不是伪报发送成功。SMTP 失败时事件进入 `failed` 并按退避时间自动重试；达到 5 次后保留失败记录等待人工处理。列表和重试接口不返回邮件正文、密码、主机或上游错误原文。

## 5. 审计与失败语义

- `membership.invite`：邀请核心记录创建成功。
- `membership.invite_email`：记录每次邮件阶段结果，`result` 为 `sent`、`failed` 或未启用时的 `skipped`。
- `notification.invitation_template_update`：组织邀请模板更新成功。
- `notification.retry`：审批通知被人工重新排队。
- 适配器使用连接级总超时；SMTP 错误会截断到 500 字符，不包含请求中的邀请 token。
- 重试端点受敏感操作限流保护，避免对同一地址反复发送。
- SMTP 成功仅代表上游服务器接受邮件，不代表最终进入收件箱；退信与送达回执属于后续异步通知能力。

## 6. 本地验证

默认 Compose 使用 `EMAIL_DRIVER=disabled`，可验证邀请链接降级和审批通知 `disabled` 可见路径：

```powershell
docker compose --env-file .env up -d --build api web
.\scripts\run-s2-integration.ps1
```

接入测试 SMTP 时，在 `deploy/compose/.env` 设置 SMTP 变量并重建 API。不要把真实密码提交到仓库，也不要将其写入任何 `VITE_*` 变量。

重建后可在 `/admin/settings` 查看适配器状态、编辑邀请模板并检查审批通知队列。验证真实发送时应使用测试邮箱和专用 SMTP 凭据；SMTP 接受不等于最终送达，测试记录必须在结束后按组织数据清理策略处理。
