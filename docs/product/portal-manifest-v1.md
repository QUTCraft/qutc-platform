# 自定义门户 Manifest v1

> 状态：设计冻结  
> Schema：`qutc.portal/v1`  
> 适用范围：默认 MD3 门户、QUTCraft Minecraft 第一方门户及未来第三方门户

## 1. 目的

Manifest 是门户呈现层的注册声明，不是可执行插件，也不是管理员 API 的授权凭据。它声明一个门户的入口、版本、主题资源和允许消费的公开能力。核心服务根据 Manifest 白名单提供公开内容；门户不能借此上传后端代码、读取内部数据或执行服务器命令。

## 2. 最小 Manifest

```json
{
  "schema": "qutc.portal/v1",
  "id": "qutcraft-minecraft",
  "version": "0.1.0",
  "display_name": "QUTCraft Minecraft Portal",
  "entry": "/portals/qutcraft-minecraft/index.html",
  "theme": {
    "mode": "custom",
    "tokens": "/portals/qutcraft-minecraft/theme.json"
  },
  "capabilities": [
    "organization.read",
    "public_content.read",
    "projects.read",
    "assets.read",
    "knowledge.read",
    "server.status.read"
  ],
  "fallback": "md3"
}
```

可参考 [Manifest 示例](examples/qutcraft-minecraft.portal.json) 与 [主题 Token 示例](examples/qutcraft-minecraft.theme.json)。

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

## 4. 允许能力

| 能力 | 对应 Portal 数据 | 允许用途 |
| --- | --- | --- |
| `organization.read` | 组织公开资料 | 品牌、简介、公开外链。 |
| `public_content.read` | 已发布动态 | 首页、公告、活动。 |
| `projects.read` | 公开项目 | 项目目录与摘要。 |
| `assets.read` | 公开资源与受控下载地址 | 资源列表与下载入口。 |
| `knowledge.read` | 已发布知识文章元数据 | Wiki/知识库目录。 |
| `server.status.read` | 脱敏服务器状态 | 在线状态、版本、申请入口。 |

禁止声明或授予：`admin.*`、`membership.*`、`application.*`、`audit.*`、`server.command`、数据库访问、对象存储密钥、Refresh Token、管理 JWT、任意脚本执行或任意网络代理。

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

## 7. 资源加载安全规则

- `entry` 和 `theme.tokens` 必须是同源、已发布、可校验的静态资源。
- 门户内容使用 Portal API 的结构化 JSON，不从 Manifest 注入未信任 HTML 或 JavaScript。
- Content Security Policy 至少禁止门户加载未经批准的脚本源，限制 `connect-src` 到 Portal API 与必要的受控资源域名。
- 服务端对门户包版本做状态管理：`draft`、`review`、`published`、`disabled`；只有 `published` 可被公开加载。
- 缓存键包含门户 `id`、`version` 和资源 hash；发布/回退时显式失效。

## 8. 版本兼容性

- Manifest `MAJOR` 变化表示不兼容 Schema 或能力语义变化。
- `MINOR` 仅可新增可选字段与能力；旧加载器应忽略未知可选字段。
- `PATCH` 仅修复资源或文案，不能改变能力边界。
- Portal API 的破坏性变化通过 `/api/v2/portal` 等新前缀发布；现有门户可在迁移完成前保持运行。
