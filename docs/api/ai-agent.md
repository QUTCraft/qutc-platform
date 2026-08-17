# 组织运营智能体 API 规范

> 实施状态：内容协作闭环、服务创新类活动策划、持久运行队列、质量评估与领域专用人工批准已实现
> 更新日期：2026-08-05
> 事实来源：[OpenAPI 3.1](openapi.yaml)
> 架构与长期边界：[AI 智能体集成设计](../architecture/ai-agent-integration.md)

## 1. 当前交付范围

当前实现包含两条受控闭环，不是聊天装饰，也不会绕过 CMS、项目权限或人工批准：

1. 成员读取当前组织的运行策略与脱敏供应商状态；组织所有者可在 `/admin/ai` 保存策略、OpenAI 兼容接口地址、模型名和 API Key。
2. 管理员或编辑读取当前组织可用的 `content-copilot`。
3. 使用 `knowledge:read` 在当前组织内检索知识内容。
4. 显式选择策略允许数量的知识资料并创建异步运行。
5. 服务端调用开发 Mock 或 OpenAI-compatible 真实模型。
6. 查询运行，取得标准 Markdown、固定引用快照、模型模式和 Token 用量。
7. Admin 内容编辑器展示安全 Markdown、当前正文对比和固定引用。
8. 用户再次确认后选择“应用到编辑器但不保存”，或显式调用现有 CMS 创建接口生成 `draft`；发布仍走原有人工权限流程。
9. `activity-planner` 接收结构化活动需求与显式知识引用，生成活动方案和固定建议操作。
10. 用户逐项批准后，服务端在一个事务中创建非公开项目、所选里程碑和公告草稿，并逐项写审计。

本次同时实现：

- `ModelProvider` 供应商中立接口；
- `disabled`、明确标识的 deterministic `mock`、`openai_compatible` 三种驱动；
- `agent_definitions`、`agent_configurations`、`agent_runs`、`agent_citations` 数据模型；
- 基于 `AgentRun` 的单机持久化 worker：运行先写入 `queued`，worker 原子领取；重启保留 `queued` 任务并重新领取，上一个进程遗留的 `running` 任务明确收口为 `failed`，固定引用正文仍保存在数据库中；
- 组织级启停、超时、引用/上下文上限、每用户小时配额和配置审计；
- 异步运行与取消；
- `ai:use` RBAC、`knowledge:read` 权限交集和组织隔离；
- 创建、终态和取消审计；
- Admin 内容编辑器中的知识检索、跨检索选择、异步轮询/取消、Markdown 预览、正文对比和引用详情；
- 两级人工确认：应用到当前编辑器不会保存，创建新草稿不会发布；
- OpenAPI、Swagger、Apifox、TypeScript client 和 Compose 集成测试。

当前未实现：

- 面向任意智能体的通用 `AgentToolCall`、`AgentApproval` 工作流设计器；活动策划已实现领域专用批准接口；
- 公开 Portal 问答、自动周报、多智能体和任意外部命令工具。

## 2. 权限

| 接口 | 必要权限 |
| --- | --- |
| `GET /api/v1/admin/ai/config` | `ai:use` |
| `PATCH /api/v1/admin/ai/config` | `organization:configure` |
| `GET /api/v1/admin/ai/agents` | `ai:use` |
| `POST /api/v1/admin/ai/knowledge/search` | `ai:use` ∩ `knowledge:read` |
| `POST /api/v1/admin/ai/runs` | `ai:use` ∩ `knowledge:read` |
| `GET /api/v1/admin/ai/runs/{run_id}` | `ai:use`，并强制当前组织 |
| `POST /api/v1/admin/ai/runs/{run_id}/cancel` | `ai:use`，并强制当前组织 |
| `POST /api/v1/admin/ai/activity-plans` | `ai:use` ∩ `knowledge:read` |
| `GET /api/v1/admin/ai/activity-plans` | `ai:use`，并强制当前组织 |
| `GET /api/v1/admin/ai/activity-plans/evaluation-summary` | `ai:use`，只汇总当前组织且不返回评语正文 |
| `GET /api/v1/admin/ai/activity-plans/{plan_id}` | `ai:use`，并强制当前组织 |
| `GET /api/v1/admin/ai/activity-plans/{plan_id}/evaluation` | `ai:use`，仅返回当前用户评分 |
| `PUT /api/v1/admin/ai/activity-plans/{plan_id}/evaluation` | `ai:use`，并强制当前组织 |
| `POST /api/v1/admin/ai/activity-plans/{plan_id}/approve` | `ai:use` ∩ `project:manage` ∩ `content:create` |

