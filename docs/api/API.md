# QUTCraft CMS API 文档

> 版本：`v1 / 0.1.0`  
> 契约源：[openapi.yaml](openapi.yaml)（OpenAPI 3.1.1）  
> 适用对象：Go 后端、Vue 前端、Apifox 测试、Swagger UI、自定义门户开发者

本文说明当前已开发的认证（Auth）、公开门户（Portal）与后台工作台（Admin）接口。它是对 OpenAPI 契约的可读补充：**路径、字段类型、必填性和状态码以 `openapi.yaml` 为最终事实来源**。本文会明确区分已落地的 CMS、资源、知识目录、成员邀请、多组织会话、门户配置与审计能力和仍在排期中的运行时能力。

> 实现状态：`/api/v1/auth/*`、多组织列举/安全切换、邀请注册/接受、Portal 公开内容读取、Admin 内容生命周期、媒体资产、用户资料、项目/成员/里程碑、知识库目录、申请审批/可禁用 ServerAdapter、门户配置、审计查询和存活/就绪探针已在 Go API 底座中实现。

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
http://localhost:18080
```

生产 Web 容器通过同源 `/api` 反向代理访问 API，普通管理员无需填写服务器地址，也不会因域名或端口变化重新打包前端。`VITE_API_BASE_URL` 仅保留给本地前后端分端口调试；生产构建始终使用真实 API，`mock` 仅能在显式启用的本地 Vite 开发服务器中使用。

### 2.2 导入 Apifox / Swagger

1. 将 [openapi.yaml](openapi.yaml) 导入 Apifox，选择 OpenAPI 3.1。
2. 创建 `local`、`staging`、`production` 环境，只切换 base URL 与管理员 token，不复制接口定义。
3. 后端应将同一份契约暴露给 Swagger UI 或在 CI 中校验其一致性。

### 2.3 Portal 请求示例

```bash
curl "http://localhost:18080/api/v1/portal/organizations/qutcraft/projects?page=1&page_size=20&status=active" \
  -H "Accept: application/json"
```

### 2.4 Admin 请求示例

```bash
curl "http://localhost:18080/api/v1/admin/dashboard" \
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
| `401` | 未带 token、token 失效、签名无法验证，或账户/当前成员关系已停用 | 尝试一次刷新；刷新失败后清理会话并进入登录。 |
| `403` | 已登录但 RBAC 角色无权操作 | 保留页面上下文，提示权限不足。 |
| `404` | 公开组织或资源不存在 | Portal 显示未找到页；不要暴露内部存在性信息。 |
| `409` | 审批对象已被其他操作处理，状态冲突 | 刷新列表后重新决定，不自动重试。 |
| `413` | 上传请求体超过服务端总大小限制 | 停止上传并提示重新选择文件，不重试原请求。 |
| `429` | 当前客户端对该接口超过频率限制 | 读取 `Retry-After`，等待窗口结束后再重试。 |
| `5xx` | 服务端或依赖故障 | 显示通用失败与 `request_id`；可由客户端有限重试只读请求。 |

### 3.4 分页与筛选

所有已分页的列表均支持：

| 参数 | 位置 | 类型 | 规则 |
| --- | --- | --- | --- |
| `page` | query | integer | 可选，最小 1，默认 1。 |
| `page_size` | query | integer | 可选，范围 1–100，默认 20。 |

后端必须对非法分页参数返回 `400`，不应悄悄截断为未知值。请求筛选条件改变时，前端应重置 `page=1`。

### 3.5 频率限制

当前单服务器部署按“客户端 IP + Gin 路由模板”使用固定窗口限流，不信任未经配置的 `X-Forwarded-For`。默认值如下，可通过服务端环境变量调整：

| 范围 | 默认值 | 环境变量 |
| --- | --- | --- |
| 注册、登录、刷新、邀请预览/接受 | 每分钟 20 次 | `AUTH_RATE_LIMIT_PER_MINUTE` |
| 公开申请提交 | 每小时 10 次 | `PUBLIC_WRITE_LIMIT_PER_HOUR` |
| 资产上传、受限服务器命令 | 每分钟 30 次 | `SENSITIVE_RATE_LIMIT_PER_MINUTE` |

受限接口的正常响应也携带 `X-RateLimit-Limit`、`X-RateLimit-Remaining` 和 `X-RateLimit-Reset`。超过限额时返回 `429 request.rate_limited` 与 `Retry-After`。当前实现针对比赛版的单 API 实例；扩展到多实例时应把计数器迁移到 Redis，不能依赖各进程独立内存。

### 3.6 Request ID、日志与探针

所有 API 请求先经过 Request ID 中间件。客户端可发送 `X-Request-ID`，但值必须是 1–64 个安全字符（字母、数字、`.`、`_`、`:`、`-`），否则服务端生成新的 UUID。最终值通过同名响应头返回，并用于响应封装、JSON 访问日志和业务审计关联。

- `GET /healthz` 只检查 API 进程存活，成功返回 `{"status":"ok"}`。
- `GET /readyz` 检查 MySQL 与 Redis；全部可用返回 `200/status=ready`，任一不可用返回 `503/status=unavailable`。
- 结构化日志不记录查询串、请求体、Authorization、邮箱、密码或外部服务凭据。

字段、排错流程和审计边界见 [API 可观测性与审计规范](observability.md)。

## 4. 认证、组织与 RBAC

### 4.1 JWT

Portal API 一律不得携带、要求或解析管理员 token。Admin API 的所有接口均要求：

```http
Authorization: Bearer <JWT>
```

后端对每次受保护请求同时验证 token 的签名、过期时间、全局账户状态和 token 所属组织的成员关系；账户或成员关系不是 `active` 时返回 `401 auth.session_inactive`。认证成功不等于有权限：每个管理端操作还必须按数据库中的实时角色授权，并限制在当前组织的资源范围内，因此角色调整无需等待旧 JWT 过期即可生效。

