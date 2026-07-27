# QUTCraft Platform

QUTCraft Platform 是一个面向校园社团与民间组织的可扩展内容与组织协作平台。项目以“公开门户与内部管理相分离”为基础：访客通过门户了解组织、项目、资源与知识内容；组织成员通过独立后台完成内容生产、成员协作、申请审核及可选的服务器适配操作。

青岛理工大学 QUTCraft Minecraft 社团是本项目的首个真实落地场景。它验证了平台既可以保持通用的组织数字化能力，也可以通过公开 API 构建具有社团特色的门户，而不会让 Minecraft 服务器能力污染通用业务核心。

> 当前仓库处于 MVP 开发阶段。前端支持契约 Mock 与远程 API 两种模式；Go API、MySQL 基础迁移、JWT/RBAC、Compose 开发环境以及内容、资源、项目和申请审核的基础业务端点已经落地，真实服务器适配与全链路收口仍按项目排期推进。

## 核心原则

- **门户与后台分离**：公开门户只展示已发布的公开信息；后台不作为门户页面的一部分。
- **API-first**：Portal 与 Admin 接口均以 OpenAPI 3.1 契约为先，前端类型与接口文档同步演进。
- **安全边界明确**：成员邮箱、草稿、审核、RCON 命令和服务端凭据只允许在受控管理 API 中处理。
- **可替换门户**：默认提供 MD3 门户；组织可以基于 Portal API 开发自己的公开门户主题。
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
- 成员与权限：查看成员、组织角色和状态。
- 审核与服务器：处理白名单/成员申请，演示受限 RCON 命令入口。
- 组织设置页面与 Portal/Admin 安全边界提示。

> 后台当前为前端 Mock 实现。页面操作会更新浏览器运行时 Mock 数据，但不会连接 Minecraft 服务器或写入真实数据库。

### 身份与工程底座

- Go + Gin + GORM API 服务，包含健康检查与统一 JSON 响应。
- MySQL 初始迁移：组织、用户、角色、权限、成员关系、刷新令牌与审计事件。
- 注册、登录、刷新令牌轮换、退出撤销与当前会话接口。
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
| `/admin/settings` | 组织设置 |

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

API 默认监听 `http://localhost:8080`，健康检查地址为 `GET /healthz`。首次启动会创建基础表、默认组织、角色与权限；开发环境可使用 `.env` 中的 `BOOTSTRAP_ADMIN_*` 创建第一个所有者。生产环境必须替换 JWT 密钥和引导密码，且不得提交 `.env`。

### 启动完整开发环境（Docker Compose）

```bash
cd deploy/compose
copy .env.example .env
docker compose up --build
```

默认启动 MySQL、Redis、API 与 Web；如果使用仓库当前 `deploy/compose/.env`，Web/API 地址分别为 `http://localhost:8082` 和 `http://localhost:18080`。

需要对象存储时使用：

```bash
docker compose --profile storage up --build
```

需要本地 Swagger UI 时使用：

```bash
docker compose --profile docs up
```

随后访问 `http://localhost:8081`。Swagger UI 加载仓库中的 `docs/api/openapi.yaml`，因此它与 Apifox 的事实来源一致；本地联调时在 Swagger/Apifox 中将 API Server 覆盖为 `http://localhost:8080`。

当前电脑若未安装 Go 或 Docker，可继续使用前端 Mock 模式；安装工具链后按以上命令验证 API 和 Compose。

## API 与接口协作

[docs/api/openapi.yaml](docs/api/openapi.yaml) 是 Portal 与 Admin API 的唯一机器可读契约源，支持直接导入 Apifox 或 Swagger / Redoc 工具。

- [完整 API 文档](docs/api/API.md)：认证、RBAC、响应封装、字段说明、错误语义、示例与安全边界。
- [API 协作说明](docs/api/README.md)：Apifox、Swagger 与契约变更流程。
- Portal API 前缀：`/api/v1/portal/organizations/{organization_slug}`，无认证、仅返回公开已发布数据。
- Admin API 前缀：`/api/v1/admin`，要求 Bearer JWT 与服务端 RBAC 授权。

