# 自定义门户 Manifest v1

> 状态：设计冻结  
> Schema：`qutc.portal/v1`  
> 适用范围：默认 MD3 门户及遵循公开能力边界的自定义门户
>
> 实现状态：Go 校验器、组织级草稿/生效配置持久化、Admin RBAC、审计、管理端保存/预览/启用、公开运行时加载、MD3 自动回退与永久恢复均已落地；CSP 与现有门户兼容检查已完成。当前版本不交付第一方主题门户。

## 1. 目的

Manifest 是门户呈现层的注册声明，不是可执行插件，也不是管理员 API 的授权凭据。它声明一个门户的入口、版本、主题资源和允许消费的公开能力。核心服务根据 Manifest 白名单提供公开内容；门户不能借此上传后端代码、读取内部数据或执行服务器命令。

## 2. 最小 Manifest

```json
{
  "schema": "qutc.portal/v1",
  "id": "campus-club",
  "version": "0.1.0",
  "display_name": "Campus Club Portal",
  "entry": "/portals/campus-club/index.html",
  "theme": {
    "mode": "custom",
    "tokens": "/portals/campus-club/theme.json"
  },
  "capabilities": [
    "organization.read",
    "public_content.read",
    "projects.read",
    "assets.read",
    "knowledge.read"
  ],
  "fallback": "md3"
}
```

可参考 [Manifest 示例](examples/custom-portal.portal.json) 与 [主题 Token 示例](examples/custom-portal.theme.json)。

## 3. 字段定义

| 字段 | 必填 | 类型/规则 | 说明 |
| --- | --- | --- | --- |
| `schema` | 是 | 固定 `qutc.portal/v1` | Manifest Schema 标识。 |
| `id` | 是 | 小写字母、数字、连字符 | 门户稳定标识，不可在发布后复用到不同门户。 |
| `version` | 是 | SemVer | 门户包版本；与核心 API 版本独立。 |
| `display_name` | 是 | string，最长 80 | 管理端展示名称。 |
| `entry` | 是 | 同源绝对路径 | 已审核门户入口；不得为任意外站脚本 URL。 |
| `theme.mode` | 是 | `md3` / `custom` | 默认主题或自定义主题。 |
| `theme.tokens` | 条件必填 | 同源 JSON 路径 | `custom` 时必填；仅包含允许的展示 Token。 |
| `capabilities` | 是 | 非空 string[] | 声明的公开读取能力。 |
| `fallback` | 是 | 固定 `md3` | 加载/兼容性失败后的回退门户。 |
| `integrity` | 后续推荐 | SRI/hash | 生产发布时校验入口资源内容。 |

当前校验器还执行以下安全约束：

- `entry` 必须以单个 `/` 开头并指向 `.html`，`theme.tokens` 必须指向 `.json`。
- 路径不得包含主机名、查询参数、片段、反斜杠、编码目录穿越或非规范化路径。
- 能力最多 6 项，不得重复，且必须完全属于本文公开能力白名单。
- 可选 `integrity` 只接受 `sha256`、`sha384` 或 `sha512` SRI 格式。
- 主题颜色使用 `#RRGGBB`；圆角满足 `0 ≤ small ≤ medium ≤ large ≤ 48`。
- 字体族不得包含 `url(...)`、分号或 CSS 块，防止借主题 Token 加载外部资源或注入声明。

## 4. 允许能力

| 能力 | 对应 Portal 数据 | 允许用途 |
| --- | --- | --- |
| `organization.read` | 组织公开资料 | 品牌、简介、公开外链。 |
| `public_content.read` | 已发布动态 | 首页、公告、活动。 |
| `projects.read` | 公开项目 | 项目目录与摘要。 |
| `assets.read` | 公开资源与受控下载地址 | 资源列表与下载入口。 |
| `knowledge.read` | 已发布知识文章元数据 | Wiki/知识库目录。 |
禁止声明或授予：`admin.*`、`membership.*`、`application.*`、`audit.*`、数据库访问、对象存储密钥、Refresh Token、管理 JWT、任意脚本执行或任意网络代理。

## 5. 主题 Token

主题只能影响视觉呈现，不可改变鉴权、API 地址、权限判定或加载任意脚本。Token 推荐以 JSON 资源提供，并受 CSP、同源和发布审核约束。