### 4.2 当前角色枚举

| 角色值 | 中文 | 当前语义 |
| --- | --- | --- |
| `owner` | 所有者 | 组织最高管理角色。 |
| `administrator` | 管理员 | 管理成员、审核与服务器适配能力的角色。 |
| `editor` | 编辑者 | 维护内容工作区的角色。 |
| `member` | 成员 | 普通组织成员。 |

OpenAPI 描述每个操作的认证要求，具体授权由服务端数据库 RBAC 实时执行。当前内置角色和精确权限已经落地：`editor` 具备内容/资产基础能力、知识读取和 `ai:use`；`administrator` 额外具备发布、成员、项目、知识目录、申请、审计与服务器状态权限；只有 `owner` 默认拥有 `organization:configure` 和 `server:command`。完整事实矩阵见 [RBAC 权限矩阵](../architecture/rbac-matrix.md)，前端菜单和按钮隐藏不能替代这些检查。

### 4.3 组织隔离

Portal 通过 `organization_slug` 定位组织。Admin 是“当前 token 所属组织”的工作台，因此业务 URL 中不暴露组织标识，服务端必须从认证上下文取组织 ID。唯一允许客户端提交目标组织 ID 的入口是专用的 `POST /api/v1/auth/switch-organization`；该接口会重新验证 active 成员关系、轮换 Refresh Token 并重新签发绑定目标组织的 Access Token，不能用请求体中的组织 ID 直接读取或修改业务数据。

## 5. Auth API（身份与会话）

公共前缀：`/api/v1/auth`。注册和登录使用 JSON 请求体；刷新、退出和组织切换从 `qutc_refresh` HttpOnly Cookie 读取刷新令牌；`/me` 与组织接口必须携带 Bearer JWT。Access Token 只在前端运行内存中保存，刷新令牌不会进入响应 JSON、浏览器脚本或 Web Storage。生产环境 Cookie 同时启用 `Secure`，所有环境均使用 `SameSite=Strict`。

| 方法与路径 | 认证 | 说明 |
| --- | --- | --- |
| `POST /register` | 无 | 注册用户并作为 `member` 加入默认组织。 |
| `POST /login` | 无 | 验证邮箱和密码，返回 Access Token，并通过 HttpOnly Cookie 下发 Refresh Token。 |
| `POST /refresh` | HttpOnly Cookie | 原子轮换 Refresh Token，返回新的 Access Token。 |
| `POST /logout` | HttpOnly Cookie | 撤销 Cookie 对应 Refresh Token 并清除 Cookie；重复调用保持安全。 |
| `GET /me` | Bearer JWT | 返回当前用户、当前组织和角色。 |
| `GET /organizations` | Bearer JWT | 返回当前账户具有 active 成员关系的组织、组织内角色和当前组织标记。 |
| `POST /switch-organization` | Bearer JWT + HttpOnly Cookie | 验证目标成员关系，轮换会话、保存默认组织偏好并返回绑定目标组织的新 Access Token。 |

### 5.1 当前用户资料

`PATCH /api/v1/auth/me` 更新当前用户的显示名、简介和头像地址。`display_name` 必填且最长 80 字符；`bio` 和 `avatar_url` 可选，分别最长 500 字符。接口只允许修改当前 token 对应用户，不允许修改邮箱、角色或组织关系。

`GET /api/v1/membership/history` 返回当前用户在当前组织的成员关系历史，记录状态为 `active`、`invited`、`disabled` 或 `left`，并包含 `reason` 与 `created_at`。

`POST /api/v1/membership/leave` 由当前用户主动退出组织。服务端将成员关系置为 `left`、移除组织角色并写入历史记录；已退出关系重复调用返回 `409`。

注册请求：

```json
{
  "email": "member@example.com",
  "display_name": "Member",
  "password": "at-least-12-characters"
}
```

登录与刷新成功均返回以下响应体，同时通过 `Set-Cookie` 下发或轮换 Refresh Token：

```json
{
  "data": {
    "access_token": "<jwt>",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": "user_...",
      "email": "member@example.com",
      "display_name": "Member",
      "organization_id": "org_...",
      "default_organization_id": "org_...",
      "roles": ["member"]
    }
  },
  "meta": { "request_id": "req_example" }
}
```

访问令牌短时有效且只驻留于页面内存；刷新令牌只以哈希形式保存在服务端，并在刷新时轮换。密码最少 12 个字符。认证失败、禁用用户或失效刷新令牌均不得泄露账户是否存在以外的敏感细节。

### 5.2 多组织会话切换

`GET /api/v1/auth/organizations` 返回当前账户可进入的组织。每项包含 `id`、`slug`、`name`、`short_name`、该组织内独立计算的 `roles`，以及是否与当前 Access Token 一致的 `current`。停用、退出或不存在的成员关系不会出现在列表中。

用户显式切换组织后，服务端将目标组织写入 `users.default_organization_id`。下次登录优先进入该组织；若偏好组织已失去 active 成员关系，则自动回退到最早的有效成员关系。该偏好不允许跨组织读取或由 Portal 修改。

切换请求：

```json
{
  "organization_id": "org_target"
}
```

`POST /api/v1/auth/switch-organization` 同时要求当前 Bearer JWT 和浏览器自动携带的 `qutc_refresh` Cookie。成功时服务端在同一事务中撤销旧 Refresh Token，为目标组织签发新的 Token Pair，并写入目标组织的 `auth.organization_switch` 审计；前端必须替换 Access Token、当前用户与角色，并重新加载后台数据。之后调用 `/refresh` 仍保持目标组织，不会回退到最早加入的组织。