接口或字段变更必须按以下顺序进行：更新 OpenAPI → 更新文档与示例 → 更新后端 DTO/鉴权/测试 → 更新前端 API client 与页面。禁止在前端猜测尚未定义的 URL 或字段。

可使用以下命令校验 OpenAPI：

```bash
pnpm --package=@redocly/cli@1.34.6 dlx redocly lint docs/api/openapi.yaml
```

同时检查 Go 的 Gin 路由是否与 OpenAPI 方法、路径和路径参数一致：

```bash
python scripts/check-openapi-routes.py
```

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
| [功能地图 v2](docs/product/feature-map-v2.md) | 截至 2026 年 7 月 25 日的真实完成度、MVP 收口、依赖和 v0.2+ 扩展路线。 |
| [项目排期 v2](schedule.md) | 从 2026 年 7 月 25 日重排至 8 月 31 日的业务切片、阶段门与延期切线。 |
| [需求范围 v1](docs/product/requirements-v1.md) | MVP 边界、角色、用户故事、优先级与非功能要求。 |
| [MVP 验收清单](docs/product/mvp-acceptance.md) | 截至 7 月 17 日的冻结检查点及后续可执行验收用例。 |
| [比赛叙事一页纸](docs/product/competition-narrative.md) | 产品价值、技术亮点与演示路径。 |
| [平台统一规范](docs/architecture/platform-standard.md) | 前后端、接口、权限、门户扩展和质量规范。 |
| [RBAC 权限矩阵](docs/architecture/rbac-matrix.md) | 角色、权限名称、范围限制和后台路由建议。 |
| [信息架构](docs/product/information-architecture.md) | Portal/Admin 路由、状态字典与页面组件边界。 |
| [Portal Manifest v1](docs/product/portal-manifest-v1.md) | 自定义门户注册、主题 Token、能力边界与回退规则。 |
| [完整 API 文档](docs/api/API.md) | 当前 API 的可读说明与安全约束。 |
| [OpenAPI 契约](docs/api/openapi.yaml) | 可导入 Apifox / Swagger 的事实来源。 |
| [MD3 门户演示](docs/product/style_demo.html) | 默认门户视觉演示。 |
| [Minecraft 门户演示](docs/product/style_mc.html) | QUTCraft 第一方主题探索。 |

## 开发约定

- 所有跨端接口改动先修改 OpenAPI；API client 不使用 `any` 绕过字段变更。
- 公开页面只使用 Portal API。后台页面可使用 Admin API，但服务端仍是权限控制的唯一可信边界。
- 所有列表具备加载、空状态、失败重试和分页语义；所有写操作都应具备明确的成功、失败与冲突处理。
- 资源下载链接由后端签发，前端不得拼接 MinIO / S3 URL 或泄露存储凭据。
- RCON 命令只能由服务端白名单执行，并记录操作者、命令摘要、结果与 `request_id`。
- 提交前至少运行与改动相对应的类型检查、构建、契约校验或测试。

## 路线图

`v0.1.0-mvp` 的目标完成时间为 **2026 年 8 月 31 日**。当前开发按三个可验收闭环收口：内容发布到门户、申请审批到服务器适配器、自定义门户加载与 MD3 回退。随后依次进入成员/项目协作、质量加固和比赛交付冻结。

实时模块状态、未完成错位和中长期扩展见 [功能地图 v2](docs/product/feature-map-v2.md)；日期、阶段门和延期规则见 [项目排期 v2](schedule.md)。

## 贡献与安全

欢迎通过 Issue 和 Pull Request 参与。涉及权限、资源上传、数据库迁移、服务器适配与部署配置的改动应至少经过一名成员复核，并在 PR 中说明接口影响、权限影响、测试方式和回滚方案。

请勿提交 `.env.local`、访问 token、数据库密码、RCON 密码、MinIO/S3 密钥、生产日志或真实成员隐私数据。如发现安全问题，请勿在公开 Issue 中披露可利用细节，应先联系项目维护者。

## 许可

本仓库当前使用 `Proprietary` 占位许可。开源范围、第三方贡献与再分发规则将在项目对外发布前由维护团队明确。
