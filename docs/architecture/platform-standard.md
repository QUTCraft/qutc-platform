# QUTCraft CMS 统一平台规范 v0.1

> 状态：草案冻结，适用于 2026-07-14 至 2026-08-31 的 MVP 与比赛演示。
>
> 适用对象：QUTCraft 技术部门、通用组织管理平台、QUTCraft Minecraft 第一方门户及未来第三方自定义门户。

## 1. 设计原则

### 1.1 一个核心，多个门户

业务核心只负责身份、组织、内容、项目、知识库、资源、审批、审计和适配器接口。门户是消费这些能力的呈现层：

- 默认门户使用统一 Material Design 3（MD3）。
- QUTCraft 门户可以使用 Minecraft 主题，但不能复制核心业务逻辑。
- 自定义门户必须通过 Portal API 与 Manifest 接入，不得直接访问数据库、管理 API、RCON 或服务端密钥。
- 没有可用的自定义门户、版本不兼容或加载失败时，系统必须回退到默认 MD3 门户。

### 1.2 比赛版与社团版的边界

比赛版强调组织数字化管理的通用能力；社团版增加 Minecraft 服务器适配器。Minecraft 相关内容应位于适配器、门户主题和演示数据层，而不是散落在用户、内容、项目等通用模块中。

### 1.3 服务端是权限与事实来源

浏览器隐藏按钮不等于权限控制。所有写入、导出、审核、资源访问和服务器命令都必须在 API 服务端重新鉴权、校验资源归属并记录审计事件。

## 2. 技术基线

| 层 | 约定 |
| --- | --- |
| 前端 | Vue 3 + Vite + TypeScript + Element Plus；门户主题使用独立 CSS Token 与组件层 |
| 后端 | Go + Gin；按 handler、service、repository、model、platform 分层 |
| ORM | GORM；所有结构变更必须有可回滚/可追踪迁移 |
| 数据库 | MySQL；时间统一使用 UTC 存储，展示层按用户/组织时区转换 |
| 缓存 | Redis；只缓存可重建数据，必须定义 TTL 与失效策略 |
| 文件 | MinIO/S3 兼容对象存储；浏览器使用短时签名 URL，不接触长期密钥 |
| 部署 | Docker Compose + Nginx；配置通过环境变量注入 |
| 认证 | JWT Access Token + Refresh Token；RBAC 权限模型 |
| 文档 | Swagger/OpenAPI；接口变更必须同步文档与示例 |

## 3. 仓库与模块规范

```text
apps/api/cmd/server       # API 启动入口
apps/api/internal/config  # 配置读取与校验
apps/api/internal/handler # HTTP 参数解析、鉴权入口、响应组装
apps/api/internal/service # 业务用例与事务边界
apps/api/internal/repository # 数据访问
apps/api/internal/model   # 数据模型与 DTO 映射
apps/api/internal/platform # cache、database、storage、server adapter
apps/api/migrations       # 数据库迁移
apps/web/src/api          # API client 与类型
apps/web/src/components   # 可复用组件
apps/web/src/layouts      # 门户/后台布局
apps/web/src/router       # 路由与权限元数据
apps/web/src/stores       # 会话与跨页面状态
apps/web/src/styles       # MD3 与主题 Token
apps/web/src/types        # 前端领域类型
apps/web/src/views        # 页面级视图
docs/architecture         # 架构与工程规范
docs/api                  # API 使用与示例
docs/product              # 产品文档与页面演示
```

## 4. 领域模型

核心实体至少包括：`Organization`、`User`、`Role`、`Permission`、`Membership`、`Project`、`Page`、`Post`、`KnowledgeArticle`、`Asset`、`Application`、`AuditEvent`、`Portal`、`ServerAdapter`。

- 所有组织级数据必须带 `organization_id`，查询必须带租户范围。
- 公共内容使用 `draft`、`published`、`archived` 状态，不用物理删除代替下线。
- 审核申请使用 `pending`、`approved`、`rejected`、`cancelled`，状态变化必须有操作者和时间。
- 资源保存原始文件名、MIME、大小、哈希、对象键、上传者、可见范围和引用关系。
- 外部适配器的状态只能作为同步结果保存，不能覆盖通用成员/用户事实。

## 5. HTTP API 规范

### 5.1 路径与方法

- 基础路径：`/api/v1`。
- 资源使用复数名词：`/users`、`/projects`、`/pages`、`/assets`。
- 使用 HTTP 方法表达语义：`GET` 查询、`POST` 创建/动作、`PATCH` 局部更新、`DELETE` 软删除或撤销。
- 管理 API 与 Portal API 分离：`/api/v1/admin/*`、`/api/v1/portal/*`。
- 需要动作语义时使用明确的子路径，例如 `POST /applications/{id}/approve`，不要用含糊的 `POST /update`。

### 5.2 统一响应

成功响应：

```json
{
  "data": {},
  "meta": { "request_id": "req_01..." }
}
```

列表响应：

```json
{
  "data": [],
  "meta": {
    "request_id": "req_01...",
    "page": 1,
    "page_size": 20,
    "total": 0
  }
}
```

错误响应：

```json
{
  "error": {
    "code": "application.not_approvable",
    "message": "当前申请不能被审核",
    "details": {},
    "request_id": "req_01..."
  }
}
```

### 5.3 状态码与错误码

- `200` 查询/更新成功，`201` 创建成功，`202` 异步任务已接受，`204` 无响应体成功。
- `400` 参数格式错误，`401` 未认证，`403` 无权限，`404` 不存在或对当前租户不可见，`409` 状态冲突，`422` 业务校验失败，`429` 频率限制，`500` 未知服务错误。
- 错误码使用稳定的 `domain.reason`，前端不得根据中文 `message` 判断逻辑。
- `500` 不得向客户端泄露 SQL、栈信息、RCON 响应中的敏感内容。