`editor`、`administrator`、`owner` 默认具有 `ai:use`；`member` 默认没有。服务端从 Bearer JWT 对应的活动成员关系取得 `organization_id`，请求体和查询参数不能覆盖组织范围。

知识检索和运行引用只接受当前组织中 `type=knowledge` 的 `contents`。其他组织 ID、非知识内容 ID 和不存在的 ID 统一返回 `404 ai.source_not_found`，避免通过错误差异探测资源。

## 3. API

### 3.1 读取与更新组织配置

```http
GET /api/v1/admin/ai/config
Authorization: Bearer <access-token>
```

组织尚未保存配置时，读取接口返回由服务端部署变量派生的默认配额，以及固定默认引用/上下文上限；保存后返回 `id`、`updated_by` 和 `updated_at`。响应示例：

```json
{
  "data": {
    "id": "configuration-id",
    "enabled": true,
    "run_limit_per_hour": 20,
    "request_timeout_seconds": 30,
    "max_sources": 10,
    "max_context_characters": 30000,
    "provider": {
      "provider": "mock",
      "mode": "mock",
      "model": "mock-content-v1",
      "enabled": true,
      "configured": true
    },
    "provider_config": {
      "driver": "mock",
      "base_url": "",
      "model": "mock-content-v1",
      "api_key_configured": true,
      "api_key_hint": "••••••mock",
      "source": "server"
    },
    "updated_by": "owner-user-id",
    "updated_at": "2026-07-30T08:00:00Z"
  },
  "meta": { "request_id": "..." }
}
```

```http
PATCH /api/v1/admin/ai/config
Content-Type: application/json
Authorization: Bearer <owner-access-token>

{
  "enabled": true,
  "run_limit_per_hour": 20,
  "request_timeout_seconds": 30,
  "max_sources": 10,
  "max_context_characters": 30000,
  "provider": "openai_compatible",
  "base_url": "https://api.example.com/v1",
  "api_key": "sk-example",
  "model": "example-model"
}
```

写入请求必须包含全部五个字段，范围分别为：

| 字段 | 范围 | 生效方式 |
| --- | --- | --- |
| `enabled` | 布尔值 | `false` 时新建运行返回 `409 ai.feature_disabled`。 |
| `run_limit_per_hour` | 1—200 | 按当前用户和当前组织统计最近一小时运行。 |
| `request_timeout_seconds` | 5—120 | 创建运行时固定到该次模型调用。 |
| `max_sources` | 1—10 | 创建运行时限制显式引用数量。 |
| `max_context_characters` | 1,000—100,000 | 服务端按字符截取发送给模型的知识正文总量。 |
| `provider` | `disabled` / `openai_compatible` | 当前组织使用的驱动；生产环境不允许通过页面选择 Mock。 |
| `base_url` | 有效 HTTP(S) 根地址 | 只填写供应商根地址，服务端追加 `/chat/completions`；生产环境必须 HTTPS。 |
| `model` | 1—120 字符 | 供应商支持的模型标识。 |
| `api_key` | 由供应商定义 | 新 Key 会在服务端加密保存；留空表示保留已保存 Key 或服务端环境变量。 |

组织配置优先于服务端默认环境变量。GET 响应会返回非敏感的 `provider_config.base_url`、模型名、来源和 `api_key_hint`，但永远不会返回 API Key 原文。服务端使用应用的 `JWT_ACCESS_SECRET` 派生加密密钥保存组织 Key；因此更换该应用密钥后，既有组织 Key 需要重新录入。

### 3.2 获取智能体目录

```http
GET /api/v1/admin/ai/agents
Authorization: Bearer <access-token>
```

响应：

```json
{
  "data": {
    "agents": [
      {
        "id": "agent-definition-id",
        "key": "content-copilot",
        "name": "内容协作智能体",
        "purpose": "根据当前组织内已授权的知识资料生成带引用的 Markdown 内容提案；结果必须由人工确认。",
        "system_policy_version": "content-copilot/v1",
        "allowed_tool_keys": ["knowledge.search", "knowledge.read"],
        "model_profile": "content-generation",
        "enabled": true
      }
    ],
    "provider": {
      "provider": "mock",
      "mode": "mock",
      "model": "mock-content-v1",
      "enabled": true,
      "configured": true
    }
  },
  "meta": { "request_id": "..." }
}
```

