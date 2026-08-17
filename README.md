# QUTCraft Platform

QUTCraft Platform 是“社团智枢 Commons Agent”的工程底座：一套面向校园社团与民间组织的 AI 活动策划、内容与组织协作平台。成员可以从组织知识生成带引用的活动方案，人工逐项批准后创建项目、里程碑与公告草稿；访客仍只通过独立门户读取已发布公开信息。

青岛理工大学 QUTCraft Minecraft 社团是本项目的首个真实落地场景。它验证了平台既可以保持通用的组织数字化能力，也可以通过公开 API 构建具有社团特色的门户，同时保持门户、内容与组织协作之间的清晰边界。

> 当前仓库处于比赛版收口阶段。前端支持契约 Mock 与远程 API 两种模式；Go API、001—018 版本化迁移、JWT/RBAC、Compose 环境，以及内容、资源、项目、预创建成员邀请、申请审核和 AI 活动策划闭环均已落地。当前重点是新机器复现、视觉/故障回归和交付冻结。

## 核心原则

- **门户与后台分离**：公开门户只展示已发布的公开信息；后台不作为门户页面的一部分。
- **API-first**：Portal 与 Admin 接口均以 OpenAPI 3.1 契约为先，前端类型与接口文档同步演进。
- **安全边界明确**：成员邮箱、草稿、审核和服务端凭据只允许在受控管理 API 中处理。
- **可替换门户**：默认提供 MD3 门户；组织可以基于 Portal API 开发自己的公开门户主题。
- **安全回退**：自定义入口需通过同源、类型和门户 ID 标记探测；超时或资源错误自动保留 MD3，`?portal=md3` 可强制恢复。
- **人工审批可追溯**：加入申请只在平台内由管理员处理，审批事实、通知与审计均可独立验证。
- **AI 建议而非越权执行**：智能体只读取显式选择的组织知识并提出类型化操作；项目、里程碑和草稿必须由人批准，内容永不自动发布。

## 当前能力

### 公开门户

- MD3 风格的组织首页、公开项目、资源中心与知识库列表。
- 组织动态、项目、资源与知识文章的公开数据，以及独立的加入申请入口。
- 响应式导航和独立的公开页面布局。

### 管理工作台

- 独立的后台侧边导航与工作台布局，入口为 `/admin`。
- Dashboard：组织概览、待办申请、项目状态、近期内容与活动运营入口。
- 内容工作区：标准 Markdown 编辑、实时预览、Markdown 导入、图片/附件插入、发布/下线，以及不可变修订历史和恢复为新草稿。
- 资源文件工作区：后台提供独立的拖拽/多选上传入口、上传队列、文件搜索、受控下载链接复制和未关联文件清理；可选择将文件关联到门户内容。
- 成员与权限：查看成员、组织角色和状态；创建单个/批量邀请时预创建待激活账户与成员关系，并显示可选 SMTP 的真实投递结果、组织级邀请模板与失败重试。
- 申请审核：处理服务器加入/组织成员申请，审批只更新平台状态并写入审计与通知队列。
- 审计记录：按动作、对象、结果、Request ID 与日期查询当前组织的管理操作。
- 组织公开资料：维护名称、简称、标语、介绍、公开邮箱、社交链接和公开状态，修改立即作用于门户并留存审计。
- 门户 Manifest 与通知设置：真实读取、草稿保存、JSON 导入、同源预览、独立启用、邀请邮件模板，以及申请审批通知 Outbox 的状态与重试。
- AI 活动策划：结构化活动需求、组织知识检索、带引用方案、建议操作审查，以及人工批准创建非公开项目、里程碑和公告草稿。
- 质量证据：活动方案历史恢复、按评审人保存的五维人工评分、组织级模型/Prompt 质量汇总，以及不保存生成正文的真实模型稳定性演练。
- 内容协作智能体：从选定知识生成 Markdown，支持预览、对比、引用核对和人工确认创建草稿。
- 多组织会话：同一账户可切换有效组织，Access/Refresh 上下文随之轮换，并持久化默认组织偏好。
- 资源运营数据：受控下载累计次数与最近下载时间，按组织隔离并由后台查询。

> 前端仍保留契约 Mock 供离线演示；Compose 默认使用 remote 模式连接真实 API/MySQL。邀请邮件使用可选 SMTP Adapter，默认关闭且不会伪报发送成功。