- 目标成员关系不存在或不是 `active`：`403 organization.membership_unavailable`。
- Refresh Cookie 缺失、过期、已轮换或不属于当前用户：`401 auth.refresh_invalid`。
- 切换后若目标成员关系被停用，旧 Access Token 在下一次受保护请求时立即返回 `401`，Refresh 也不能续期。
- 所有 Admin 查询继续只使用 JWT 中经过签发的当前组织，不接受业务请求自行覆盖 `organization_id`。

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
| `contact_email` | string(email) 或空字符串 | 可公开联系邮箱；组织不公开邮箱时为空。 |
| `social_links` | array | `{ label, href }` 形式的公开 HTTPS/HTTP 外链。 |
| `is_public` | boolean | 是否允许通过 Portal API 公开访问。 |
| `updated_at` | string(date-time) | 组织资料更新时间。 |

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
	"social_links": [{ "label": "GitHub", "href": "https://github.com/QUTCraft/qutc-platform" }],
	"is_public": true,
	"updated_at": "2026-08-01T10:00:00Z"
  },
  "meta": { "request_id": "req_example" }
}
```

可能返回：`404`（组织不存在或未启用公开门户）。

后台使用 `GET /api/v1/admin/organization` 读取当前组织资料，使用 `PATCH /api/v1/admin/organization` 修改。两个端点都需要 `organization:configure`；修改在数据库事务中写入 `organization.profile_update` 审计并失效当前组织的 Portal 缓存。组织 `slug` 和内部 ID 不允许通过该接口修改。

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

每项 `KnowledgeArticle`：`id`、`title`（最多 160）、`summary`（最多 500）、`category`（最多 64）、`updated_at`、`reading_minutes`（≥1）。列表支持 `category` 与 `q`（标题、摘要或 Markdown 正文的基础匹配）筛选。

知识文章正文统一使用通用内容详情接口，不另造 `/knowledge/articles/{id}`：

`GET /api/v1/portal/organizations/{organization_slug}/content/{content_id}`
Operation ID：`getPortalContentDetail`

当 `content_id` 对应已发布且公开的 `knowledge` 内容时，响应返回 `PortalContentDetail`：`id`、`title`、`type`、`category`、`excerpt`、标准 Markdown `body`、`published_at`、`updated_at`、`reading_minutes`、可选 `asset` 与受控 `download_url`。草稿、下线内容、未公开目录文章和跨组织 ID 统一返回 `404`。正文中的 CMS 媒体必须使用服务端返回的 `/assets/{asset_id}/download` 地址；客户端不得自行拼接对象存储路径。

`GET /api/v1/portal/organizations/{organization_slug}/knowledge/directories` 返回已公开的知识库目录及其已发布文章数量。目录字段为 `id`、`name`、`slug`、`description`、`article_count`、`updated_at`；目录不是文章正文，文章仍通过上面的文章列表接口读取。

### 5.6 获取公开服务器状态

`GET /api/v1/portal/organizations/{organization_slug}/server-status`  
Operation ID：`getPortalServerStatus`

服务器适配器是可选的组织场景能力。当前只有 `qutcraft` 返回已配置的脱敏 Minecraft 状态；其他公开组织返回 `enabled=false`、`state=offline`、`apply_url=null`，不会读取或暴露 QUTCraft 的玩家数据。

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

### 5.7 提交公开加入申请

`POST /api/v1/portal/organizations/{organization_slug}/apply`

Operation ID：`submitPortalApplication`

该接口不要求登录，用于门户提交白名单或成员申请。默认 `type` 为 `whitelist`，也可以显式提交 `membership`。申请会进入当前组织的待审批队列，响应只返回申请凭证和 `pending` 状态，不返回后台审批信息或服务器内部数据。

请求体：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `type` | enum，可选 | `whitelist` 或 `membership`，默认 `whitelist`。 |
| `class_name` | string，可选 | `whitelist` 类型必填；`membership` 类型可省略，最长 120 字符。 |
| `name` | string，1–80 字符 | 申请人姓名。 |
| `game_id` | string，可选 | `whitelist` 类型必填；`membership` 类型可省略，最长 80 字符。 |
| `qq_number` | string，可选 | `whitelist` 类型必填且为 5–15 位数字；`membership` 类型可省略。 |
| `email` | email | 用于后续联系的邮箱。 |
| `note` | string，可选，最多 500 字符 | 申请补充说明。 |

`membership` 申请只要求姓名和邮箱，可用 `note` 描述申请方向；`whitelist` 申请仍要求班级/专业、Minecraft 游戏 ID 和 QQ。成功返回 `201`，数据为 `id`、`status=pending`、`submitted_at`。同一组织中 membership 按邮箱、whitelist 按邮箱或游戏 ID检测待审批重复时返回 `409 application.duplicate_pending`；请求字段不合法返回 `400`。申请列表、姓名详情、QQ 和邮箱只在受保护 Admin API 中提供。

## 7. Admin API（受认证后台）

公共前缀：`/api/v1/admin`。本节所有接口均要求 `Authorization: Bearer <JWT>`，并可能返回 `401` 或 `403`。

### 7.1 获取后台概览

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

### 7.2 内容工作区

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
| `category` | string | 可选分类/目录，最长 64 字符。 |
| `status` | enum | `draft`、`review`、`published`、`archived`。 |
| `author` | string，最多 80 字符 | 当前负责人显示名。 |
| `updated_at` | date-time | 最后修改时刻。 |
| `excerpt` | string | 可选门户摘要，最长 500 字符。 |
| `body` | string | 可选正文；管理端可见，Portal 列表不返回正文。 |
| `knowledge_directory_id` | string 或 null | 知识库内容所属目录。 |
| `revision_count` | integer | 当前内容已有的不可变修订数量。 |
| `asset` | object 或 null | 内容关联资产的管理端元数据、受控下载地址和下载统计。 |

#### 获取单条后台内容

`GET /api/v1/admin/content/{content_id}`
Operation ID：`getAdminContent`

按 ID 返回当前组织范围内的完整 `AdminContent`，需要 `content:read`。独立编辑页必须使用本接口加载正文，不能从分页列表中查找；内容不存在或属于其他组织时统一返回 `404`，避免泄露跨组织数据。

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

成功返回 `201` 和完整 `AdminContent`，其初始状态为 `draft`。

#### 编辑、发布与下线

- `PATCH /api/v1/admin/content/{content_id}`：更新草稿标题、类型、分类、摘要和正文，需要 `content:update`；已发布内容必须先下线。
- `POST /api/v1/admin/content/{content_id}/publish`：将草稿发布到 Portal，需要 `content:publish`。只有 `published` 内容会出现在公开动态接口。
- `POST /api/v1/admin/content/{content_id}/archive`：下线内容，需要 `content:archive`；下线使用 `archived` 状态，不物理删除。
- `GET /api/v1/admin/content/{content_id}/revisions` 与 `GET /api/v1/admin/content/{content_id}/revisions/{revision_id}`：读取当前组织的不可变修订摘要或完整 Markdown，需要 `content:read`。
- `POST /api/v1/admin/content/{content_id}/revisions/{revision_id}/restore`：把指定快照恢复为新的 `draft`，不会直接发布，需要 `content:update`，并写入 `content.revision_restore` 审计。

草稿创建、草稿更新、发布、下线和恢复都会生成版本号递增的快照；恢复操作保留当前内容 ID，并通过新版本记录恢复事实。015 迁移会为升级前已经存在且尚无修订的内容补齐 version 1 基线，不覆盖已有历史。当前版本接口提供完整快照，不承诺行级 diff。

#### 媒体资源

- `GET /api/v1/admin/assets`：列出当前组织的媒体资源元数据，支持 `page`、`page_size` 和按原始文件名查询的 `query`，需要 `asset:read`；适用于后台 `/admin/assets` 快捷上传工作区。
- `POST /api/v1/admin/assets`：以 `multipart/form-data` 上传字段 `file`，可选 `content_id` 建立引用，需要 `asset:upload`；单文件上限 10 MiB、整个 multipart 请求上限 11 MiB，仅允许 PNG/JPEG/WebP/PDF/ZIP/MP4。
- `POST /api/v1/admin/assets/{asset_id}/publish`：把一个尚未关联内容的文件归档为公开 `resource`，请求体包含 `title`、`kind`（`document|template|package|video`）与可选 `description`；需要同时具备 `asset:manage` 和 `content:publish`。服务端在同一事务中创建草稿修订、发布修订、关联文件并写入审计，成功返回 `201` 和已发布的 `AdminContent`；已关联文件返回 `409 asset.already_linked`。
- `GET /api/v1/admin/assets/{asset_id}/download`：管理端受权限保护的下载，需要 `asset:read`。
- `GET /api/v1/admin/assets/{asset_id}/stats`：读取当前组织资产的 `download_count` 与 `last_downloaded_at`，需要 `asset:read`。
- `DELETE /api/v1/admin/assets/{asset_id}`：删除未被内容引用的资产及其对象，需要 `asset:manage`；已关联资产返回 `409 asset.in_use`。
- `GET /api/v1/portal/organizations/{organization_slug}/assets/{asset_id}/download`：仅当资产关联的内容已发布时允许下载，不返回草稿或管理字段。

上传响应只返回资产元数据和服务端生成的下载地址，并包含下载计数。上传本身不会自动公开文件；管理员可以在 `/admin/assets` 对未关联文件执行“归档到门户”，也可以把文件关联到现有内容并通过常规内容发布流程公开。客户端不得根据原始文件名、对象存储 bucket 或资产 ID 自行拼接下载 URL。每次管理端或已公开 Portal 受控下载成功后递增计数，不记录访客身份。API 可通过 `STORAGE_DRIVER=local|s3` 使用本地目录或 MinIO/S3；存储凭据、驱动和对象键不进入公开响应。存储暂不可用时上传/下载返回 `503`，详细配置、迁移与补偿边界见 [媒体存储适配规范](storage-adapter.md)。

### 7.3 成员与角色

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

`PATCH /api/v1/admin/users/{user_id}` 更新当前组织内的成员角色和状态，需要 `membership:manage`，并写入审计事件。响应可见状态为 `active`、`invited`、`disabled`，但该接口只允许写 `active` 或 `disabled`；`invited` 只能由邀请流程产生。角色为 `owner`、`administrator`、`editor`、`member`。

成员停用与全局账户停用是两个领域：该接口只更新 `memberships.state`，不能修改 `users.state`。停用事务会撤销该用户已有 Refresh Token；旧 Access Token 在下一次受保护请求时通过实时成员校验返回 `401 auth.session_inactive`。重新启用后需要重新登录。角色修改也实时作用于 RBAC 查询，旧 Access Token 不携带角色快照，因此降级后下一次无权操作立即返回 `403`。

Owner 不能被降级或停用，管理员不能授予 `owner`，操作者也不能通过成员管理接口解除自己的管理权限。保护冲突返回稳定的 `membership.owner_protected`、`membership.owner_only` 或 `membership.self_change_forbidden`。

### 7.4 项目管理

- `GET /api/v1/admin/projects`：读取组织项目，需要 `project:read`。
- `POST /api/v1/admin/projects`：创建项目并自动把当前操作者登记为负责人，需要 `project:manage`。
- `PATCH /api/v1/admin/projects/{project_id}`：更新标题、简介、状态、标签和公开开关，需要 `project:manage`。

项目状态为 `active`、`research`、`completed`。只有 `is_public=true` 的项目会出现在 Portal API；负责人、成员关系和里程碑属于管理域。项目列表额外返回 `member_count` 和 `milestone_count`，用于后台概览。

#### 项目成员

- `GET /api/v1/admin/projects/{project_id}/members`：列出项目成员，需要 `project:read`。
- `POST /api/v1/admin/projects/{project_id}/members`：添加或更新项目成员，需要 `project:manage`。
- `PATCH /api/v1/admin/projects/{project_id}/members/{user_id}`：更新项目成员角色，需要 `project:manage`。
- `DELETE /api/v1/admin/projects/{project_id}/members/{user_id}`：移除项目成员，需要 `project:manage`；项目负责人不可移除或改角色。

添加请求：

```json
{
  "user_id": "user-id",
  "role": "contributor"
}
```

`role` 为 `member`、`contributor`、`lead`。只能加入当前组织中状态为 `active` 的成员。成员列表不会出现在公开 Portal。

#### 项目里程碑

- `GET /api/v1/admin/projects/{project_id}/milestones`：列出里程碑，需要 `project:read`。
- `POST /api/v1/admin/projects/{project_id}/milestones`：创建里程碑，需要 `project:manage`。
- `PATCH /api/v1/admin/projects/{project_id}/milestones/{milestone_id}`：编辑里程碑，需要 `project:manage`。
- `DELETE /api/v1/admin/projects/{project_id}/milestones/{milestone_id}`：删除里程碑，需要 `project:manage`。

里程碑请求字段为 `title`（最长 160）、`status` 和可选的 `due_at`。状态为 `planned`、`active`、`completed`；完成时服务端写入 `completed_at`，重新打开时清空该字段。日期必须使用 RFC3339。

### 7.5 白名单与成员申请

#### 列出申请

`GET /api/v1/admin/applications`  
Operation ID：`listAdminApplications`

列表在数据库中执行组织隔离、筛选和分页，支持：

| 查询参数 | 可选值/限制 | 说明 |
| --- | --- | --- |
| `page` | 大于等于 1 | 页码，默认 1。 |
| `page_size` | 1–100 | 每页数量，默认 20。 |
| `status` | `pending`、`approved`、`rejected` | 按审批状态筛选。 |
| `type` | `whitelist`、`membership` | 按申请类型筛选。 |
| `server_sync_status` | `none`、`pending`、`succeeded`、`failed` | 按最新服务器同步状态筛选；`none` 表示没有同步任务。 |
| `query` | 最多 80 字符 | 模糊匹配姓名、游戏 ID、邮箱和 QQ。 |

未知枚举、超长搜索词或非法分页返回 `400`。筛选必须在 `organization_id` 约束内执行，不能先读取跨组织全量数据后再由前端过滤。

`AdminApplication` 字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 申请 ID。 |
| `applicant` | string，最多 80 字符 | 申请人显示名。 |
| `type` | enum | `whitelist`（服务器白名单）或 `membership`（成员申请）。 |
| `submitted_at` | date-time | 提交时间。 |
| `note` | string，最多 500 字符 | 申请说明。 |
| `status` | enum | `pending`、`approved`、`rejected`。 |
| `class_name` | string，最多 120 字符 | 班级/专业或组织身份信息。 |
| `game_id` | string，最多 80 字符 | Minecraft 游戏 ID 或外部平台标识。 |
| `qq_number` | string | 申请人 QQ 号码，仅限受保护 Admin API。 |
| `email` | email | 申请人联系邮箱，仅限受保护 Admin API。 |
| `decided_at` | date-time / null | 审批完成时间。 |
| `decided_by` | string | 审批操作者 ID；不向 Portal 返回。 |
| `decision_reason` | string，最多 500 字符 | 通过备注或拒绝原因，仅限 Admin API。 |
| `server_sync` | object / null | 白名单批准后的服务器同步结果；成员申请或拒绝操作为 `null`。 |

#### 通过申请

`POST /api/v1/admin/applications/{application_id}/approve`  
Operation ID：`approveAdminApplication`

请求体可选传入 `{ "reason": "资料完整，符合要求。" }` 作为通过备注，最多 500 字符。成功 `200` 返回已更新的 `AdminApplication`；发生重复审批或并发状态变化时返回 `409`。服务端必须记录审批人、原状态、新状态、原因、时间和 `request_id`。白名单申请会另外创建 `server_sync` 记录，其 `status` 为 `pending`、`succeeded` 或 `failed`。审批决定一旦提交，不因外部适配器失败而回滚；失败只更新同步记录，且错误摘要必须脱敏。

#### 拒绝申请

`POST /api/v1/admin/applications/{application_id}/reject`  
Operation ID：`rejectAdminApplication`

请求体必须传入 `{ "reason": "资料需要补充。" }`，去除首尾空白后不能为空且最多 500 字符，否则返回 `400 application.decision_reason_required` 或 `application.decision_reason_too_long`。拒绝原因只对 Admin 可见，当前不提供公开申请进度查询，因此不会进入 Portal API。

#### 重试服务器同步

`POST /api/v1/admin/applications/{application_id}/server-sync/retry`
Operation ID：`retryAdminApplicationServerSync`

只允许具备 `application:approve` 权限的成员重试已批准白名单申请中最新的 `failed` 同步记录。服务端以条件更新将记录从 `failed` 原子切换为 `pending`，避免并发重复执行；`pending`、`succeeded`、非白名单或未批准申请返回 `409`。每次重试与最终结果分别写入审计，响应返回最新 `ApplicationServerSync`。外部服务仍失败时接口仍返回 `200`，但 `data.status` 为 `failed`，审批状态保持 `approved`。

完整字段表、状态机、成功/失败响应、错误码与审计规范见 [申请审批与 ServerAdapter API 规范](server-adapter.md)。

### 7.6 服务器适配器与受限 RCON

#### 获取后台服务器状态

`GET /api/v1/admin/server/status`  
Operation ID：`getAdminServerStatus`

`AdminServerStatus`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `enabled` | boolean | 是否启用服务器适配器。 |
| `adapter` | string | 服务端选择的适配器名称。 |
| `mode` | enum | `disabled`、`mock` 或 `rcon`；生产未接入 RCON 时必须返回 `disabled`，不能模拟成功。 |
| `label` | string，最多 100 字符 | 后台服务器名称。 |
| `state` | enum | `online`、`maintenance`、`offline`。 |
| `online_players` | integer，≥0 | 当前在线人数。 |
| `max_players` | integer，≥0 | 最大人数；未接入时为 0。 |
| `last_command_at` | date-time / null | 最近一次受控命令时间。 |
| `updated_at` | date-time | 适配器状态更新时间。 |

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
  "executed": false,
  "mode": "mock",
  "message": "Mock 适配器已模拟受理命令，未连接真实 RCON。",
  "executed_at": "2026-07-17T04:20:00Z"
}
```