`mode=mock` 是开发结果，不能在比赛演示中冒充真实模型；真实兼容模型返回 `provider=openai_compatible`、`mode=real`。响应永远不返回模型 API Key，但组织配置响应会返回经过校验的非敏感上游根地址。

### 3.3 检索授权知识

```http
POST /api/v1/admin/ai/knowledge/search
Content-Type: application/json
Authorization: Bearer <access-token>

{
  "query": "暑期建筑活动",
  "limit": 10
}
```

`query` 为 1—80 字符，`limit` 为 1—20，默认 10。响应只包含引用选择需要的最少字段：

```json
{
  "data": [
    {
      "source_type": "content",
      "id": "content-id",
      "title": "暑期建筑活动记录",
      "excerpt": "活动目标与执行记录。",
      "status": "draft",
      "updated_at": "2026-07-30T08:00:00Z"
    }
  ],
  "meta": { "request_id": "..." }
}
```

内部知识助手允许有权限的用户读取同组织草稿、审核中和已发布知识；这不改变 Portal 只能读取已发布内容的边界。

### 3.4 创建异步运行

```http
POST /api/v1/admin/ai/runs
Content-Type: application/json
Authorization: Bearer <access-token>

{
  "agent_key": "content-copilot",
  "task": "根据活动记录生成一篇门户动态提案",
  "context_refs": [
    { "type": "content", "id": "content-id" }
  ],
  "output_mode": "proposal"
}
```

约束：

- `task` 为 1—1000 字符；
- `context_refs` 为 1—10 条且不能重复，同时不能超过当前组织的 `max_sources`；
- 当前只支持 `type=content`，对应内容必须是当前组织的知识内容；
- 发送给模型的单条正文最多 12000 字符、单次上下文正文合计不能超过组织的 `max_context_characters`；标题、摘要和引用版本仍保留；
- 当前只支持 `output_mode=proposal`；
- 每个用户在每个组织内默认每小时最多创建 20 次运行，组织所有者可配置为 1—200；
- 组织关闭智能体时返回 `409 ai.feature_disabled`；
- 模型未启用时返回 `503`，不会降级成伪造的真实结果。

成功返回 `202 Accepted`。运行可能已从 `queued` 进入 `running`，快速 Mock 也可能在响应前完成；客户端仍应按运行状态查询，而不是假定同步完成。

### 3.5 查询运行

```http
GET /api/v1/admin/ai/runs/{run_id}
Authorization: Bearer <access-token>
```

状态机：

```text
queued → running → succeeded
                 └→ failed
queued/running ───→ canceled
```

API 进程重启时，启动恢复只把上一个进程遗留的 `running` 收口为 `failed` 并写入 `failure_code=ai.run_interrupted`；`queued` 任务保留，由单机 worker 重新领取。该实现不承诺多实例公平调度或独立消息队列。

终态响应包含：

- `output_title`、`output_excerpt`、`output_markdown`；
- `provider`、`mode`、`model`、`prompt_version`；
- `input_tokens`、`output_tokens`；
- `failure_code`、脱敏后的 `failure_message`；
- 引用资料 ID、标题、摘要和引用时的 `source_updated_at`；
- `request_id`、开始/结束/过期时间。

不保存或返回模型隐藏推理过程。Markdown 属于不可信输出，前端必须使用现有安全 Markdown 渲染流程，不能直接执行其中的 HTML、脚本或链接。

建议轮询间隔为 500—1000 ms，在 `succeeded`、`failed`、`canceled` 任一终态停止；首次保存组织配置前使用服务端 `AI_REQUEST_TIMEOUT` 默认值，之后使用组织的 `request_timeout_seconds`。

### 3.6 取消运行

```http
POST /api/v1/admin/ai/runs/{run_id}/cancel
Authorization: Bearer <access-token>
```

只有 `queued`、`running` 可取消。终态重复取消返回 `409 ai.run_not_cancelable`。取消会更新数据库状态、尝试终止当前进程内模型请求，并写入审计。

### 3.7 Admin 人工确认编排

内容编辑器的“从知识生成”工作台只编排已经定义的接口，不新增可绕过 CMS 的特殊写入口：

1. `GET /api/v1/admin/ai/config` 与 `GET /api/v1/admin/ai/agents` 检查组织策略和供应商模式。
2. `POST /api/v1/admin/ai/knowledge/search` 检索资料；用户逐条明确选择。
3. `POST /api/v1/admin/ai/runs` 创建运行，并轮询 `GET /api/v1/admin/ai/runs/{run_id}`。
4. 页面使用统一的安全 Markdown 渲染器展示结果，并排呈现当前正文与生成正文，同时展示引用 ID 和引用版本。
5. “应用到当前编辑器”只修改浏览器内表单，不发起写请求。
6. “确认并创建新草稿”经过确认框后调用现有 `POST /api/v1/admin/content`，响应状态必须为 `draft`。
7. 草稿只能由既有 `content:publish` 流程人工发布；AI API 没有发布能力。