### 身份与工程底座

- Go + Gin + GORM API 服务，包含存活/就绪探针、Request ID、脱敏 JSON 访问日志与统一 JSON 响应。
- MySQL 版本化迁移：SQL 内嵌于 API 二进制、按序执行并记录 `schema_migrations`，兼容旧 AutoMigrate 数据卷升级。
- 注册、登录、HttpOnly Refresh Cookie 轮换、退出撤销与当前会话接口；Access Token 只驻留于前端内存。
- JWT Bearer 鉴权与基于 `resource:action` 的 RBAC 中间件。
- 前端登录页、会话恢复、后台路由守卫与 Mock 演示账号。
- MySQL、Redis、API、Web 及可选 MinIO 的 Docker Compose 开发环境；媒体卷由一次性 `media-init` 服务初始化权限，API 保持 nonroot 运行。

> Auth、Portal 与 Admin 的基础业务 API 已由 `apps/api` 实现；Web 可通过 `VITE_API_MODE=remote` 联调真实 Compose API，也可切换回契约 Mock 做离线页面演示。

## 技术栈

| 层级 | 当前选型 |
| --- | --- |
| Web | Vue 3、Vite、TypeScript、Vue Router、Element Plus、Material Design 3 设计语言 |
| API | Go、Gin、GORM、Swagger / OpenAPI |
| 数据 | MySQL、Redis、MinIO（可选服务） |
| 认证与授权 | JWT、RBAC |
| 部署 | Docker Compose、Nginx / OpenResty |
| 接口协作 | OpenAPI 3.1、Apifox、Swagger UI |

## 快速开始

### 环境要求

- Node.js 20 或更高版本
- pnpm 9 或更高版本
- Python 3.10 或更高版本（运行契约检查脚本）
- Go 1.22 或更高版本（运行 API）
- Docker Compose v2（运行完整本地依赖环境）

### 启动前端

```bash
cd apps/web
pnpm install
pnpm dev
```

Vite 启动后访问终端提示的本地地址，通常是：

```text
http://localhost:8082
```

常用页面：

| 地址 | 说明 |
| --- | --- |
| `/` | 公开门户首页 |
| `/projects` | 公开项目 |
| `/resources` | 资源中心 |
| `/knowledge` | 知识库 |
| `/invite/:token` | 成员邀请预览与接受 |
| `/register` | 注册账户并接受邀请 |
| `/admin` | 管理工作台概览 |
| `/admin/content` | 内容工作区 |
| `/admin/assets` | 资源文件快捷上传与媒体资产管理 |
| `/admin/knowledge` | 知识目录管理 |
| `/admin/projects` | 项目、成员与里程碑 |
| `/admin/users` | 成员与权限 |
| `/admin/reviews` | 申请审核与服务器适配 |
| `/admin/activity-planner` | AI 活动策划、历史方案与人工评分 |
| `/admin/ai` | 智能体供应商与组织策略 |
| `/admin/audit` | 审计查询 |
| `/admin/settings` | 门户 Manifest 与通知设置 |

访问 `/admin` 会先进入登录页。默认 Mock 演示账号为：

```text
邮箱：admin@qutcraft.local
密码：demo-admin-pass
```

### 校验前端

```bash
cd apps/web
pnpm check
```

该命令执行 TypeScript 类型检查和生产构建。

### 环境变量

复制 [`apps/web/.env.example`](apps/web/.env.example) 为 `apps/web/.env.local` 后配置：

```dotenv
# remote：请求真实后端 API；仅本地 Vite 开发可显式改为 mock
VITE_API_MODE=remote
# 生产与 Compose 留空，使用 Web 内置同源 /api；本地分端口联调可填 http://localhost:8080
VITE_API_BASE_URL=
VITE_ORGANIZATION_SLUG=qutcraft
```

生产构建始终强制使用真实 API，即使构建命令遗漏 `VITE_API_MODE` 也不会回退到 Fixture。`mock` 仅供显式启动的本地 Vite 开发服务器使用。不要将生产 token、数据库连接或对象存储密钥写入任何 `VITE_*` 变量；这些变量会被打包到浏览器端。

### 启动 API（本机 Go）

