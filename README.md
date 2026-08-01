# QUTCraft Platform

QUTCraft Platform 是一个面向校园社团与民间组织的可扩展内容与组织协作平台。项目以“公开门户与内部管理相分离”为基础：访客通过门户了解组织、项目、资源与知识内容；组织成员通过独立后台完成内容生产、成员协作、申请审核及可选的服务器适配操作。

青岛理工大学 QUTCraft Minecraft 社团是本项目的首个真实落地场景。它验证了平台既可以保持通用的组织数字化能力，也可以通过公开 API 构建具有社团特色的门户，而不会让 Minecraft 服务器能力污染通用业务核心。

> 当前仓库处于 MVP 开发阶段。前端支持契约 Mock 与远程 API 两种模式；Go API、MySQL 基础迁移、JWT/RBAC、Compose 开发环境以及内容、资源、项目、成员邀请和申请审核的基础业务端点已经落地，真实服务器适配与全链路收口仍按项目排期推进。

## 核心原则

- **门户与后台分离**：公开门户只展示已发布的公开信息；后台不作为门户页面的一部分。
- **API-first**：Portal 与 Admin 接口均以 OpenAPI 3.1 契约为先，前端类型与接口文档同步演进。
- **安全边界明确**：成员邮箱、草稿、审核、RCON 命令和服务端凭据只允许在受控管理 API 中处理。
- **可替换门户**：默认提供 MD3 门户；组织可以基于 Portal API 开发自己的公开门户主题。
- **安全回退**：自定义入口需通过同源、类型和门户 ID 标记探测；超时或资源错误自动保留 MD3，`?portal=md3` 可强制恢复。
- **可验证的适配**：QUTCraft 服务器状态与白名单审批属于可选服务器适配器能力；没有真实服务器时使用 Mock 保持演示可复现。

## 当前能力

### 公开门户

- MD3 风格的组织首页、公开项目、资源中心与知识库列表。
- 组织动态、项目、资源、知识文章及公开服务器状态的契约 Mock 数据。
- 响应式导航和独立的公开页面布局。

### 管理工作台

- 独立的后台侧边导航与工作台布局，入口为 `/admin`。
- Dashboard：组织概览、待办申请、近期内容与服务器适配状态。
- 内容工作区：查看内容状态并创建草稿。
- 成员与权限：查看成员、组织角色和状态，创建邀请，并显示可选 SMTP 的真实投递结果与失败重试。
- 审核与服务器：处理白名单/成员申请，演示受限 RCON 命令入口。
- 审计记录：按动作、对象、结果、Request ID 与日期查询当前组织的管理操作。
- 组织公开资料：维护名称、简称、标语、介绍、公开邮箱、社交链接和公开状态，修改立即作用于门户并留存审计。
- 门户 Manifest 设置：真实读取、草稿保存、JSON 导入、同源预览、独立启用与 Portal/Admin 安全边界提示。

> 前端仍保留契约 Mock 供离线演示；Compose 默认使用 remote 模式连接真实 API/MySQL。服务器操作使用明确标识的 Mock ServerAdapter，不会连接真实 Minecraft RCON；邀请邮件使用可选 SMTP Adapter，默认关闭且不会伪报发送成功。

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
| `/admin/users` | 成员与权限 |
| `/admin/reviews` | 申请审核与服务器适配 |
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
# mock：使用仓库内契约 fixture；remote：请求后端 API
VITE_API_MODE=mock
VITE_API_BASE_URL=http://localhost:8080
VITE_ORGANIZATION_SLUG=qutcraft
```

只有后端实现并可用后才应切换为 `VITE_API_MODE=remote`。不要将生产 token、数据库连接、RCON 凭据或对象存储密钥写入任何 `VITE_*` 变量；这些变量会被打包到浏览器端。

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
# 先在 .env 中设置 STORAGE_DRIVER=s3，并让 S3_ACCESS_KEY/S3_SECRET_KEY
# 与 MINIO_ROOT_USER/MINIO_ROOT_PASSWORD 对应。
docker compose --profile storage up --build
```

此时 API 会真实将新上传媒体写入 MinIO，而不是只启动一个未被使用的容器。默认 `STORAGE_DRIVER=local` 继续使用受控媒体卷。两种后端都只通过 API 分发文件，浏览器不会获得对象存储凭据；完整变量、迁移限制和验证脚本见 [媒体存储适配规范](docs/api/storage-adapter.md)。

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

数据库迁移、默认组织、RBAC 和可选的 `BOOTSTRAP_ADMIN_*` 所有者属于基础引导。内容、知识目录、项目、里程碑、申请和 Mock 同步记录属于演示数据，默认不会自动创建。需要本地或比赛演示数据时，在 `deploy/compose/.env` 中显式设置：

```dotenv
DEMO_SEED_ENABLED=true
DEMO_SEED_PROFILE=qutcraft
```