生成正文缺失某条固定引用 ID 时，前端会补充“引用资料”章节，保证创建的草稿保留来源标识。模型输出、标题和摘要仍可在保存或发布前由人工编辑。

### 3.8 校园活动策划与人工批准

创建活动策划：

```http
POST /api/v1/admin/ai/activity-plans
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "title": "校园开源创作工作坊",
  "objective": "帮助学生完成第一次开源协作实践",
  "audience": "全校对开源与内容创作感兴趣的学生",
  "venue": "嘉陵江路校区机房",
  "starts_at": "2026-08-20T06:00:00Z",
  "ends_at": "2026-08-20T10:00:00Z",
  "expected_participants": 40,
  "budget": "500 元",
  "constraints": "需要提前确认场地、设备与安全责任人",
  "context_refs": [{ "type": "content", "id": "knowledge-content-id" }]
}
```

响应为 `202`。客户端轮询 `GET /api/v1/admin/ai/activity-plans/{plan_id}`，直到 `status` 进入 `ready`、`failed` 或 `canceled`。`ready` 响应同时包含底层 `run`、固定引用和以下建议操作键：

- `create_project`；
- `create_preparation_milestone`；
- `create_promotion_milestone`；
- `create_execution_milestone`；
- `create_retrospective_milestone`；
- `create_announcement_draft`。

人工批准：

```http
POST /api/v1/admin/ai/activity-plans/{plan_id}/approve
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "actions": [
    "create_project",
    "create_preparation_milestone",
    "create_announcement_draft"
  ]
}
```

服务端只接受上述固定键，拒绝重复键、未知键和缺少 `create_project` 的里程碑操作。批准必须满足 `ai:use`、`project:manage` 和 `content:create`，并且只能执行一次。所有选中对象在同一事务中创建：项目固定为非公开，里程碑固定为 `planned`，公告固定为 `draft`；任一创建或审计失败都会整体回滚。接口没有发布、审批、邮件或任意外部命令能力。

活动策划历史按组织分页返回：

```http
GET /api/v1/admin/ai/activity-plans?page=1&page_size=20
Authorization: Bearer <access-token>
```

摘要包含方案状态、活动时间、模型、Prompt 版本、已经创建的业务对象 ID，以及当前登录用户自己的 `has_my_evaluation` / `my_evaluation_score`。这两个评分字段只用于构建“待我评分”队列，不会泄露其他评审人的评分或评语。管理端用摘要恢复历史方案；详情、固定引用和正文仍由单条查询返回。

方案进入 `ready` 或 `applied` 后，具有 `ai:use` 的用户可以保存自己的五维人工评分：

```http
PUT /api/v1/admin/ai/activity-plans/{plan_id}/evaluation
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "accuracy": 5,
  "feasibility": 4,
  "campus_fit": 5,
  "clarity": 4,
  "adoptability": 3,
  "notes": "场地容量仍需线下确认"
}
```

五项分数均为 `1..5`，服务端计算 `overall_score`。同一用户对同一方案重复 `PUT` 会更新原记录；`GET /api/v1/admin/ai/activity-plans/{plan_id}/evaluation` 只返回当前用户的记录，尚未评分时返回 `data: null`。评分写入 `ai.activity_plan_evaluate` 审计事件，但不会批准建议、创建对象或调用外部服务。

`GET /api/v1/admin/ai/activity-plans/evaluation-summary` 汇总当前组织的评价次数、已评方案数、总均分、五个维度均分，以及按 `provider + mode + model + prompt_version` 分组的评价数量和均分。该接口不接受组织参数，不返回评审人 ID 或 `notes`，用于管理端展示比赛质量证据，而不是导出成员评价原文。

## 4. 部署配置与组织策略

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `AI_PROVIDER` | API 直接运行：`disabled`；Compose：`mock` | `disabled`、`mock`、`openai_compatible`。生产环境禁止 `mock`。 |
| `AI_BASE_URL` | 空 | 兼容服务的 API 根，例如 `https://models.example.com/v1`；服务端追加 `/chat/completions`。 |
| `AI_API_KEY` | 空 | 只注入 API 服务，不进入前端、OpenAPI 示例、日志或响应。 |
| `AI_MODEL` | `mock-content-v1` | 模型标识。 |
| `AI_REQUEST_TIMEOUT` | `30s` | 组织尚未保存策略时的单次上游调用默认超时。 |
| `AI_RUN_LIMIT_PER_HOUR` | `20` | 组织尚未保存策略时的每用户小时默认配额。 |