```bash
cd apps/api
copy .env.example .env
go mod tidy
go run ./cmd/server
```

API 默认监听 `http://localhost:8080`；`GET /healthz` 检查进程存活，`GET /readyz` 检查 MySQL 与 Redis 是否可接收流量。首次启动会创建基础表、默认组织、角色与权限；开发环境可使用 `.env` 中的 `BOOTSTRAP_ADMIN_*` 创建第一个所有者。生产环境必须替换 JWT 密钥和引导密码，且不得提交 `.env`。

### 启动完整开发环境（Docker Compose）

```bash
cd deploy/compose
copy .env.example .env
docker compose up --build
```

默认启动 MySQL、Redis、API 与 Web；如果使用仓库当前 `deploy/compose/.env`，Web/API 地址分别为 `http://localhost:8082` 和 `http://localhost:18080`。

需要对象存储时使用：

```bash
cd deploy/compose
docker compose --profile storage up --build
```

MinIO 启动后，组织所有者可在 Admin 的“系统设置 → 服务接入”选择 S3 / MinIO、填写地址与凭据并验证连接；无需重启 API。配置完成后进入“资源文件”即可使用拖拽/多选快捷上传，上传文件由 API 写入 MinIO，浏览器不会直连对象存储。默认 `local` 继续使用受控媒体卷。SMTP 与门户公网地址也在同一页面维护，AI 模型接口在“智能体配置”中维护。所有敏感凭据由 API 加密保存且不会回传浏览器；完整边界见 [媒体存储适配规范](docs/api/storage-adapter.md) 与 [邮件适配规范](docs/api/email-adapter.md)。

### 备份与恢复演练

本地卷部署可创建包含 MySQL、媒体归档、逐表数量和 SHA-256 清单的备份，并安全恢复到临时数据库/卷验证：

```powershell
.\scripts\backup-compose.ps1
.\scripts\verify-backup-restore.ps1 -BackupPath .\backups\qutcraft-<UTC时间>

# 一次性创建、恢复验证并清理临时备份
.\scripts\run-backup-restore-rehearsal.ps1
```

默认备份会短暂暂停 API 写入；恢复验证绝不会覆盖当前数据库或媒体卷。S3/MinIO 媒体需要使用 bucket 版本控制、复制或提供方快照，不能把本地空卷当作完整备份。完整安全流程见 [Compose 备份与恢复手册](docs/operations/backup-restore.md)。

需要本地 Swagger UI 时使用：

```bash
docker compose --profile docs up
```

随后访问 `http://localhost:8081`。Swagger UI 加载仓库中的 `docs/api/openapi.yaml`，因此它与 Apifox 的事实来源一致；使用仓库当前 Compose 配置联调时，在 Swagger/Apifox 中将 API Server 覆盖为 `http://localhost:18080`。

当前电脑若未安装 Go 或 Docker，可继续使用前端 Mock 模式；安装工具链后按以上命令验证 API 和 Compose。

### 演示数据

数据库迁移、默认组织、RBAC 和可选的 `BOOTSTRAP_ADMIN_*` 所有者属于基础引导。内容、知识目录、项目、里程碑和申请属于演示数据，默认不会自动创建。需要本地或比赛演示数据时，在 `deploy/compose/.env` 中显式设置：

```dotenv
DEMO_SEED_ENABLED=true
DEMO_SEED_PROFILE=qutcraft
# 同时准备 QUTCraft 与 Campus Commons，并让同一引导管理员加入两个组织。
DEMO_SEED_MULTI_ORGANIZATION=true
# Web 未携带 organization 查询参数时使用的默认公开组织。
VITE_ORGANIZATION_SLUG=qutcraft
```

`DEMO_SEED_PROFILE=qutcraft` 提供 QUTCraft 社团演示资料；比赛或通用产品演示使用 `generic`，并建议同时将 `DEFAULT_ORGANIZATION_SLUG` 改为通用标识。开启 `DEMO_SEED_MULTI_ORGANIZATION` 后，系统会在主 Profile 之外创建另一套独立命名空间的演示组织；同一引导管理员可在 Admin 顶栏切换。Profile 和多组织开关只影响缺失的演示记录，不会覆盖后台已经编辑的资料。

