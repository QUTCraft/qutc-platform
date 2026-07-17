# RBAC 权限矩阵 v1

> 状态：设计冻结  
> 关联：[需求范围 v1](../product/requirements-v1.md)、[API 文档](../api/API.md)

## 1. 权限模型

权限以 `resource:action` 命名。角色是权限集合，成员关系决定用户在某组织内是否拥有该角色。每次 Admin API 请求必须同时验证 token、成员关系、组织范围与权限；前端的按钮隐藏、路由守卫和菜单筛选均只用于体验，不能代替服务端授权。

| 资源 | 权限 | 说明 |
| --- | --- | --- |
| `portal` | `read`、`configure` | 读取/配置门户与自定义门户版本。 |
| `content` | `read`、`create`、`update`、`submit`、`publish`、`archive` | 管理内容生命周期。 |
| `asset` | `read`、`upload`、`manage` | 管理资源元数据和授权下载。 |
| `project` | `read`、`create`、`update`、`manage_members` | 管理组织项目。 |
| `knowledge` | `read`、`create`、`update`、`publish` | 管理知识库。 |
| `membership` | `read`、`invite`、`update_role`、`disable` | 查看和管理成员关系。 |
| `application` | `read`、`approve`、`reject` | 处理成员/白名单申请。 |
| `server` | `read_status`、`command`、`manage_adapter` | 服务器状态与受限适配器操作。 |
| `audit` | `read` | 阅读组织审计事件。 |
| `organization` | `read`、`configure` | 组织级配置。 |

## 2. 角色矩阵

`✓` 表示默认允许；`△` 表示仅对本人负责或被授权范围；`—` 表示默认拒绝。所有角色均可消费 Portal 的公开信息，但该能力不属于 Admin 权限。

| 能力 | 成员 | 内容编辑 | 项目负责人 | 审核员 | 管理员 | 所有者 |
| --- | --- | --- | --- | --- | --- | --- |
| 查看受授权内部内容 | △ | ✓ | △ | △ | ✓ | ✓ |
| 创建/编辑内容草稿 | — | ✓ | △ | — | ✓ | ✓ |
| 提交内容审核 | — | ✓ | △ | — | ✓ | ✓ |
| 发布/下线内容 | — | — | — | — | ✓ | ✓ |
| 上传/管理资源 | — | ✓ | △ | — | ✓ | ✓ |
| 创建/管理本人负责项目 | — | — | ✓ | — | ✓ | ✓ |
| 发布项目公开摘要 | — | — | △ | — | ✓ | ✓ |
| 管理知识文章 | — | ✓ | △ | — | ✓ | ✓ |
| 查看成员目录 | △ | △ | ✓ | △ | ✓ | ✓ |
| 邀请成员/修改角色/停用成员 | — | — | — | — | ✓ | ✓ |
| 查看申请 | — | — | — | ✓ | ✓ | ✓ |
| 通过/拒绝申请 | — | — | — | ✓ | ✓ | ✓ |
| 查看服务器后台状态 | — | — | — | △ | ✓ | ✓ |
| 执行受限服务器命令 | — | — | — | — | △ | ✓ |
| 配置服务器适配器 | — | — | — | — | — | ✓ |
| 查看审计记录 | — | — | — | △ | ✓ | ✓ |
| 配置组织与门户 | — | — | — | — | △ | ✓ |

### 2.1 范围限制

- 内容编辑只能更新自己创建或被显式授权的内容，不能直接发布。
- 项目负责人只能管理本人负责的项目及成员，不获得成员全局管理权。
- 审核员只能访问待审申请和必要材料，不能读取完整成员隐私或执行 RCON。
- 管理员默认可发布内容、管理成员和处理申请；`server:command` 与 `portal:configure` 应以额外权限开关授予。
- 所有者拥有组织最高配置权，但仍应通过二次确认和审计处理高风险操作。

## 3. 路由访问建议

| 前端路由 | 最低权限 | 无权限行为 |
| --- | --- | --- |
| `/admin` | 任一 Admin 成员 | 跳转登录或展示 403。 |
| `/admin/content` | `content:read` | 隐藏导航并拒绝 API。 |
| `/admin/users` | `membership:read` | 拒绝访问，不能只隐藏邮箱列。 |
| `/admin/reviews` | `application:read` 或 `server:read_status` | 仅显示被授权分区，服务端继续过滤。 |
| `/admin/settings` | `organization:configure` 或 `portal:configure` | 拒绝访问。 |

## 4. 强制审计事件

下列操作无论成功或失败都应写入 `AuditEvent`：登录与会话撤销、角色变更、成员停用、内容发布/下线、资源下载授权、申请审批、门户版本切换、服务器命令、服务器适配器配置变更。

审计最小字段：`id`、`organization_id`、`actor_user_id`、`action`、`target_type`、`target_id`、`result`、`request_id`、`created_at`。命令或申请说明等敏感字段仅保存经过最小化处理的摘要，且按权限可见。