这是高风险接口，后端至少必须做到：

1. 用服务端命令白名单与参数规则校验，不允许浏览器提交任意 RCON 字符串。
2. 限制到拥有专门权限的角色；建议将命令权限单列，不能仅以“管理员”名称判断。
3. 为每次尝试（包括被拒绝的尝试）记录操作者、组织、命令摘要、结果、IP/会话信息与 `request_id`。
4. 不在响应、日志错误或前端中泄露 RCON 密码、连接串或原始认证异常。
5. 离线或维护状态下返回明确错误，不应假装命令已成功执行。

本地 Vue 开发 Mock 只记录命令并返回模拟成功，**不会连接任何 Minecraft 服务器**。生产环境默认使用 `disabled` 适配器，命令和白名单同步会明确失败，直到接入真实 RCON。

### 7.7 成员邀请规范

成员邀请由 Admin 创建、公开链接预览，随后由邀请邮箱对应的账户接受：

1. 管理员调用 `POST /api/v1/admin/invitations`，服务端只保存 token 的 SHA-256 哈希，响应中的 `invite_url` 是唯一一次展示明文 token 的位置。
2. 邀请链接访问 `GET /api/v1/invitations/{token}`，只返回组织名称、邀请邮箱、角色、状态和过期时间，不返回操作者、成员列表或内部权限数据。
3. 已有账户登录后调用 `POST /api/v1/invitations/{token}/accept`；新用户可在 `POST /api/v1/auth/register` 传入 `invitation_token`，注册、成员关系、角色和 token 消费在同一事务内完成。
4. 默认有效期为 7 天，最大 30 天；同一组织同一邮箱只能存在一个待处理邀请，邀请不能授予 `owner`。
5. 邮箱不匹配返回 `403`；重复、已使用、过期或撤销的链接分别使用统一冲突/失效错误，不返回 token 哈希。
6. 创建响应包含真实 `delivery` 状态。邮件未启用或发送失败时邀请仍有效，Admin 必须保留复制链接入口。
7. `POST /api/v1/admin/invitations/{invitation_id}/email/retry` 会先轮换 token，使旧链接失效，再在事务外投递；邮件失败不回滚新邀请链接。
8. `GET /api/v1/admin/invitations` 按当前组织分页返回邀请和邮件状态，可使用 `status=pending|accepted|expired|revoked` 筛选，但不会返回或恢复明文 token。
9. `DELETE /api/v1/admin/invitations/{invitation_id}` 只撤销当前组织中的待处理邀请；成功后原链接立即返回 `410 invitation.revoked`，重复撤销、撤销已接受或已过期邀请返回 `409`，操作写入审计事件。
10. `POST /api/v1/admin/invitation-batches` 一次接收 1—20 条邀请并保持请求顺序逐项返回结果；单条校验、已有成员或重复邀请只影响该条，成功项分别创建、投递和审计，明文链接仍只在本次响应中展示。