Admin 中的“查看门户”与“返回公开门户”会携带当前组织 slug，例如 `/?organization=campus-commons`。公开门户会在当前浏览器会话中保持该组织上下文，后续访问动态、项目、资源和知识库仍只读取对应组织的 Portal API。

然后重新创建 API 容器：

```bash
cd deploy/compose
docker compose up -d --build api
```

演示 seed 使用固定 ID，只补充缺失记录；重复启动不会生成重复数据，也不会覆盖已有记录的人工修改。每个组织会创建 6 条内容（其中 3 条为活动策划可引用知识）、3 个知识目录、3 个项目及 2 个里程碑，以及待处理、已通过、已拒绝三种申请。第二组织使用独立 ID，不会覆盖主组织数据。

生产环境必须同时保持 `DEMO_SEED_ENABLED=false` 和 `DEMO_SEED_MULTI_ORGANIZATION=false`，且必须替换引导密码与 JWT 密钥。关闭 seed 不会删除已有数据；如需清理演示记录，应使用受控清理脚本或重建专用演示数据库，不能在生产库直接执行通配删除。

## API 与接口协作

[docs/api/openapi.yaml](docs/api/openapi.yaml) 是 Portal 与 Admin API 的唯一机器可读契约源，支持直接导入 Apifox 或 Swagger / Redoc 工具。

- [完整 API 文档](docs/api/API.md)：认证、RBAC、响应封装、字段说明、错误语义、示例与安全边界。
- [申请审批 API 规范](docs/api/application-review.md)：审批事务、通知、错误码与审计约束。
- [邮件与通知适配器规范](docs/api/email-adapter.md)：SMTP 服务端配置、邀请模板、审批通知 Outbox、失败重试与凭据安全边界。
- [API 可观测性与审计规范](docs/api/observability.md)：Request ID、结构化日志、存活/就绪探针和审计查询边界。
- [数据库迁移与回退规范](docs/operations/database-migrations.md)：版本账本、旧数据卷基线、升级演练和备份回退边界。
- [比赛演示运行手册](docs/product/competition-demo-runbook.md)：通用产品叙事、演示路径、环境门禁和故障回退。
- [已知限制与延期项](docs/operations/known-issues.md)：AI 任务、邮件、限流和赛后能力的真实边界。
- [延期功能评估](docs/product/deferred-work.md)：已提前落地的增强、继续延期的边界和 v0.2 建议顺序。
- [API 协作说明](docs/api/README.md)：Apifox、Swagger 与契约变更流程。
- [AI 智能体集成设计](docs/architecture/ai-agent-integration.md)：组织运营智能体的能力边界、架构、权限、工具与分阶段落地方案。
- [AI 活动策划评测基线](docs/product/ai-activity-evaluation.md)：10 组校园场景、Prompt Injection、引用与真实模型质量门禁。
- Portal API 前缀：`/api/v1/portal/organizations/{organization_slug}`，无认证、仅返回公开已发布数据。
- Admin API 前缀：`/api/v1/admin`，要求 Bearer JWT 与服务端 RBAC 授权。

接口或字段变更必须按以下顺序进行：更新 OpenAPI → 更新文档与示例 → 更新后端 DTO/鉴权/测试 → 更新前端 API client 与页面。禁止在前端猜测尚未定义的 URL 或字段。

仓库内置统一质量门禁，覆盖 OpenAPI 结构与安全语义、94 条 Gin 路由、169 个 Schema、前端请求契约、26 个 Apifox 核心请求、Go 测试、前端类型检查和生产构建：

```powershell
.\scripts\run-quality-gate.ps1
```

Compose 已启动时，可同时执行 Web/API 路由冒烟和 S1—S6 真实 MySQL/Redis 集成套件：

```powershell
.\scripts\run-quality-gate.ps1 -Integration
```

比赛环境可以额外连续执行三轮真实 API 活动策划演练。脚本读取被 Git 忽略的 Compose 环境，只保存延迟、模型、Token、引用/操作计数和失败类别，不保存生成正文，也不会伪造人工评分或批准：

```powershell
.\scripts\run-ai-activity-demo-rehearsal.ps1 -Rounds 3
```

双组织环境可分别切换到 QUTCraft 与 Campus Commons 各跑一轮；报告按组织拆分，仍不会自动评分、批准或发布：