`DEMO_SEED_PROFILE=qutcraft` 提供 QUTCraft 社团演示资料；比赛或通用产品演示使用 `generic`，并建议同时将 `DEFAULT_ORGANIZATION_SLUG` 改为通用标识。Profile 只影响首次创建的组织和缺失的固定演示记录，不会覆盖后台已经编辑的资料。

然后重新创建 API 容器：

```bash
cd deploy/compose
docker compose up -d --build api
```

演示 seed 使用固定 ID，只补充缺失记录；重复启动不会生成重复数据，也不会覆盖已有记录的人工修改。当前会创建 4 条内容、3 个知识目录、3 个项目及 2 个里程碑，以及待处理、已通过、已拒绝三种申请和一条明确标识为 Mock 的服务器同步结果。

生产环境必须保持 `DEMO_SEED_ENABLED=false`，且必须替换引导密码与 JWT 密钥。关闭 seed 不会删除已有数据；如需清理演示记录，应使用受控清理脚本或重建专用演示数据库，不能在生产库直接执行通配删除。

## API 与接口协作

[docs/api/openapi.yaml](docs/api/openapi.yaml) 是 Portal 与 Admin API 的唯一机器可读契约源，支持直接导入 Apifox 或 Swagger / Redoc 工具。

- [完整 API 文档](docs/api/API.md)：认证、RBAC、响应封装、字段说明、错误语义、示例与安全边界。
- [申请审批与 ServerAdapter API 规范](docs/api/server-adapter.md)：审批事务、外部同步状态、失败重试、错误码与审计约束。
- [邀请邮件适配器规范](docs/api/email-adapter.md)：SMTP 服务端配置、投递状态、token 轮换重试与凭据安全边界。
- [API 可观测性与审计规范](docs/api/observability.md)：Request ID、结构化日志、存活/就绪探针和审计查询边界。
- [数据库迁移与回退规范](docs/operations/database-migrations.md)：版本账本、旧数据卷基线、升级演练和备份回退边界。
- [比赛演示运行手册](docs/product/competition-demo-runbook.md)：通用产品叙事、演示路径、环境门禁和故障回退。
- [已知限制与延期项](docs/operations/known-issues.md)：RCON、AI 任务、邮件、限流和赛后能力的真实边界。
- [API 协作说明](docs/api/README.md)：Apifox、Swagger 与契约变更流程。
- [AI 智能体集成设计](docs/architecture/ai-agent-integration.md)：组织运营智能体的能力边界、架构、权限、工具与分阶段落地方案。
- Portal API 前缀：`/api/v1/portal/organizations/{organization_slug}`，无认证、仅返回公开已发布数据。
- Admin API 前缀：`/api/v1/admin`，要求 Bearer JWT 与服务端 RBAC 授权。

接口或字段变更必须按以下顺序进行：更新 OpenAPI → 更新文档与示例 → 更新后端 DTO/鉴权/测试 → 更新前端 API client 与页面。禁止在前端猜测尚未定义的 URL 或字段。

仓库内置统一质量门禁，覆盖 OpenAPI 结构与安全语义、72 条 Gin 路由、65 个前端请求、19 个 Apifox 核心请求、Go 测试、前端类型检查和生产构建：

```powershell
.\scripts\run-quality-gate.ps1
```

Compose 已启动时，可同时执行 Web/API 路由冒烟和 S1—S6 真实 MySQL/Redis 集成套件：

```powershell
.\scripts\run-quality-gate.ps1 -Integration
```

Apifox 集合、环境模板、各检查器的单独运行方式见 [API 协作说明](docs/api/README.md)。`apps/web` 另提供 `pnpm test:e2e`，使用 Playwright 在桌面和移动视口验证 Portal、登录、组织设置与 Markdown 编辑器。GitHub Actions 会在 push 与 pull request 上运行基础门禁及 Chromium 关键流程。

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

该套件执行“创建邀请 → 公开预览 → 携带 token 注册 → 权限调整/停用/恢复 → 分配项目 → 完成里程碑”的真实 API 流程，并覆盖重复邀请、邮箱不匹配、token 重用、角色降级即时生效、旧 Access/Refresh 失效、重新登录恢复、Owner 保护、成员角色幂等更新和 RFC3339 日期校验。默认邮件驱动关闭时，测试会确认响应与数据库均明确记录 `disabled/0 attempts`，同时确认邀请只持久化 token 哈希，并在结束前清理、复查临时账户、成员关系、投递记录、审计、邀请和里程碑。

### S3 申请审批与服务器适配集成测试

启动 Compose 后运行：

```powershell
.\scripts\run-s3-integration.ps1
```

该套件验证申请提交、审批事务、重复审批冲突、Mock 白名单同步、失败重试和受限命令，并注入故障适配器确认“审批决定”不会因外部同步失败而回滚。Mock 响应会明确返回 `mode: mock` 与 `executed: false`；同步记录独立保存状态、尝试次数和脱敏错误，测试数据结束后自动清理。适配调用超时由 `SERVER_ADAPTER_TIMEOUT` 控制，默认 5 秒。

### S5 可观测性集成测试

启动 Compose 后运行：