### 7.8 外部适配器与通知规范

#### 1. SMTP 邮件提醒适配器
- **当前状态**：邀请邮件已实现 `disabled`/`smtp` 可替换适配器；默认关闭，关闭时明确返回 `delivery.status=disabled`。
- **投递边界**：邀请先提交，邮件在事务外同步尝试；失败单独记录为 `failed`，不会回滚邀请。
- **失败恢复**：Admin 可轮换邀请链接并重试；服务端不保存明文 token，旧链接在重试时立即失效。
- **配置状态**：`GET /api/v1/admin/notifications/email/status` 只返回当前组织的有效驱动、发件人和安全模式，不返回连接与认证凭据。
- **邀请模板**：`GET/PATCH /api/v1/admin/notifications/invitation-template` 读取或更新当前组织模板；允许变量只有 `{{organization}}`、`{{role}}`、`{{invite_url}}`、`{{expires_at}}`，主题最长 255、正文最长 4000 字符，留空恢复默认模板。
- **审批通知**：审批成功/拒绝在审批事务内写入唯一 `notification_outboxes` 事件；`GET /api/v1/admin/notifications/outbox` 查看队列，`POST /api/v1/admin/notifications/outbox/{notification_id}/retry` 重新排队失败或禁用通知。通知 worker 失败不会回滚审批决定，最多 5 次自动尝试并使用退避。
- **网页配置**：`GET/PATCH /api/v1/admin/integrations` 允许组织所有者在系统设置中维护 SMTP；授权码由 API 使用 AES-GCM 加密持久化，响应只返回“已配置”与尾号提示。首次配置前仍可沿用部署默认值。
- **连接验证**：`POST /api/v1/admin/integrations/test` 传入 `section=email`，只验证网络、TLS 和身份认证，不发送测试邮件。
- **安全约束**：SMTP 授权码绝不可进入前端静态代码、日志或 API 响应。网页输入只在保存请求中传输，保存后立即从表单清空。完整配置、模板、审批通知 Outbox 和错误语义见 [邮件与通知适配器规范](email-adapter.md)。