```powershell
.\scripts\run-multi-organization-demo-rehearsal.ps1 -RoundsPerOrganization 1
```

Apifox 集合、环境模板、各检查器的单独运行方式见 [API 协作说明](docs/api/README.md)。`apps/web` 另提供 `pnpm test:e2e`，使用 Playwright 在桌面和移动视口验证 Portal、登录、组织设置、Markdown 编辑器与 AI 活动策划工作台。GitHub Actions 会在 push 与 pull request 上运行基础门禁及 Chromium 关键流程。

### S1 内容闭环集成测试

启动 Compose 后，可在 Windows PowerShell 中运行：

```powershell
.\scripts\run-s1-integration.ps1
```

脚本读取本机 `deploy/compose/.env`，不会输出管理员密码，并以 `integration` 构建标签运行真实 API、MySQL 与 Redis 测试。测试连续执行三轮“草稿不可见 → 发布可见 → 重复发布冲突 → 下线不可见”，同时确认 Portal 列表与详情缓存被及时失效；资源用例覆盖草稿、跨组织、已发布和已下线四种下载边界。临时数据库记录和测试文件会在用例结束时清理。

### S2 成员与项目协作集成测试

启动 Compose 后，可在 Windows PowerShell 中运行：

```powershell
.\scripts\run-s2-integration.ps1
```

该套件执行“创建邀请并预创建待激活账户 → 公开预览 → 携带 token 设置密码并激活原账户 → 权限调整/停用/恢复 → 分配项目 → 完成里程碑”的真实 API 流程，并覆盖撤销清理、重复邀请、邮箱不匹配、token 重用、角色降级即时生效、旧 Access/Refresh 失效、重新登录恢复、Owner 保护、成员角色幂等更新和 RFC3339 日期校验。默认邮件驱动关闭时，测试会确认响应与数据库均明确记录 `disabled/0 attempts`，同时确认邀请只持久化 token 哈希。

### S3 申请审批集成测试

启动 Compose 后运行：

```powershell
.\scripts\run-s3-integration.ps1
```

该套件验证申请提交、条件筛选、审批事务、必填拒绝原因、重复审批冲突、审计记录和通知 Outbox，并确认已取消的服务器控制接口持续返回 404。测试数据结束后自动清理。

### S5 可观测性集成测试

启动 Compose 后运行：

```powershell
.\scripts\run-s5-observability-integration.ps1
```

该套件验证 MySQL/Redis readiness、合法 Request ID 传播、非法 ID 替换、审计查询鉴权与日期校验，并使用同一 Request ID 创建跨组织记录，确认 Admin 只能读取当前组织事件。临时审计与组织记录会在结束时清理。

## 项目结构

```text
apps/
├── api/                         # Go API 服务
│   ├── cmd/server/               # 服务启动入口
│   ├── internal/                 # 领域、服务、仓储与基础设施
│   └── migrations/               # 数据库迁移
└── web/                          # Vue 前端
    └── src/
        ├── api/                  # 类型化 Portal / Admin API client 与 Mock
        ├── components/           # 通用展示组件
        ├── layouts/              # Portal / Admin 页面外壳
        ├── router/               # 前端路由
        ├── styles/               # MD3 Token 与全局样式
        └── views/                # 公开页面与后台页面
deploy/
├── compose/                      # Docker Compose 开发环境
└── openresty/                    # 网关配置（规划）
docs/
├── adr/                          # 架构决策记录
├── api/                          # OpenAPI 与 API 文档
├── architecture/                 # 平台规范与架构说明
└── product/                      # 产品与视觉演示资料
scripts/                          # 工程脚本
tests/integration/                # 集成测试
```

## 文档导航