```json
{
  "schema": "qutc.portal-theme/v1",
  "color": {
    "primary": "#4f46a5",
    "on_primary": "#ffffff",
    "surface": "#fdfbff",
    "on_surface": "#1b1b1f"
  },
  "shape": {
    "small": 12,
    "medium": 20,
    "large": 28
  },
  "typography": {
    "body_family": "Noto Sans SC, system-ui, sans-serif"
  }
}
```

服务端必须校验颜色、字号、圆角等字段的类型和允许范围。自定义 CSS、远程字体、背景图和第三方脚本应采用单独审核流程，不能由用户填写任意 URL 即上线。

## 6. 加载与回退流程

```text
读取已发布 Portal 配置
  → 校验 schema / version / capability / entry 同源性
  → 加载已审核的门户资源与 Token
  → 仅注入允许的 Portal API 数据
  → 加载失败、超时、哈希不符或能力不兼容
  → 回退到默认 MD3 Portal，并记录可观测事件
```

回退页面必须继续可访问公开内容，并可提供不含敏感信息的提示；不得因自定义门户失败暴露管理端、内部 API 或错误堆栈。

当前运行时通过 `GET /api/v1/portal/organizations/{organization_slug}/configuration` 读取再次校验后的生效 Manifest。没有生效配置或数据库中的生效 Manifest 已损坏时，接口返回 `source=default` 和内置 `qutc-md3`，不会向公开响应泄露草稿、操作者或审计信息。

浏览器在公开页面启动时执行以下检查：

1. 获取运行时配置，超过 1.8 秒或请求失败则保留默认 MD3。
2. 自定义入口必须返回成功的 `text/html`，并包含 `<meta name="qutc-portal-id" content="<manifest.id>">`。
3. 入口 404、超时、类型错误或标记不匹配时保留 MD3，并显示通用回退提示。
4. `/admin`、登录、注册、邀请和申请页面始终由平台自身提供，不参与门户切换。
5. 访问公开页面时添加 `?portal=md3` 可强制使用默认门户，用于恢复和排错。

### 6.1 管理端配置 API

| 方法与路径 | 权限 | 语义 |
| --- | --- | --- |
| `GET /api/v1/admin/portal/config` | `organization:configure` | 读取当前组织的草稿与生效 Manifest；未配置时两者均为 `null`。 |
| `PATCH /api/v1/admin/portal/config` | `organization:configure` | 校验并保存草稿，写入 `portal.config_update` 审计；不改变当前生效版本。 |
| `POST /api/v1/admin/portal/config/enable` | `organization:configure` | 再次校验草稿并在事务内复制为生效版本，写入 `portal.config_enable` 审计。 |
| `POST /api/v1/admin/portal/config/restore-default` | `organization:configure` | 在单个事务内将内置 MD3 同时写为草稿和生效版本，记录 `portal.config_restore_default`。 |
| `GET /api/v1/portal/organizations/{slug}/configuration` | 公开 | 只返回已校验生效 Manifest；无有效配置时返回内置 MD3。 |

管理页的“预览入口”只打开草稿声明的同源入口，不会启用草稿。草稿和生效 Manifest 分列存储，刷新或 API 重启后仍可读取。

“恢复默认 MD3”是永久恢复操作：服务端原子替换草稿和生效版本，避免两次请求导致半完成状态。`?portal=md3` 仅为当前访问的临时回退，不改数据库。

## 7. 资源加载安全规则

- `entry` 和 `theme.tokens` 必须是同源、已发布、可校验的静态资源。
- `/portals/` 下不存在的文件必须返回真实 `404`，不得由 SPA fallback 返回首页冒充门户包。
- 每个入口 HTML 必须声明与 Manifest 一致的 `qutc-portal-id` meta；不一致时不得切换。
- 门户内容使用 Portal API 的结构化 JSON，不从 Manifest 注入未信任 HTML 或 JavaScript。
- Content Security Policy 至少禁止门户加载未经批准的脚本源，限制 `connect-src` 到 Portal API 与必要的受控资源域名。
- 服务端对门户包版本做状态管理：`draft`、`review`、`published`、`disabled`；只有 `published` 可被公开加载。
- 缓存键包含门户 `id`、`version` 和资源 hash；发布/回退时显式失效。

## 8. 版本兼容性

- Manifest `MAJOR` 变化表示不兼容 Schema 或能力语义变化。
- `MINOR` 仅可新增可选字段与能力；旧加载器应忽略未知可选字段。
- `PATCH` 仅修复资源或文案，不能改变能力边界。
- Portal API 的破坏性变化通过 `/api/v2/portal` 等新前缀发布；现有门户可在迁移完成前保持运行。