#### 2. Minecraft RCON 隔离与审计规范
- **当前状态**：真实 RCON 暂时搁置；生产默认使用 `disabled` Adapter，不生成虚假服务器状态或模拟成功记录。Mock Adapter 仅供自动化测试和本地开发。
- **网络隔离**: RCON 端口（`25575`）与指令下发仅在内部私有网络中打通，不向公网暴露。
- **强制审计**: 所有受控 RCON 命令下发必须包含操作者身份（`operator_id`）、操作时间戳（`executed_at`）与关联追踪 ID（`request_id`）并生成留存审计日志。

#### 3. 统一服务接入配置

系统设置中的“服务接入”覆盖门户公网地址、SMTP 和媒体存储；AI 模型接口继续由相邻的“智能体配置”页面维护。保存结果按组织隔离并即时生效：

| 方法与路径 | 用途 | 安全边界 |
| --- | --- | --- |
| `GET /api/v1/admin/integrations` | 返回有效配置、凭据掩码和部署维护项。 | 不返回任何明文密码或密钥。 |
| `PATCH /api/v1/admin/integrations` | 保存门户地址、SMTP、本地/S3 存储。 | 凭据留空保留原值；仅显式 `clear_*` 清除；写入审计。 |
| `POST /api/v1/admin/integrations/test` | 验证 `email` 或 `storage`。 | 不发邮件、不上传对象、不创建存储桶。 |

