# 信息架构与页面路由 v1.1

> 状态：基于 2026 年 7 月 25 日实现同步
>
> 关联：[功能地图 v2](feature-map-v2.md)、[需求范围 v1](requirements-v1.md)、[API 文档](../api/API.md)

## 1. 双应用边界

公开门户与后台工作台不是同一个导航体系。它们可以由同一个 Web 工程提供，但必须使用独立布局、路由元数据、API client 和访问边界。

```text
访客
  └─ Portal Layout（公开、只读）
       └─ /, /posts, /projects, /resources, /knowledge, /apply

组织成员
  └─ Admin Layout（认证、RBAC）
       └─ /admin, /admin/content, /admin/projects, /admin/users, /admin/reviews, /admin/settings
```

门户不出现成员邮箱、角色、审核队列、草稿、RCON 面板或“后台管理”入口。后台可提供“查看门户”链接，但不得把它作为权限绕过手段。

## 2. 公开门户路由

| 路由 | 页面职责 | 数据来源 | 公开限制 |
| --- | --- | --- | --- |
| `/` | 组织介绍、最新动态、项目/资源/知识库入口、公开服务器状态 | `Organization`、`Post`、`Project`、`Resource`、`KnowledgeArticle`、`ServerStatus` | 仅 `published` 与明确公开字段。 |
| `/posts` | 已发布动态/公告目录 | `Post[]` | 不显示草稿、审核状态与内部作者字段。 |
| `/projects` | 公开项目目录 | `Project[]` | 不显示内部成员、里程碑或私人备注。 |
| `/resources` | 资源筛选与受控下载 | `Resource[]` | 不暴露对象键、长期 URL 或存储凭据。 |
| `/knowledge` | 知识库文章目录 | `KnowledgeArticle[]` | 不显示草稿与未公开文章。 |
| `/apply` | 成员/白名单公开申请表单 | `ApplicationCreate` | 只允许提交；不能读取审批队列、审核人或服务器内部状态。已由 Portal API 持久化进入后台待审批队列。 |
| `/login` | 账户登录 | Auth API | 登录不是门户管理入口；成功后按权限进入独立工作台。 |
| `/:pathMatch(.*)*` | 公开 404 | 无 | 不泄露内部路由是否存在。 |

### 2.1 门户状态

每个公开页面必须支持：加载、空数据、网络失败、组织未启用/不存在、服务器未启用和响应字段为 `null`。门户 API 的 `404` 只能解释为“没有可公开展示的组织或内容”，不能反向推断内部资源存在。

## 3. 管理工作台路由

| 路由 | 页面职责 | 当前 API | 最低权限建议 |
| --- | --- | --- | --- |
| `/admin` | 聚合指标、待审申请、近期内容、适配器状态 | `GET /api/v1/admin/dashboard` | 已认证成员。 |
| `/admin/content` | 内容列表、创建草稿、后续编辑/发布入口 | `GET/POST /api/v1/admin/content` | `content:read` / `content:create`。 |
| `/admin/projects` | 项目、公开范围、成员与里程碑 | projects / members / milestones endpoints | `project:read`；写操作需要 `project:manage`，成员/里程碑 UI 已接入。 |
| `/admin/users` | 成员、角色与状态 | `GET /api/v1/admin/users` | `membership:read`。 |
| `/admin/reviews` | 申请审核、服务器状态、受限命令 | applications / server endpoints | `application:read`、`server:read_status`。 |
| `/admin/settings` | 门户 Manifest 草稿、启用、默认 MD3 恢复与通知设置入口 | Admin Portal Configuration API；SMTP 仍未持久化 | `organization:configure`。 |

管理页面在服务端返回 `401` 时进入认证流程，在 `403` 时展示权限拒绝状态并停止后续写操作；不能仅靠前端路由守卫隐藏页面。

## 4. 内容与状态字典

### 4.1 内容类型

| 值 | 中文 | 公开呈现 |
| --- | --- | --- |
| `news` | 动态/公告 | 首页最新动态或详情页。 |
| `resource` | 资源 | 资源中心与受控下载。 |
| `knowledge` | 知识文章 | 知识库目录与详情。 |

### 4.2 内容生命周期

| 值 | 中文 | Portal 可见 | 说明 |
| --- | --- | --- | --- |
| `draft` | 草稿 | 否 | 编辑中，不进入公开索引或缓存。 |
| `review` | 待审核 | 否 | 等待具有发布权限的操作者决定。 |
| `published` | 已发布 | 是 | 可按公开范围提供给 Portal。 |
| `archived` | 已归档 | 否 | 赛后/后续 API 扩展状态；不删除审计和引用。 |

当前 Admin API 已实现 `draft`、`published` 与 `archived` 的基础操作；`review` 已进入类型和契约，但完整的送审/审核操作流仍是后续扩展，前端不可自行假设该流程已完成。

### 4.3 其他枚举

| 领域 | 值 |
| --- | --- |
| 项目状态 | `active`、`research`、`completed` |
| 资源类型 | `document`、`template`、`package`、`video` |
| 申请类型 | `whitelist`、`membership` |
| 申请状态 | `pending`、`approved`、`rejected`、`cancelled`（后续） |
| 服务器状态 | `online`、`maintenance`、`offline` |
| 成员状态 | `active`、`invited`、`disabled` |

## 5. 页面组件边界

| 层级 | 责任 | 不应承担的责任 |
| --- | --- | --- |
| `layouts/PortalLayout` | 公开导航、页脚、主题 Token、公开空错状态 | 登录态、审批、写操作、RCON。 |
| `layouts/AdminLayout` | 后台导航、账户上下文、权限提示、工作台框架 | 直接做服务端授权决定。 |
| `views/*` | 页面编排、加载/空/错状态和用户交互 | 复制 API 协议、硬编码权限。 |
| `api/*` | 类型化请求、响应封装、错误转换 | 业务展示、DOM 操作。 |
| `styles/*` | MD3 Token 与主题变量 | 通过 CSS 隐藏敏感数据。 |

## 6. 路由与 API 的演进规则

- 新页面先明确它属于 Portal 还是 Admin，再创建对应路由与布局。
- 新公开页面只能绑定 Portal operationId；需要内部数据时应改为 Admin 页面或设计脱敏公开 DTO。
- 添加路由同时定义 loading、empty、error、forbidden（Admin）和 404（Portal）状态。
- 新增内容详情、项目详情、申请表单等页面前，必须先在 OpenAPI 定义详情/写入端点，不能凭 URL 命名推断。