生产启动规则：

- `AI_PROVIDER=mock` 被拒绝；
- `openai_compatible` 必须配置 HTTPS `AI_BASE_URL`、`AI_API_KEY` 和非空模型；
- `disabled` 可以安全启动，但创建运行返回 `503 ai.provider_unavailable`；
- API Key 只存在于服务端内存和上游 `Authorization` 请求头中。

兼容驱动使用 `POST {AI_BASE_URL}/chat/completions`，发送 `model`、`messages`、`temperature`，读取 `choices[0].message.content` 与标准 `usage` 字段。

组织策略和组织级 Provider 配置保存在 `agent_configurations`。API Key 以 AES-GCM 密文保存，日志、审计和响应均不包含原文。每次创建运行都会重新读取组织策略和 Provider 配置，因此保存后无需重启 API；留空 API Key 不会覆盖已有密钥。

## 5. 错误码

| HTTP | 错误码 | 语义 |
| --- | --- | --- |
| `400` | `ai.config_validation_failed` | 组织配置缺字段、字段类型错误或超出允许范围。 |
| `400` | `ai.validation_failed` | 查询、任务、引用数量、类型或输出模式不合法。 |
| `401` | `auth.token_missing` / `auth.token_invalid` | 未认证或会话失效。 |
| `403` | `admin.permission_denied` | 缺少 `ai:use` 或 `knowledge:read`。 |
| `404` | `ai.agent_not_found` | 当前组织没有指定的启用智能体。 |
| `404` | `ai.source_not_found` | 引用不存在、类型不符、无权访问或属于其他组织。 |
| `404` | `ai.run_not_found` | 运行不存在或不属于当前组织。 |
| `409` | `ai.feature_disabled` | 当前组织已关闭智能体功能。 |
| `409` | `ai.run_not_cancelable` | 运行已进入终态。 |
| `429` | `ai.run_quota_exceeded` | 数据库小时配额已用完；接口层限流也可能返回通用 `rate_limit.exceeded`。 |
| `503` | `ai.provider_unavailable` | 模型驱动关闭或配置不完整。 |

异步模型失败不会把上游响应、API Key 或内部错误原文返回给客户端。查询运行时通过 `failure_code` 区分 `ai.provider_timeout`、`ai.provider_disabled`、`ai.provider_invalid_response`、`ai.provider_unavailable`。

## 6. 审计

| action | result | 时机 |
| --- | --- | --- |
| `ai.config_update` | `success` | 组织所有者保存运行策略或 Provider 配置。 |
| `ai.run_create` | `accepted` | 运行和引用快照已在事务内创建。 |
| `ai.run_create` | `failed` / `quota_exceeded` | 模型关闭或小时配额拒绝。 |
| `ai.run_result` | `succeeded` / `failed` | 异步运行进入终态。 |
| `ai.run_cancel` | `success` | 用户取消 queued/running 运行。 |

运行审计目标为 `target_type=agent_run`；配置审计目标为 `target_type=agent_configuration`。两者记录当前组织、操作者和原始 HTTP `request_id`，但不保存完整 Prompt、知识正文、模型凭据或隐藏推理。

## 7. 验证

无外部服务检查：

```powershell
.\scripts\run-quality-gate.ps1
```

Compose 重建后运行 AI 真实链路：

```powershell
docker compose --env-file deploy/compose/.env -f deploy/compose/docker-compose.yml up -d --build api web
.\scripts\run-s6-agent-integration.ps1
```

S6 集成测试验证登录与 RBAC、配置读取与持久化、停用后拒绝运行、重新启用、智能体目录、知识检索、跨组织引用拒绝、Prompt Injection 隔离、异步 Mock 终态、引用快照和审计。活动策划专项还验证 `activity-planner/v2`、Editor 可读但无权批准、里程碑依赖校验、部分批准、重复批准、审批 Request ID 逐对象审计，以及真实 MySQL 外键故障下项目/草稿/审计整体回滚。测试同时连续三轮验证“生成本身不创建内容 → 人工确认创建 draft → Portal 仍不可见 → 人工发布后可见且保留来源 ID → 下线后不可见”。
