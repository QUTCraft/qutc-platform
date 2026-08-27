# 申请审批 API 规范

> 对应 OpenAPI：`docs/api/openapi.yaml`。本文只补充审批事务、权限、通知与审计语义。

## 1. 能力边界

申请模块负责五件事：

1. 公开门户提交加入申请；
2. 管理端按组织、状态、类型和关键词筛选申请；
3. 有权限的管理员人工通过或拒绝；
4. 新申请进入队列时向具备审批权限的管理员创建异步提醒；
5. 在同一事务中保存审批事实与审计记录，并将结果通知写入 Outbox。

审批不会连接、修改或执行任何游戏服务器操作。若组织需要把审批结果同步到其他系统，应由独立集成项目消费审计或通知事件，不能把外部执行塞回审批事务。

## 2. 公开提交

`POST /api/v1/portal/organizations/{organization_slug}/apply`

- 无需登录，受公开写入限流保护；
- `type` 支持 `whitelist` 与 `membership`，它们只是两类申请表单与后台分类；
- 同一组织内，相同邮箱或申请标识存在待处理记录时返回 `409 application.duplicate_pending`；
- 成功返回 `id`、`status=pending` 与 `submitted_at`，不返回内部审核字段或提醒收件人。
- 申请与 `application.submitted` Outbox 在同一事务中写入；收件人是当前组织中具备 `application:approve` 的活跃成员，按邮箱去重。没有符合条件的审核员时申请仍可提交。

## 3. 管理端查询

`GET /api/v1/admin/applications`

需要 `application:read`，并始终按当前会话的 `organization_id` 隔离。

支持参数：

| 参数 | 说明 |
| --- | --- |
| `page` / `page_size` | 标准分页，最大页大小见 OpenAPI |
| `status` | `pending`、`approved`、`rejected` |
| `type` | `whitelist`、`membership` |
| `query` | 在姓名、游戏 ID、邮箱与 QQ 中模糊搜索 |

## 4. 审批事务

- `POST /api/v1/admin/applications/{application_id}/approve`
- `POST /api/v1/admin/applications/{application_id}/reject`

两个接口都需要 `application:approve`。拒绝必须提供 1—500 字符的原因；通过备注可为空，最长 500 字符。通过时还可传入可选的 `skin_invite_code`（最长 500 字符），它会随审批结果邮件发给申请人；该值会随申请记录持久化，以保证邮件重试时仍能投递相同的邀请码。

一次成功审批在同一数据库事务中完成：

1. 以 `status=pending` 为条件原子更新申请；
2. 保存审批人、时间和原因；
3. 写入 `application.approved` 或 `application.rejected` 审计事件；
4. 邮箱存在时，写入唯一的结果通知 Outbox。

重复审批或并发竞争返回 `409 application.already_decided`。任一步数据库写入失败都会整体回滚，不会出现“状态已变但审计缺失”的半完成结果。

## 5. 通知与故障语义

申请提醒和审批结果均不依赖 SMTP 实时可用性：业务事务只创建 Outbox，后台 Worker 再按当前组织的邮件配置投递。`application.submitted` 发给有审批权限的管理员，只包含申请人姓名、邮箱、类型和提交时间，并引导登录后台查看完整资料；`application.approved/rejected` 发给申请人。投递失败会保留脱敏错误和重试状态，但不会回滚已经成立的申请或人工审批事实。

## 6. 审计与安全

- 审计包含当前组织、操作者、申请 ID、动作、结果、时间与 `request_id`；
- 门户不能读取申请列表、审批人、原因或通知队列；
- 管理接口不能通过参数覆盖当前组织；
- API 响应与日志不得泄露 SMTP、对象存储、模型供应商或其他服务端凭据。
