# QUTCraft CMS API 文档

> 版本：`v1 / 0.1.0`  
> 契约源：[openapi.yaml](openapi.yaml)（OpenAPI 3.1.1）  
> 适用对象：Go 后端、Vue 前端、Apifox 测试、Swagger UI、自定义门户开发者

本文说明当前已开发的认证（Auth）、公开门户（Portal）与后台工作台（Admin）接口。它是对 OpenAPI 契约的可读补充：**路径、字段类型、必填性和状态码以 `openapi.yaml` 为最终事实来源**。本文不会把尚未实现的内容正文编辑、文件上传、成员邀请或设置保存接口伪装成已存在能力。

> 实现状态：`/api/v1/auth/*` 已在 Go API 底座中实现。Portal 与 Admin 的内容/资源/项目/审核端点已冻结为前端 Mock 和 OpenAPI 契约，将在后续内容闭环阶段接入持久化 handler；不要因契约已存在而假定当前远程 API 已完成这些业务路由。

## 1. 设计边界

系统有两个严格分离的 API 面：

| API 面 | 前缀 | 使用者 | 数据范围 | 认证 |
| --- | --- | --- | --- | --- |
| Auth API | `/api/v1/auth` | 登录页、会话恢复 | 注册、登录、刷新、退出与当前用户 | 视端点而定 |
| Portal API | `/api/v1/portal` | 官网、资源页、Wiki、自定义门户 | 已发布且可公开的数据 | 无 |
| Admin API | `/api/v1/admin` | 内部后台 | 草稿、成员、审批、受限服务器能力 | Bearer JWT + RBAC |

Portal 不是后台的只读镜像。它不能取得成员邮箱、组织内角色、待审申请、草稿、RCON 连接信息、命令历史或原始服务器监控信息。自定义门户只能使用 Portal API，并自行遵守同一公开边界。

## 2. 快速开始

### 2.1 服务地址

OpenAPI 中的示例地址为：

```text
https://api.qutcraft.local
```

本地后端建议在 Apifox 中配置为：

```text
http://localhost:8080
```

前端通过 `VITE_API_BASE_URL` 覆盖服务地址；当前前端的默认 `mock` 模式不会发出网络请求。

### 2.2 导入 Apifox / Swagger

1. 将 [openapi.yaml](openapi.yaml) 导入 Apifox，选择 OpenAPI 3.1。
2. 创建 `local`、`staging`、`production` 环境，只切换 base URL 与管理员 token，不复制接口定义。
3. 后端应将同一份契约暴露给 Swagger UI 或在 CI 中校验其一致性。

### 2.3 Portal 请求示例

```bash
curl "http://localhost:8080/api/v1/portal/organizations/qutcraft/projects?page=1&page_size=20&status=active" \
  -H "Accept: application/json"
```

### 2.4 Admin 请求示例

```bash
curl "http://localhost:8080/api/v1/admin/dashboard" \
  -H "Accept: application/json" \
  -H "Authorization: Bearer <access-token>"
```

不要在浏览器客户端、主题包、静态页面或 Git 仓库中写入真实 token、RCON 密码、MinIO 密钥或对象存储直链。

## 3. 通用约定

### 3.1 HTTP 与 JSON

- URL 版本固定为 `/api/v1`；不以 Header、请求体或前端版本决定 API 版本。
- 请求与响应使用 UTF-8 JSON；客户端发送正文时设置 `Content-Type: application/json`。
- 时间使用 RFC 3339 / ISO 8601 UTC 时间，例如 `2026-07-17T04:10:00Z`。
- 所有标识符均为服务端生成的字符串，客户端应将它们视为不透明值，不从格式推导业务含义。
- 空值由字段契约决定：`type: [string, 'null']` 的字段可能返回 `null`；字段未在 `required` 中时也可能完全省略。

### 3.2 成功响应封装

单对象接口：

```json
{
  "data": { "id": "org_qutcraft" },
  "meta": { "request_id": "req_01J0QUTC9B7ABCD" }
}
```

列表接口：