媒体上传会解析当前组织的活动存储驱动，资产元数据记录实际驱动；下载旧资产时按记录的驱动解析，因此从本地切换到 S3 后不会让已有本地文件失效。MySQL、Redis、JWT、CORS、限流和监听端口属于进程启动根基，只在网页中清楚标记为“部署维护”，不能伪装成可热修改项。RCON 继续按计划延期，生产保持明确的 `disabled` 状态。

### 7.9 门户 Manifest 配置

门户配置只允许拥有 `organization:configure` 权限的组织所有者操作：

| 方法与路径 | 说明 | 审计 |
| --- | --- | --- |
| `GET /api/v1/admin/portal/config` | 返回 `draft_manifest`、`active_manifest` 和启用元数据；首次使用时 Manifest 为 `null`。 | 无 |
| `PATCH /api/v1/admin/portal/config` | 校验并持久化 `{ "manifest": ... }` 草稿；不会改变生效版本。 | `portal.config_update` |
| `POST /api/v1/admin/portal/config/enable` | 重新校验已保存草稿，并在事务内复制为生效版本。 | `portal.config_enable` |
| `POST /api/v1/admin/portal/config/restore-default` | 在单事务中将内置 MD3 同时写为草稿和生效版本。 | `portal.config_restore_default` |

Manifest 必须符合 `qutc.portal/v1`，入口与自定义主题 Token 必须是同源静态路径，能力只能从公开白名单中选择，回退固定为 `md3`。校验失败返回 `400 portal_config.manifest_invalid`，`error.details` 是包含 `field`、`code`、`message` 的违规项数组；没有草稿时启用返回 `409 portal_config.draft_missing`。

草稿和生效版本分列存储，因此保存、导入和预览不会污染线上门户。配置持久化、RBAC、审计、管理端操作、公开运行时读取、加载超时与自动回退均已接入。

公开运行时调用 `GET /api/v1/portal/organizations/{organization_slug}/configuration`。响应只包含：

```json
{
  "data": {
    "manifest": {
      "schema": "qutc.portal/v1",
      "id": "qutc-md3",
      "version": "0.1.0",
      "display_name": "QUTCraft MD3 Portal",
      "entry": "/index.html",
      "theme": { "mode": "md3" },
      "capabilities": ["organization.read", "public_content.read"],
      "fallback": "md3"
    },
    "source": "default"
  },
  "meta": { "request_id": "..." }
}
```

`source=active` 表示返回经服务端再次校验的启用配置；`source=default` 表示无配置或配置损坏，调用方必须使用内置 MD3。该公开端点使用 `Cache-Control: no-store`，不返回 `updated_by`、`activated_by`、草稿或违规详情。

### 7.10 审计查询

`GET /api/v1/admin/audit`
Operation ID：`listAdminAuditEvents`

需要 `audit:read` 权限。服务端始终把查询绑定到当前会话的 `organization_id`，接口不接受组织参数。支持通用分页和 `action`、`target_type`、`result`、`actor_user_id`、`request_id` 精确筛选；`date_from`、`date_to` 使用 `YYYY-MM-DD` UTC 自然日并包含边界日期。

每条记录返回 `id`、`actor_user_id`、`actor_name`、`action`、`target_type`、`target_id`、`result`、`request_id` 与 `created_at`，按时间倒序。响应不包含操作者邮箱、请求正文、命令原文或服务端凭据。完整约束见 [API 可观测性与审计规范](observability.md)。

### 7.11 组织运营智能体

当前智能体基础、内容协作与活动策划闭环提供 14 条管理接口：