| 文档 | 用途 |
| --- | --- |
| [功能地图 v2](docs/product/feature-map-v2.md) | 截至 2026 年 8 月 5 日的真实完成度、比赛版收口、依赖和 v0.2+ 扩展路线。 |
| [项目排期 v2](schedule.md) | 从 2026 年 7 月 25 日重排至 8 月 31 日的业务切片、阶段门与延期切线。 |
| [需求范围 v1](docs/product/requirements-v1.md) | MVP 边界、角色、用户故事、优先级与非功能要求。 |
| [MVP 验收清单](docs/product/mvp-acceptance.md) | 截至 8 月 5 日同步的比赛版可执行验收用例。 |
| [比赛叙事一页纸](docs/product/competition-narrative.md) | 产品价值、技术亮点与演示路径。 |
| [平台统一规范](docs/architecture/platform-standard.md) | 前后端、接口、权限、门户扩展和质量规范。 |
| [RBAC 权限矩阵](docs/architecture/rbac-matrix.md) | 角色、权限名称、范围限制和后台路由建议。 |
| [信息架构](docs/product/information-architecture.md) | Portal/Admin 路由、状态字典与页面组件边界。 |
| [Portal Manifest v1](docs/product/portal-manifest-v1.md) | 自定义门户注册、主题 Token、能力边界与回退规则。 |
| [自定义门户包指南](docs/product/custom-portal-package.md) | 静态包结构、入口标记、Portal API、安全策略、发布与恢复流程。 |
| [完整 API 文档](docs/api/API.md) | 当前 API 的可读说明与安全约束。 |
| [申请审批 API 规范](docs/api/application-review.md) | 审批事务、通知、错误码和审计的详细规范。 |
| [媒体存储适配规范](docs/api/storage-adapter.md) | 本地卷与 MinIO/S3 驱动、配置安全、对象迁移和真实集成测试。 |
| [API 可观测性与审计规范](docs/api/observability.md) | Request ID、结构化访问日志、存活/就绪探针与组织隔离审计查询。 |
| [Compose 备份与恢复手册](docs/operations/backup-restore.md) | MySQL/本地媒体备份、校验清单、隔离恢复演练与 S3 边界。 |
| [项目部署/维护/开发者手册](PROJECT_GUIDE.md) | 从 Compose 部署、环境变量、迁移、备份、故障排查到开发和发布门禁的统一操作说明。 |
| [OpenAPI 契约](docs/api/openapi.yaml) | 可导入 Apifox / Swagger 的事实来源。 |
| [AI 智能体集成设计](docs/architecture/ai-agent-integration.md) | 比赛版组织运营智能体的架构、安全边界与分阶段实现状态。 |
| [组织运营智能体 API 规范](docs/api/ai-agent.md) | 已实现内容协作、活动策划、持久化运行队列、人工批准与质量评估接口。 |
| [MD3 门户演示](docs/product/style_demo.html) | 默认门户视觉演示。 |

## 开发约定

- 所有跨端接口改动先修改 OpenAPI；API client 不使用 `any` 绕过字段变更。
- 公开页面只使用 Portal API。后台页面可使用 Admin API，但服务端仍是权限控制的唯一可信边界。
- 所有列表具备加载、空状态、失败重试和分页语义；所有写操作都应具备明确的成功、失败与冲突处理。
- 资源下载链接由后端签发，前端不得拼接 MinIO / S3 URL 或泄露存储凭据。
- 提交前至少运行与改动相对应的类型检查、构建、契约校验或测试。

## 路线图

比赛版本的目标完成时间为 **2026 年 8 月 31 日**。当前按四条已贯通的闭环收口：内容发布到门户、加入申请到人工审批与通知、自定义门户加载与 MD3 回退，以及“组织知识 → AI 活动方案 → 人工批准 → 项目/里程碑/公告草稿”。剩余工作以部署复现、真实模型稳定性、视觉/故障回归和交付冻结为主，不再扩张成通用 OA、游戏服务器控制或通用智能体工作流平台。

实时模块状态、未完成错位和中长期扩展见 [功能地图 v2](docs/product/feature-map-v2.md)；日期、阶段门和延期规则见 [项目排期 v2](schedule.md)。

## 贡献与安全

欢迎通过 Issue 和 Pull Request 参与。涉及权限、资源上传、数据库迁移、外部服务集成与部署配置的改动应至少经过一名成员复核，并在 PR 中说明接口影响、权限影响、测试方式和回滚方案。

请勿提交 `.env.local`、访问 token、数据库密码、MinIO/S3 密钥、生产日志或真实成员隐私数据。如发现安全问题，请勿在公开 Issue 中披露可利用细节，应先联系项目维护者。

## 许可

本仓库当前使用 `Proprietary` 占位许可。开源范围、第三方贡献与再分发规则将在项目对外发布前由维护团队明确。
