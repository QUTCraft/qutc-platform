# RBAC 权限矩阵 v1

> 状态：实现同步（2026-08-05）
> 关联：[需求范围 v1](../product/requirements-v1.md)、[API 文档](../api/API.md)

## 1. 权限模型

权限以 `resource:action` 命名。角色是权限集合，成员关系决定用户在某组织内是否拥有该角色。每次 Admin API 请求必须同时验证 token、成员关系、组织范围与权限；前端的按钮隐藏、路由守卫和菜单筛选均只用于体验，不能代替服务端授权。

| 资源 | 权限 | 说明 |
| --- | --- | --- |
| `content` | `read`、`create`、`update`、`submit`、`publish`、`archive` | 管理内容生命周期。 |
| `asset` | `read`、`upload`、`manage` | 管理资源元数据和授权下载。 |
| `project` | `read`、`manage` | 管理组织项目、成员与里程碑。 |
| `knowledge` | `read`、`manage` | 管理知识库目录；文章仍沿用内容生命周期权限。 |
| `membership` | `read`、`manage` | 查看成员目录；邀请、角色/状态修改均使用 `manage`。 |
| `application` | `read`、`approve` | 查看及处理成员/白名单申请；通过与拒绝共用 `approve`。 |
| `server` | `read_status`、`command` | 服务器状态与受限适配器操作。 |
| `audit` | `read` | 阅读组织审计事件。 |
| `organization` | `read`、`configure` | 组织级配置。 |
| `ai` | `use` | 使用内容协作与活动策划智能体；具体操作还与知识、项目和内容权限取交集。 |

## 2. 角色矩阵

`✓` 表示当前 seed 默认授予；`—` 表示默认拒绝。所有角色均可消费 Portal 的公开信息，但该能力不属于 Admin 权限。

| 能力 / 权限 | member | editor | administrator | owner |
| --- | --- | --- | --- | --- |
| 组织内部基础信息 `organization:read` | ✓ | ✓ | ✓ | ✓ |
| 内容读取/创建/更新/提交 | — | ✓ | ✓ | ✓ |
| 内容发布/下线 | — | — | ✓ | ✓ |
| 资产读取/上传 | — | ✓ | ✓ | ✓ |
| 资产管理 | — | — | ✓ | ✓ |
| 项目读取 | — | ✓ | ✓ | ✓ |
| 项目、成员与里程碑管理 | — | — | ✓ | ✓ |
| 知识读取 | — | ✓ | ✓ | ✓ |
| 知识目录管理 | — | — | ✓ | ✓ |
| 成员读取与管理 | — | — | ✓ | ✓ |
| 申请读取与处理 | — | — | ✓ | ✓ |
| 服务器后台状态 | — | — | ✓ | ✓ |
| 执行受限服务器命令 | — | — | — | ✓ |
| 审计查询 | — | — | ✓ | ✓ |
| 配置组织、门户与通知 | — | — | — | ✓ |
| 使用智能体 `ai:use` | — | ✓ | ✓ | ✓ |

### 2.1 范围限制

- `editor` 可维护当前组织中的未发布内容，但不能发布/下线、管理成员或修改组织配置。
- `administrator` 可发布内容、管理组织项目/成员、处理申请并读取审计，但默认不能执行服务器命令，也不能修改组织、门户和通知配置。
- `owner` 拥有组织最高配置权和受限服务器命令权限；Owner 成员关系不能通过普通成员管理接口被降级或停用。
- 活动方案生成要求 `ai:use ∩ knowledge:read`；人工批准进一步要求 `project:manage ∩ content:create`。评分只要求 `ai:use`，不会触发业务执行。
- 所有业务查询仍必须带当前组织范围；角色权限不能跨组织继承。

## 3. 路由访问建议

| 前端路由 | 最低权限 | 无权限行为 |
| --- | --- | --- |
| `/admin` | 任一 Admin 成员 | 跳转登录或展示 403。 |
| `/admin/content` | `content:read` | 隐藏导航并拒绝 API。 |
| `/admin/knowledge` | `knowledge:read` | 拒绝访问；写操作继续要求 `knowledge:manage`。 |
| `/admin/users` | `membership:read` | 拒绝访问，不能只隐藏邮箱列。 |
| `/admin/projects` | `project:read` / `project:manage` | 拒绝访问，服务端继续校验项目归属。 |
| `/admin/reviews` | `application:read` 或 `server:read_status` | 仅显示被授权分区，服务端继续过滤。 |
| `/admin/activity-planner` | `ai:use`；生成另需 `knowledge:read` | 无 AI 权限时拒绝；批准按钮不能代替服务端权限交集。 |
| `/admin/ai` | `ai:use` | 可查看脱敏状态；保存配置仍要求 `organization:configure`。 |
| `/admin/audit` | `audit:read` | 拒绝访问并保持当前组织范围。 |
| `/admin/settings` | `organization:configure` | 拒绝访问；门户、邀请模板和通知队列使用同一组织配置权限。 |

## 4. 强制审计事件

下列操作无论成功或失败都应写入 `AuditEvent`：登录与会话撤销、角色变更、成员停用、内容发布/下线、资源下载授权、申请审批、门户版本切换、服务器命令、服务器适配器配置变更。

审计最小字段：`id`、`organization_id`、`actor_user_id`、`action`、`target_type`、`target_id`、`result`、`request_id`、`created_at`。命令或申请说明等敏感字段仅保存经过最小化处理的摘要，且按权限可见。