### 5.4 查询、排序与幂等

- 列表默认 `page=1&page_size=20`，`page_size` 最大 100。
- 排序必须指定白名单字段，例如 `sort=-created_at`；服务端拒绝任意 SQL 片段。
- 搜索参数统一为 `q`，过滤参数按字段命名。
- 创建外部副作用或重试可能重复的请求支持 `Idempotency-Key`。
- 所有响应返回 `request_id`；日志、审计和前端错误上报使用同一 ID 关联。

## 6. 认证、RBAC 与审计

- Access Token 短时有效，Refresh Token 可撤销并轮换；浏览器优先使用 HttpOnly、Secure、SameSite Cookie。
- JWT 只携带稳定身份信息，不把完整权限列表当作长期事实；关键请求服务端检查当前权限。
- 权限命名格式：`resource:action`，例如 `content:publish`、`application:approve`、`server:command`。
- 角色是权限集合，成员关系是用户加入组织的事实；不要把“管理员”硬编码为布尔字段。
- 登录、角色变更、内容发布、资源下载授权、审核、RCON 命令和门户配置变更必须写入 `AuditEvent`。

## 7. 前端规范

### 7.1 统一信息架构

- 管理端与比赛版门户均使用 MD3 Token：颜色、字号、间距、圆角、阴影和状态色集中定义。
- 页面必须具备 loading、empty、error、forbidden 四种状态。
- 表单必须显示字段级错误、提交中状态、成功反馈和重复提交保护。
- 所有破坏性操作需要确认；审核动作必须显示目标、结果和失败原因。
- 交互控件触摸目标不小于 44px，支持键盘聚焦和 `prefers-reduced-motion`。

### 7.2 门户与管理端分离

- `layouts/PortalLayout` 只消费公开 Portal API。
- `layouts/AdminLayout` 允许管理 API，但所有页面路由声明所需权限。
- 主题只负责呈现，不得在主题中复制认证、审核、上传、RCON 等业务逻辑。
- 页面内容优先使用结构化数据；主题通过 slots、组件映射和 Token 改变表现。

### 7.3 命名与类型

- Vue 组件使用 PascalCase，页面使用 `*View.vue`，API client 使用 `*.api.ts`。
- 前端类型与 OpenAPI schema 对齐，不使用 `any` 绕过接口变更。
- 日期、金额、状态等领域格式化集中在 `utils/formatters`，不要散落在模板内。

## 8. 自定义门户规范

### 8.1 Portal Manifest v1

自定义门户以 Manifest 注册，不允许通过上传任意后端代码扩展：

```json
{
  "schema": "qutc.portal/v1",
  "id": "qutcraft-minecraft",
  "version": "0.1.0",
  "display_name": "QUTCraft Minecraft Portal",
  "entry": "/portals/qutcraft/index.html",
  "theme": {
    "mode": "custom",
    "tokens": "/portals/qutcraft/theme.json"
  },
  "capabilities": ["public_content.read", "projects.read", "assets.read", "server.status.read"],
  "fallback": "md3"
}
```

### 8.2 能力边界

允许：公开页面、公告、项目、知识库、公开资源、组织信息、服务器公开状态。

禁止：用户密码、Refresh Token、管理端数据、成员隐私、审核写入、文件签名密钥、数据库、Redis、RCON。

所有能力必须由服务端根据 Manifest 白名单签发；前端自行拼接管理 API 路径视为不兼容实现。

### 8.3 版本与回退

- Manifest 使用 SemVer；`MAJOR` 表示不兼容 API/资源契约，`MINOR` 增加可选能力，`PATCH` 修复实现。
- Portal API 版本与核心 API 分开演进，例如 `/api/v1/portal`。
- 加载失败、能力缺失、资源超时或 schema 不兼容时，显示默认 MD3 门户，不显示空白页。
- 生产环境只启用已审核、已校验哈希和已发布状态的门户版本。

## 9. 文件、缓存与适配器

- 文件上传先申请上传凭证，服务端校验 MIME、大小、扩展名、哈希和组织归属；下载按可见范围重新鉴权。
- Redis Key 使用 `qutc:{env}:{domain}:{id}`，必须设置 TTL；写操作要使相关缓存失效。
- `ServerAdapter` 至少提供 `HealthCheck`、`GetStatus`、`AddWhitelist`、`RemoveWhitelist`、`Execute`，每个动作返回结构化结果和外部请求 ID。
- RCON 只允许白名单命令或明确的管理员权限；命令超时、重试和失败必须可观测。

## 10. 测试、发布与变更

- API：服务层单测、handler 契约测试、权限矩阵测试、关键流程集成测试。
- 前端：路由权限、表单校验、空/错状态、门户回退和核心流程冒烟测试。
- 发布前必须完成迁移检查、`.env.example` 校验、Swagger 生成、Docker Compose 启动和安全检查。
- 破坏性 API 变更先新增版本，不直接修改已发布字段语义。
- PR 必须说明影响模块、数据库/接口变更、权限影响、测试方式和回滚方法。

## 11. MVP 验收清单

- [ ] 非 Minecraft 组织可以创建内容、项目、成员和资源并在 MD3 门户展示。
- [ ] QUTCraft Minecraft 门户使用同一套 Portal API，但视觉与导航可以独立。
- [ ] 管理端所有写操作有权限校验和审计记录。
- [ ] 白名单审批在 Mock Adapter 下可重复演示，在真实 RCON 下不会误报成功。
- [ ] 自定义门户只能使用声明的公开能力，加载失败会回退 MD3。
- [ ] 新机器按文档可以启动，关键流程可以连续三次成功。