```json
{
  "data": [{ "id": "project_cms" }],
  "meta": {
    "request_id": "req_01J0QUTC9B7ABCD",
    "page": 1,
    "page_size": 20,
    "total": 1
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `data` | object / array | 当前接口的业务数据。 |
| `meta.request_id` | string | 服务端请求追踪 ID；排错、审批和 RCON 审计必须记录。 |
| `meta.page` | integer | 当前页，从 1 开始；仅列表接口。 |
| `meta.page_size` | integer | 当前每页数量，范围 1–100；仅列表接口。 |
| `meta.total` | integer | 符合筛选条件的总条数；仅列表接口。 |

### 3.3 错误响应

任何非 2xx 响应应使用同一错误外壳：

```json
{
  "error": {
    "code": "admin.application_already_decided",
    "message": "该申请已被处理。",
    "details": {
      "application_id": "application_001"
    },
    "request_id": "req_01J0QUTC9B7ABCD"
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `error.code` | string | 稳定的、可供程序判断的错误码；不要依赖中文文案。 |
| `error.message` | string | 可展示给操作人员的错误说明。 |
| `error.details` | object | 可选的结构化补充信息；不得含密钥、token、RCON 凭据或隐私数据。 |
| `error.request_id` | string | 与服务端日志关联的追踪 ID。 |

建议错误码以领域命名，例如 `portal.organization_not_found`、`auth.token_invalid`、`admin.permission_denied`、`admin.command_not_allowed`。当前 OpenAPI 已定义下列状态码语义：

| 状态码 | 含义 | 客户端处理 |
| --- | --- | --- |
| `400` | 请求字段、长度、枚举值或命令格式不合法 | 就地展示校验信息，不重试。 |
| `401` | 未带 token、token 失效或签名无法验证 | 清理会话并进入登录或刷新流程。 |
| `403` | 已登录但 RBAC 角色无权操作 | 保留页面上下文，提示权限不足。 |
| `404` | 公开组织或资源不存在 | Portal 显示未找到页；不要暴露内部存在性信息。 |
| `409` | 审批对象已被其他操作处理，状态冲突 | 刷新列表后重新决定，不自动重试。 |
| `5xx` | 服务端或依赖故障 | 显示通用失败与 `request_id`；可由客户端有限重试只读请求。 |

### 3.4 分页与筛选

所有已分页的列表均支持：

| 参数 | 位置 | 类型 | 规则 |
| --- | --- | --- | --- |
| `page` | query | integer | 可选，最小 1，默认 1。 |
| `page_size` | query | integer | 可选，范围 1–100，默认 20。 |

后端必须对非法分页参数返回 `400`，不应悄悄截断为未知值。请求筛选条件改变时，前端应重置 `page=1`。

## 4. 认证、组织与 RBAC

### 4.1 JWT

Portal API 一律不得携带、要求或解析管理员 token。Admin API 的所有接口均要求：

```http
Authorization: Bearer <JWT>
```

后端至少验证 token 的签名、过期时间、主体身份及组织成员关系。认证成功不等于有权限：每个管理端操作还必须按角色授权，并限制在当前组织的资源范围内。

### 4.2 当前角色枚举

| 角色值 | 中文 | 当前语义 |
| --- | --- | --- |
| `owner` | 所有者 | 组织最高管理角色。 |
| `administrator` | 管理员 | 管理成员、审核与服务器适配能力的角色。 |
| `editor` | 编辑者 | 维护内容工作区的角色。 |
| `member` | 成员 | 普通组织成员。 |

OpenAPI 当前将具体权限决策留给服务端 RBAC 策略，因此不能把下面的建议视为已实现的强制授权表：内容读取/新建通常至少需 `editor`，成员读取、审批、RCON 通常至少需 `administrator`，组织最高风险设置通常需 `owner`。后端实现时应把精确的 `permission` 常量及审计规则写入架构规范并补充测试。

### 4.3 组织隔离

Portal 通过 `organization_slug` 定位组织。Admin 当前为“当前 token 所属组织”的工作台，因此 URL 中不暴露组织标识；服务端必须从认证上下文取组织 ID，绝不可接受客户端随意提交的组织 ID 来越权切换数据。

## 5. Auth API（身份与会话）

公共前缀：`/api/v1/auth`。注册、登录、刷新与退出使用 JSON 请求体；`/me` 必须携带 Bearer JWT。当前开发环境返回令牌对供前端联调，生产部署应优先将刷新令牌置于 `HttpOnly`、`Secure`、`SameSite` Cookie，并避免将其长期存入浏览器存储。

| 方法与路径 | 认证 | 说明 |
| --- | --- | --- |
| `POST /register` | 无 | 注册用户并作为 `member` 加入默认组织。 |
| `POST /login` | 无 | 验证邮箱和密码，签发访问/刷新令牌对。 |
| `POST /refresh` | 刷新令牌 | 轮换刷新令牌，签发新令牌对。 |
| `POST /logout` | 无 | 撤销提交的刷新令牌；重复调用应保持安全。 |
| `GET /me` | Bearer JWT | 返回当前用户、当前组织和角色。 |

注册请求：

```json
{
  "email": "member@example.com",
  "display_name": "Member",
  "password": "at-least-12-characters"
}
```

登录与刷新成功均返回：

```json
{
  "data": {
    "access_token": "<jwt>",
    "refresh_token": "<opaque-token>",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": "user_...",
      "email": "member@example.com",
      "display_name": "Member",
      "organization_id": "org_...",
      "roles": ["member"]
    }
  },
  "meta": { "request_id": "req_example" }
}
```

访问令牌短时有效；刷新令牌只以哈希形式保存在服务端，并在刷新时轮换。密码最少 12 个字符。认证失败、禁用用户或失效刷新令牌均不得泄露账户是否存在以外的敏感细节。

## 6. Portal API（公开只读）

公共前缀：`/api/v1/portal/organizations/{organization_slug}`。其中 `organization_slug` 必填，格式为小写字母、数字与连字符：`^[a-z0-9]+(?:-[a-z0-9]+)*$`，例如 `qutcraft`。

### 5.1 获取组织公开资料

`GET /api/v1/portal/organizations/{organization_slug}`  
Operation ID：`getPortalOrganization`

返回官网标题、介绍、联系邮箱和可公开社交链接；不返回组织内部信息。

**200 `data` 字段**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 组织内部 ID。 |
| `slug` | string | 公开 URL 标识。 |
| `name` | string | 组织完整名称。 |
| `short_name` | string | 导航、Logo 区等短名称。 |
| `tagline` | string | 门户主标语。 |
| `introduction` | string | 门户简介正文。 |
| `contact_email` | string(email) | 可公开联系邮箱。 |
| `social_links` | array | `{ label, href }` 形式的公开外链。 |

```json
{
  "data": {
    "id": "org_qutcraft",
    "slug": "qutcraft",
    "name": "QUTCraft Commons",
    "short_name": "QUTCraft",
    "tagline": "把社团正在发生的事，认真地呈现出来。",
    "introduction": "QUTCraft 是青岛理工大学的 Minecraft 社团。",
    "contact_email": "contact@qutcraft.example",
    "social_links": [{ "label": "GitHub", "href": "https://github.com/QUTCraft/qutc-platform" }]
  },
  "meta": { "request_id": "req_example" }
}
```

可能返回：`404`（组织不存在或未启用公开门户）。

### 5.2 获取公开动态

`GET /api/v1/portal/organizations/{organization_slug}/posts`  
Operation ID：`listPortalPosts`

| 查询参数 | 类型 | 说明 |
| --- | --- | --- |
| `page` / `page_size` | integer | 通用分页参数。 |
| `category` | string，最多 64 字符 | 可选，按公开分类筛选。 |

每项 `Post`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 动态 ID。 |
| `title` | string，最多 160 字符 | 标题。 |
| `excerpt` | string，最多 500 字符 | 门户摘要。 |
| `cover_url` | string(uri) / null | 可选封面地址。 |
| `category` | string，最多 64 字符 | 如公告、活动、社团动态。 |
| `published_at` | date-time | 发布时刻。 |
| `reading_minutes` | integer，≥1 | 预估阅读分钟数。 |

可能返回：`200`、`404`。

### 5.3 获取公开项目

`GET /api/v1/portal/organizations/{organization_slug}/projects`  
Operation ID：`listPortalProjects`

| 查询参数 | 类型 | 说明 |
| --- | --- | --- |
| `page` / `page_size` | integer | 通用分页参数。 |
| `status` | enum | 可选：`active`、`research`、`completed`。 |

每项 `Project`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 项目 ID。 |
| `title` | string，最多 160 字符 | 项目标题。 |
| `summary` | string，最多 500 字符 | 公开摘要。 |
| `status` | enum | `active`、`research`、`completed`。 |
| `tags` | string[]，最多 8 项 | 公开技术或主题标签，每项最多 32 字符。 |
| `updated_at` | date-time | 最近公开更新时刻。 |
| `public_url` | string(uri-reference) / null | 可选的项目公开地址。 |

### 5.4 获取公开资源

`GET /api/v1/portal/organizations/{organization_slug}/resources`  
Operation ID：`listPortalResources`

| 查询参数 | 类型 | 说明 |
| --- | --- | --- |
| `page` / `page_size` | integer | 通用分页参数。 |
| `kind` | enum | 可选：`document`、`template`、`package`、`video`。 |
| `q` | string，最多 128 字符 | 可选，标题与描述的公开检索词。 |

每项 `Resource`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 资源 ID。 |
| `title` | string，最多 160 字符 | 资源名称。 |
| `description` | string，最多 500 字符 | 公开说明。 |
| `kind` | enum | `document`、`template`、`package`、`video`。 |
| `size_bytes` | int64，≥0 | 服务端已知的字节大小。 |
| `updated_at` | date-time | 最后更新时刻。 |
| `download_url` | string(uri) | 服务端签发的短时、受控下载地址。 |

`download_url` 必须由服务端生成；客户端不得从 `id`、文件名或 bucket 信息拼接 MinIO/S3 URL，也不得缓存超出签名有效期的地址。

### 5.5 获取公开知识库文章

`GET /api/v1/portal/organizations/{organization_slug}/knowledge/articles`  
Operation ID：`listPortalKnowledgeArticles`

| 查询参数 | 类型 | 说明 |
| --- | --- | --- |
| `page` / `page_size` | integer | 通用分页参数。 |
| `category` | string，最多 64 字符 | 可选分类。 |
| `q` | string，最多 128 字符 | 可选公开检索词。 |

每项 `KnowledgeArticle`：`id`、`title`（最多 160）、`summary`（最多 500）、`category`（最多 64）、`updated_at`、`reading_minutes`（≥1）。当前只定义文章列表；**文章详情正文接口尚未定义**，前端不得自行猜测诸如 `/articles/{id}` 的路径。

### 5.6 获取公开服务器状态

`GET /api/v1/portal/organizations/{organization_slug}/server-status`  
Operation ID：`getPortalServerStatus`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `enabled` | boolean | 是否启用服务器适配器。 |
| `label` | string，最多 100 字符 | 门户可展示的服务器名称。 |
| `state` | enum | `online`、`maintenance`、`offline`。 |
| `version` | string / null | 可公开展示的版本号。 |
| `online_players` | integer / null | 公开在线人数。 |
| `max_players` | integer / null | 公开最大人数。 |
| `updated_at` | date-time | 公开状态最后同步时间。 |
| `apply_url` | string / null | 公开申请入口。 |

绝不返回 RCON 主机、端口、密码、白名单账户、命令记录、TPS 原始指标或管理端审批信息。

## 7. Admin API（受认证后台）

公共前缀：`/api/v1/admin`。本节所有接口均要求 `Authorization: Bearer <JWT>`，并可能返回 `401` 或 `403`。

### 6.1 获取后台概览

`GET /api/v1/admin/dashboard`  
Operation ID：`getAdminDashboard`

用于后台 `/admin` 页面的一次性聚合加载，不应用于公开门户。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `organization_name` | string | 当前 token 对应组织名称。 |
| `updated_at` | date-time | 聚合数据计算/同步时间。 |
| `metrics` | `DashboardMetric[]` | 不超过 12 个概览指标。 |
| `pending_applications` | `AdminApplication[]` | 当前管理员可处理的申请。 |
| `recent_content` | `AdminContent[]` | 最近编辑的内容。 |
| `server` | `AdminServerStatus` | 后台可见的服务器适配器状态。 |

`DashboardMetric` 字段为 `label`、`value`、可选 `change`、`tone`。`tone` 枚举：`primary`、`secondary`、`warning`、`neutral`，仅为展示语义，不能用作权限或告警的唯一判断依据。

### 6.2 内容工作区

#### 列出后台内容

`GET /api/v1/admin/content`  
Operation ID：`listAdminContent`

支持通用 `page`、`page_size`。返回内容包含已发布、待审核和草稿，因此不得转发给 Portal。

`AdminContent`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 内容 ID。 |
| `title` | string，最多 160 字符 | 标题。 |
| `type` | enum | `news`、`resource`、`knowledge`。 |
| `status` | enum | `draft`、`review`、`published`。 |
| `author` | string，最多 80 字符 | 当前负责人显示名。 |
| `updated_at` | date-time | 最后修改时刻。 |

#### 创建内容草稿

`POST /api/v1/admin/content`  
Operation ID：`createAdminContent`

请求体：

```json
{
  "title": "暑期建筑活动报名",
  "type": "news"
}
```

| 字段 | 必填 | 规则 |
| --- | --- | --- |
| `title` | 是 | 非空，最长 160 字符。 |
| `type` | 是 | `news`、`resource`、`knowledge` 之一。 |

成功返回 `201` 和完整 `AdminContent`，其初始状态为 `draft`。当前契约没有“编辑正文、提交审核、发布、删除、上传资源”接口；这些功能开始开发时必须先扩展 OpenAPI，再做 UI 或后端路由。

### 6.3 成员与角色

`GET /api/v1/admin/users`  
Operation ID：`listAdminUsers`

支持通用分页。`AdminUser` 字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 成员 ID。 |
| `name` | string，最多 80 字符 | 显示名。 |
| `email` | string(email) | 内部邮箱；严禁在 Portal 返回。 |
| `role` | enum | `owner`、`administrator`、`editor`、`member`。 |
| `state` | enum | `active`、`invited`、`disabled`。 |
| `joined_at` | date-time | 加入/被邀请的记录时间。 |

当前只定义读取接口。邀请成员、改角色、禁用成员等按钮在进入实际开发前应先补充对应写接口、审计规则及最后所有者保护策略。

### 6.4 白名单与成员申请

#### 列出申请

`GET /api/v1/admin/applications`  
Operation ID：`listAdminApplications`

`AdminApplication` 字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 申请 ID。 |
| `applicant` | string，最多 80 字符 | 申请人显示名。 |
| `type` | enum | `whitelist`（服务器白名单）或 `membership`（成员申请）。 |
| `submitted_at` | date-time | 提交时间。 |
| `note` | string，最多 500 字符 | 申请说明。 |
| `status` | enum | `pending`、`approved`、`rejected`。 |

#### 通过申请

`POST /api/v1/admin/applications/{application_id}/approve`  
Operation ID：`approveAdminApplication`

无请求体。成功 `200` 返回已更新的 `AdminApplication`；发生重复审批或并发状态变化时返回 `409`。服务端必须记录审批人、原状态、新状态、时间和 `request_id`。白名单申请若需要同步 Minecraft 服务端，应由服务端异步、受控地完成，不能把 RCON 凭据或原始命令返回给浏览器。

#### 拒绝申请

`POST /api/v1/admin/applications/{application_id}/reject`  
Operation ID：`rejectAdminApplication`

无请求体，成功与冲突语义同“通过申请”。当前契约没有拒绝原因字段；若要支持，应新增请求体并明确是否对申请人可见。

### 6.5 服务器适配器与受限 RCON

#### 获取后台服务器状态

`GET /api/v1/admin/server/status`  
Operation ID：`getAdminServerStatus`

`AdminServerStatus`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `enabled` | boolean | 是否启用服务器适配器。 |
| `label` | string，最多 100 字符 | 后台服务器名称。 |
| `state` | enum | `online`、`maintenance`、`offline`。 |
| `online_players` | integer，≥0 | 当前在线人数。 |
| `max_players` | integer，≥1 | 最大人数。 |
| `last_command_at` | date-time / null | 最近一次受控命令时间。 |

#### 执行受限命令

`POST /api/v1/admin/server/commands`  
Operation ID：`executeRestrictedServerCommand`

```json
{
  "command": "list"
}
```

| 字段 | 必填 | 规则 |
| --- | --- | --- |
| `command` | 是 | 非空，最长 256 字符；服务端仅接受命令白名单中的命令及参数。 |

响应 `data`：

```json
{
  "accepted": true,
  "message": "命令已受理。",
  "executed_at": "2026-07-17T04:20:00Z"
}
```

这是高风险接口，后端至少必须做到：

1. 用服务端命令白名单与参数规则校验，不允许浏览器提交任意 RCON 字符串。
2. 限制到拥有专门权限的角色；建议将命令权限单列，不能仅以“管理员”名称判断。
3. 为每次尝试（包括被拒绝的尝试）记录操作者、组织、命令摘要、结果、IP/会话信息与 `request_id`。
4. 不在响应、日志错误或前端中泄露 RCON 密码、连接串或原始认证异常。
5. 离线或维护状态下返回明确错误，不应假装命令已成功执行。

当前 Vue mock 环境只记录命令并返回模拟成功，**不会连接任何 Minecraft 服务器**。

### 6.6 外部适配器与通知规范

#### 1. SMTP 邮件提醒适配器
- **解耦设计**: 当玩家提交白名单/成员申请或管理员做审批决定时，由后端异步消息队列 Consumer 触发 SMTP 发件程序。
- **安全约束**: SMTP 授权码与服务器私钥（`SMTP_AUTH_CODE`）严格保存在后端受控环境变量中，绝不可暴露或持久化到前端静态代码或 UI 中。

#### 2. Minecraft RCON 隔离与审计规范
- **网络隔离**: RCON 端口（`25575`）与指令下发仅在内部私有网络中打通，不向公网暴露。
- **强制审计**: 所有受控 RCON 命令下发必须包含操作者身份（`operator_id`）、操作时间戳（`executed_at`）与关联追踪 ID（`request_id`）并生成留存审计日志。

## 8. 前端路由与接口映射

| 页面路由 | 页面用途 | 当前调用接口 |
| --- | --- | --- |
| `/` | 门户首页 | 组织、动态、项目、资源、知识库列表、公开服务器状态。 |
| `/projects` | 公开项目 | 项目列表。 |
| `/resources` | 资源中心 | 资源列表。 |
| `/knowledge` | 知识库列表 | 知识库文章列表。 |
| `/login` | 管理端登录 | 登录、刷新与当前会话接口。 |
| `/admin` | 后台概览 | `GET /admin/dashboard`。 |
| `/admin/content` | 内容工作区 | `GET/POST /admin/content`。 |
| `/admin/users` | 成员与权限 | `GET /admin/users`。 |
| `/admin/reviews` | 审批与 RCON | 申请列表、批准/拒绝、服务器状态、受限命令。 |
| `/admin/settings` | 组织设置 UI | 当前无持久化 API；仅 mock 页面状态。 |

## 9. 自定义门户接入规范

自定义门户是可替换的公开展示层，不是第三方后台：

- 只调用 `/api/v1/portal/organizations/{organization_slug}/...`。
- 只展示接口明确标为公开的数据；不尝试通过 ID、错误差异或未文档化路径探测私有数据。
- 对 `download_url` 使用服务端返回值，不重写、代理或推算存储路径。
- 为列表为空、组织不存在、服务器未启用、字段为 `null` 和网络失败提供降级状态。
- 对外链 `href` 做常规 URL 安全校验；对服务端文本按纯文本渲染或经过可信 HTML 清洗。
- 不在自定义门户中嵌入管理员 token，也不调用 `/api/v1/admin/*`。

## 10. 变更流程与兼容性

1. 先修改 [openapi.yaml](openapi.yaml)，并为每个接口维护稳定 `operationId`。
2. 更新本文的字段说明、示例和安全边界。
3. 更新后端请求/响应 DTO、鉴权与测试。
4. 更新 `apps/web/src/api/portal.ts` 或 `apps/web/src/api/admin.ts` 及页面调用。
5. 在 CI 运行 OpenAPI lint，并验证前端类型检查与构建。

`/api/v1` 内不得删除或改变已发布字段的类型、语义或必填性。需要破坏性变更时，新增 `v2` 前缀或保留旧字段直至迁移完成。新增可选字段通常兼容，但客户端仍应忽略未知字段、容忍可选字段缺失。

## 11. 当前未定义能力清单

以下能力可能是后续 CMS 所需功能，但**不是当前 API**：

- 密码重置、SSO、多组织主动切换与基于 Cookie 的刷新令牌投递。
- 内容正文详情、编辑、审核、发布、撤回、删除与版本历史。
- 文件直传、分片上传、资源签发、下载刷新与病毒扫描回调。
- 成员邀请、角色修改、停用/恢复、成员资料编辑。
- 申请创建、撤回、拒绝原因、申请人通知。
- RCON 命令白名单配置、命令历史查询、服务器监控明细。
- 组织设置与自定义门户 Manifest 的读取、保存、发布和回滚。

开发上述任一能力时，必须先将端点、权限、幂等性、审计字段、错误码和敏感信息处理加入 OpenAPI，而不是仅在前端页面中临时约定。