```powershell
.\scripts\run-s5-observability-integration.ps1
```

该套件验证 MySQL/Redis readiness、合法 Request ID 传播、非法 ID 替换、审计查询鉴权与日期校验，并使用同一 Request ID 创建跨组织记录，确认 Admin 只能读取当前组织事件。临时审计与组织记录会在结束时清理。

## 项目结构

```text
apps/
├── api/                         # Go API 服务（规划骨架）
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
| [功能地图 v2](docs/product/feature-map-v2.md) | 截至 2026 年 8 月 1 日的真实完成度、MVP 收口、依赖和 v0.2+ 扩展路线。 |
| [项目排期 v2](schedule.md) | 从 2026 年 7 月 25 日重排至 8 月 31 日的业务切片、阶段门与延期切线。 |
| [需求范围 v1](docs/product/requirements-v1.md) | MVP 边界、角色、用户故事、优先级与非功能要求。 |
| [MVP 验收清单](docs/product/mvp-acceptance.md) | 截至 7 月 17 日的冻结检查点及后续可执行验收用例。 |
| [比赛叙事一页纸](docs/product/competition-narrative.md) | 产品价值、技术亮点与演示路径。 |
| [平台统一规范](docs/architecture/platform-standard.md) | 前后端、接口、权限、门户扩展和质量规范。 |
| [RBAC 权限矩阵](docs/architecture/rbac-matrix.md) | 角色、权限名称、范围限制和后台路由建议。 |
| [信息架构](docs/product/information-architecture.md) | Portal/Admin 路由、状态字典与页面组件边界。 |
| [Portal Manifest v1](docs/product/portal-manifest-v1.md) | 自定义门户注册、主题 Token、能力边界与回退规则。 |
| [自定义门户包指南](docs/product/custom-portal-package.md) | 静态包结构、入口标记、Portal API、安全策略、发布与恢复流程。 |
| [完整 API 文档](docs/api/API.md) | 当前 API 的可读说明与安全约束。 |
| [申请审批与 ServerAdapter API 规范](docs/api/server-adapter.md) | 审批、服务器同步、失败重试、错误码和审计的详细规范。 |
| [媒体存储适配规范](docs/api/storage-adapter.md) | 本地卷与 MinIO/S3 驱动、配置安全、对象迁移和真实集成测试。 |
| [API 可观测性与审计规范](docs/api/observability.md) | Request ID、结构化访问日志、存活/就绪探针与组织隔离审计查询。 |
| [Compose 备份与恢复手册](docs/operations/backup-restore.md) | MySQL/本地媒体备份、校验清单、隔离恢复演练与 S3 边界。 |
| [OpenAPI 契约](docs/api/openapi.yaml) | 可导入 Apifox / Swagger 的事实来源。 |
| [AI 智能体集成设计](docs/architecture/ai-agent-integration.md) | 比赛版组织运营智能体的架构、安全边界与分阶段实现状态。 |
| [组织运营智能体 API 规范](docs/api/ai-agent.md) | 已实现 AI-0 接口、权限、状态机、模型配置、错误码、审计与验证。 |
| [MD3 门户演示](docs/product/style_demo.html) | 默认门户视觉演示。 |

## 开发约定

- 所有跨端接口改动先修改 OpenAPI；API client 不使用 `any` 绕过字段变更。
- 公开页面只使用 Portal API。后台页面可使用 Admin API，但服务端仍是权限控制的唯一可信边界。
- 所有列表具备加载、空状态、失败重试和分页语义；所有写操作都应具备明确的成功、失败与冲突处理。
- 资源下载链接由后端签发，前端不得拼接 MinIO / S3 URL 或泄露存储凭据。
- RCON 命令只能由服务端白名单执行，并记录操作者、命令摘要、结果与 `request_id`。
- 提交前至少运行与改动相对应的类型检查、构建、契约校验或测试。

## 路线图

比赛版本的目标完成时间为 **2026 年 8 月 31 日**。当前先按三个可验收闭环收口：内容发布到门户、申请审批到服务器适配器、自定义门户加载与 MD3 回退；核心功能和质量阶段通过后，再接入比赛要求的 AI 内容协作闭环，最后完成交付冻结。

实时模块状态、未完成错位和中长期扩展见 [功能地图 v2](docs/product/feature-map-v2.md)；日期、阶段门和延期规则见 [项目排期 v2](schedule.md)。

## 贡献与安全

欢迎通过 Issue 和 Pull Request 参与。涉及权限、资源上传、数据库迁移、服务器适配与部署配置的改动应至少经过一名成员复核，并在 PR 中说明接口影响、权限影响、测试方式和回滚方案。

请勿提交 `.env.local`、访问 token、数据库密码、RCON 密码、MinIO/S3 密钥、生产日志或真实成员隐私数据。如发现安全问题，请勿在公开 Issue 中披露可利用细节，应先联系项目维护者。

## 许可

本仓库当前使用 `Proprietary` 占位许可。开源范围、第三方贡献与再分发规则将在项目对外发布前由维护团队明确。