| 方法 | 路径 | 权限 | 作用 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/admin/ai/config` | `ai:use` | 读取当前组织策略与脱敏供应商状态。 |
| `PATCH` | `/api/v1/admin/ai/config` | `organization:configure` | 保存组织启停、配额、超时、引用与上下文限制，也可保存 OpenAI 兼容接口地址、模型名和加密 API Key。 |
| `GET` | `/api/v1/admin/ai/agents` | `ai:use` | 获取当前组织的智能体与供应商模式。 |
| `POST` | `/api/v1/admin/ai/knowledge/search` | `ai:use` ∩ `knowledge:read` | 在当前组织的知识内容中检索引用资料。 |
| `POST` | `/api/v1/admin/ai/runs` | `ai:use` ∩ `knowledge:read` | 创建异步 Markdown 提案运行。 |
| `GET` | `/api/v1/admin/ai/runs/{run_id}` | `ai:use` | 查询状态、输出、引用、模型版本与用量。 |
| `POST` | `/api/v1/admin/ai/runs/{run_id}/cancel` | `ai:use` | 取消 queued/running 运行。 |
| `GET` | `/api/v1/admin/ai/activity-plans` | `ai:use` | 分页读取当前组织的活动策划历史摘要，并返回当前登录用户自己的评分状态与均分。 |
| `GET` | `/api/v1/admin/ai/activity-plans/evaluation-summary` | `ai:use` | 汇总当前组织的五维人工评分和模型/Prompt 分组，不返回评语正文。 |
| `POST` | `/api/v1/admin/ai/activity-plans` | `ai:use` ∩ `knowledge:read` | 从结构化活动简报和固定知识引用创建策划运行。 |
| `GET` | `/api/v1/admin/ai/activity-plans/{plan_id}` | `ai:use` | 读取方案、引用、建议操作与执行状态。 |
| `GET` | `/api/v1/admin/ai/activity-plans/{plan_id}/evaluation` | `ai:use` | 读取当前用户的五维人工评分；未评分时返回 `null`。 |
| `PUT` | `/api/v1/admin/ai/activity-plans/{plan_id}/evaluation` | `ai:use` | 保存或更新当前用户评分并写入审计，不触发业务操作。 |
| `POST` | `/api/v1/admin/ai/activity-plans/{plan_id}/approve` | `ai:use` ∩ `project:manage` ∩ `content:create` | 人工批准固定建议，事务创建非公开项目、里程碑和公告草稿。 |

创建运行只读取用户显式选择、属于当前组织的 `knowledge` 内容，输出不会自动保存或发布。开发 Mock 始终返回 `provider=mock`、`mode=mock`；真实兼容模型返回 `mode=real`。模型 Key、上游地址和错误原文不会进入 API 响应。

活动策划页使用历史接口恢复方案，并把准确性、可执行性、校园适配、表达清晰度和可采用性五项 `1..5` 分评价按“方案 + 当前评审人”保存。评分与建议批准是两条独立状态线，不能替代 RBAC 或人工批准。

完整请求、响应、状态机、配置、错误码、审计和测试方式见 [组织运营智能体 API 规范](ai-agent.md)。

## 8. 前端路由与接口映射

| 页面路由 | 页面用途 | 当前调用接口 |
| --- | --- | --- |
| `/` | 门户首页 | 组织、动态、项目、资源、知识库列表、公开服务器状态。 |
| `/projects` | 公开项目 | 项目列表。 |
| `/posts/:id` | 动态详情 | 已发布内容详情。 |
| `/resources` | 资源中心 | 资源列表与受控下载入口。 |
| `/resources/:id` | 资源详情 | 已发布资源正文、文件元数据与下载入口。 |
| `/knowledge` | 知识库列表 | 知识库文章与目录列表。 |
| `/knowledge/:id` | 知识文章详情 | 已发布知识文章正文。 |
| `/invite/:token` | 成员邀请 | 公开读取邀请状态；登录后接受邀请。 |
| `/register` | 注册账户 | 可携带邀请 token 完成注册并加入组织。 |
| `/login` | 管理端登录 | 登录、刷新与当前会话接口。 |
| `/admin` | 后台概览 | `GET /admin/dashboard`。 |
| `/admin/content` | 内容工作区 | 内容创建、编辑、发布/下线与正文内资源上传。 |
| `/admin/assets` | 资源文件 | 独立批量上传、文件搜索、受控下载、未关联文件清理，以及一键归档发布到门户资源中心。 |
| `/admin/knowledge` | 知识目录 | 知识目录创建与编辑。 |
| `/admin/users` | 成员与权限 | `GET/PATCH /admin/users`、单个/批量邀请、邀请列表/撤销及邮件失败重试。 |
| `/admin/projects` | 项目管理 | 项目、项目成员和里程碑管理接口。 |
| `/admin/reviews` | 申请审核与可选服务器适配 | 所有组织可处理成员申请；仅 QUTCraft 场景展示服务器状态、同步记录与受限命令。 |
| `/admin/activity-planner` | AI 活动策划 | 结构化活动需求、知识引用、历史方案、五维评分和人工批准。 |
| `/admin/audit` | 审计记录 | `GET /admin/audit`，按组织、权限和筛选条件查询。 |
| `/admin/ai` | 智能体配置 | 读取脱敏供应商状态；组织所有者保存接口地址、模型名、API Key 和运行策略。 |
| `/admin/settings` | 门户与通知设置 | 门户 Manifest、邮件适配器状态、邀请模板和审批通知队列；不接收 SMTP 密码。 |

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

以下能力可能是后续平台所需功能，但**不是当前 API**。活动策划已经有领域专用的人工批准接口；这里未实现的是通用化工作流能力：

- 密码重置、邮箱验证、2FA、SSO，以及组织搜索和跨组织聚合（默认组织偏好已实现）。
- 内容删除、定时发布、复杂审核流、行级 diff 与版本导出（内容版本快照与恢复已实现）。
- 分片上传、病毒扫描回调与跨存储后端在线迁移。
- 成员资料编辑和多语言邀请模板。
- 申请人状态查询、撤回、补充材料；审批通知与持久化 Outbox 已实现，Webhook/企业微信仍未实现。
- RCON 命令白名单配置、命令历史查询、服务器监控明细。
- 门户历史版本回滚、资源包上传/审核与运行时健康查询。
- 通用 AI 工具调用批准/拒绝、公开知识问答与多智能体工作流。

开发上述任一能力时，必须先将端点、权限、幂等性、审计字段、错误码和敏感信息处理加入 OpenAPI，而不是仅在前端页面中临时约定。
