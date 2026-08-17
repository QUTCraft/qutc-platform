# QUTCraft Platform 项目部署、维护与开发者手册

> 适用基线：v0.1.0-competition
> 更新日期：2026-08-05
> 适用范围：单机 Docker Compose、本地 Go/前端开发、比赛演示和后续维护

这份手册是项目的操作入口。README 负责产品介绍；本文件负责部署、维护、开发和故障排查。API 字段与错误语义以 OpenAPI 契约为准，领域细节见 docs/ 下的专题文档。

## 1. 系统边界

```text
浏览器
  ├─ Web：Vue 3 构建产物，由 Nginx 提供静态文件
  └─ API：Go + Gin，负责认证、RBAC、Portal、Admin 和异步 Worker
       ├─ MySQL：业务数据、迁移账本、AgentRun 队列、审计和通知 Outbox
       ├─ Redis：公开数据缓存和就绪检查
       ├─ Local Volume / MinIO：媒体文件
       ├─ SMTP（可选）：邀请邮件和审批结果通知
       └─ OpenAI-compatible Provider（可选）：AI 活动策划与内容协作
```

必须保持的边界：

- 浏览器不直接访问 MySQL、Redis、SMTP 或对象存储凭据。
- Portal 只读取当前组织已发布的公开数据；草稿、成员隐私、申请材料和审计不会进入 Portal。
- Admin 的按钮隐藏不是权限控制；Go 服务端必须执行组织范围、RBAC、状态机和审计校验。
- AI 当前是 API 进程内的单机 Worker：queued 写入 MySQL 后领取；重启保留未领取任务，将中断的 running 标记为 failed，不承诺多实例调度。
- 加入申请只更新平台内审批状态；代码库不包含游戏服务器命令或自动同步能力。

### 1.1 服务和端口

端口由 deploy/compose/.env 决定，以下是仓库示例默认值：

| 服务 | 容器内端口 | 外部默认端口 | 用途 |
| --- | ---: | ---: | --- |
| Web | 80 | 8082 | 门户和 Admin 单页应用 |
| API | 8080 | 8080 | REST API、健康检查 |
| MySQL | 3306 | 3306 | 本地开发/集成测试；生产不应公网暴露 |
| Redis | 6379 | 6379 | 本地开发/集成测试；生产不应公网暴露 |
| Swagger UI | 8080 | 8081 | docs profile，可选 |
| MinIO API | 9000 | 9000 | storage profile，可选 |
| MinIO Console | 9001 | 9001 | storage profile，可选 |

生产和 Compose 默认由 Web 容器将同源 `/api` 代理到 API，浏览器不需要知道 API 宿主机端口。只有本地 Vite 与 API 分端口调试时才设置：

```dotenv
API_PORT=18080
VITE_API_BASE_URL=http://127.0.0.1:18080
CORS_ALLOWED_ORIGINS=http://localhost:8082,http://127.0.0.1:8082
```

生产 `VITE_API_BASE_URL` 应留空；这样域名和宿主机端口变化不需要重建 Web。

## 2. 部署

### 2.1 环境要求

- Docker Desktop（Windows）或 Docker Engine + Compose v2（Linux）。
- 前端开发：Node.js 20+、pnpm 9+。
- API/Go 测试：Go 1.22+。
- 契约脚本：Python 3.10+。
- Windows 建议使用 PowerShell 执行仓库 .ps1 脚本。

### 2.2 第一次启动

```powershell
Set-Location D:\qutc-platform
Copy-Item .\deploy\compose\.env.example .\deploy\compose\.env
notepad .\deploy\compose\.env
Set-Location .\deploy\compose
docker compose --env-file .env up -d --build
docker compose --env-file .env ps
```

开发/演示环境至少确认：

```dotenv
BOOTSTRAP_ADMIN_EMAIL=admin@qutcraft.local
BOOTSTRAP_ADMIN_PASSWORD=请替换为至少12位的开发密码
JWT_ACCESS_SECRET=请替换为随机长字符串
DEMO_SEED_ENABLED=true
DEMO_SEED_PROFILE=qutcraft
DEMO_SEED_MULTI_ORGANIZATION=true
AI_PROVIDER=mock
EMAIL_DRIVER=disabled
STORAGE_DRIVER=local
```

检查：

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
```

访问：

- Web：http://localhost:8082/
- Admin：http://localhost:8082/admin
- Swagger（启用后）：http://localhost:8081
- OpenAPI 事实来源：docs/api/openapi.yaml

若 .env 使用 API 18080，上述 API 地址改为 http://127.0.0.1:18080。

### 2.3 常用 Compose 操作

```powershell
Set-Location D:\qutc-platform\deploy\compose
docker compose --env-file .env ps
docker compose --env-file .env logs --tail=200 api
docker compose --env-file .env logs --tail=200 web
docker compose --env-file .env restart api
docker compose --env-file .env up -d --build api web
docker compose --env-file .env stop
docker compose --env-file .env down
```

上述顺序分别用于查看、日志、重启、重建、停止和删除容器/网络。不要在真实数据环境执行 docker compose down -v；它会删除 MySQL、Redis、MinIO 和媒体 Docker volume。

### 2.4 MinIO 与 Swagger profile

启用 MinIO 时，.env 至少设置：

```dotenv
STORAGE_DRIVER=s3
S3_ENDPOINT=minio:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=请替换为本地密钥
S3_BUCKET=qutcraft-media
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=请替换为本地密钥
```

```powershell
docker compose --env-file .env --profile storage up -d --build
docker compose --env-file .env --profile docs up -d swagger
```

容器内 API 使用 minio:9000；宿主机集成测试使用 localhost:9000。MinIO Console 默认在 http://localhost:9001。STORAGE_DRIVER=s3 时必须启动 storage profile。

如果 MinIO 已由 1Panel 或其他部署系统独立运行，不要在本项目里重复启动 storage profile。先确认 API 容器能够访问 MinIO 的 S3 API 端口，再由组织所有者进入 `/admin/settings` 的“服务接入”保存 S3/MinIO 配置并点击“保存并验证存储”；连接成功后进入 `/admin/assets`，可拖拽或多选上传。Endpoint 填写 API 容器实际可达的 `host:port`，不填写 `http://`、`https://`，也不把 Access Key/Secret Key 写入 `VITE_*` 或网页静态配置。

### 2.5 远程服务器

当前 Compose 是单机基线，不是高可用生产方案。远程部署至少需要：

1. 使用 HTTPS 反向代理，把域名转发到本机 Web/API 端口。
2. CORS_ALLOWED_ORIGINS 只填写真实前端 Origin，不能使用通配符。
3. 替换 JWT、数据库、Redis、MinIO、SMTP、AI 的所有默认值。
4. APP_ENV=production，关闭两个 DEMO_SEED 开关。
5. API 使用真实 Provider 或 disabled；生产环境禁止 AI_PROVIDER=mock。
6. 不把 MySQL、Redis、MinIO Console 端口暴露到公网。
7. 备份使用加密、最小权限和异地副本，并在迁移前验证恢复。
8. TLS、域名、反向代理、密钥系统和日志采集由部署环境负责。

## 3. 配置说明

| 文件 | 用途 | 是否提交 |
| --- | --- | --- |
| deploy/compose/.env | Compose 插值、API 容器、Web 构建参数 | 否，已忽略 |
| deploy/compose/.env.example | Compose 模板 | 是，不放真实凭据 |
| apps/api/.env | 本机 go run 的 API 环境 | 否 |
| apps/api/.env.example | 本机 API 模板 | 是 |
| apps/web/.env.local | Vite 本地环境 | 否 |
| apps/web/.env.example | Vite 模板 | 是 |

关键配置组：

| 配置组 | 变量 | 维护要点 |
| --- | --- | --- |
| 运行 | APP_ENV、HTTP_ADDR | production 会启用生产安全校验。 |
| 数据库 | MYSQL_DSN / MYSQL_* | API 本机使用 DSN；Compose 使用容器网络地址。 |
| Redis | REDIS_ADDR、REDIS_PASSWORD、REDIS_DB | readiness 和公开缓存依赖 Redis。 |
| JWT | JWT_ISSUER、JWT_ACCESS_SECRET、JWT_ACCESS_TTL、JWT_REFRESH_TTL | Access 短时有效，Refresh 为 HttpOnly Cookie 并轮换。 |
| 跨域/限流 | CORS_ALLOWED_ORIGINS、*_RATE_LIMIT | 当前限流按 API 实例内存计数。 |
| 引导/演示 | BOOTSTRAP_ADMIN_*、DEMO_SEED_* | 只用于首个所有者和演示资料；生产关闭 Seed。 |
| 存储默认值 | STORAGE_DRIVER、STORAGE_LOCAL_ROOT、S3_* | 首次启动默认值；日常在系统设置页面管理。 |
| 邮件默认值 | EMAIL_DRIVER、SMTP_* | 首次启动默认值；日常在系统设置页面管理。 |
| AI 默认值 | AI_PROVIDER、AI_BASE_URL、AI_API_KEY、AI_MODEL | 首次启动默认值；日常在智能体配置页面管理。 |

安全规则：

- 所有 VITE_* 变量都会进入浏览器构建产物，只能放公开地址和组织 slug，不能放任何密钥。
- AI、SMTP 和对象存储凭据可通过受保护的 Admin 页面提交，由 API 加密保存且不回传明文；数据库、JWT 与 Redis 密钥仍只由部署注入。
- PUBLIC_WEB_BASE_URL 用于生成邀请链接，必须是可访问的绝对 HTTP/HTTPS 地址。
- .env、备份、运行报告和临时文件不得提交；提交前运行 python scripts/scan-secrets.py。
- Bootstrap 配置只创建不存在的账户，不会替换已有账户密码。

## 4. 数据库、迁移与数据生命周期

当前迁移为 001—018。SQL 嵌入 API 二进制，按文件名顺序执行并写入 schema_migrations。001—008 兼容旧 AutoMigrate 数据卷；009 以后是显式迁移。014 提供通知、内容修订、下载统计、默认组织和 AI 引用正文基础；015 为既有内容补齐 version 1 修订基线；016 增加模型供应商配置；017 增加组织级加密服务接入配置；018 为媒体资产补齐外部图床提供方与 URL 字段。

新增表或字段时：

1. 新建下一个编号的 SQL，不修改已执行迁移。
2. 同步模型、服务、Handler、OpenAPI、前端类型、测试和文档。
3. 在全新数据库和已有数据卷各执行一次。
4. 检查 schema_migrations 连续，确认外键列字符集/排序规则一致。
5. 生产迁移前创建并验证备份；当前没有自动回滚，回退以备份恢复和新卷切换为主。

检查迁移账本：

```powershell
docker compose --env-file .env exec mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot "$MYSQL_DATABASE" -e "SELECT version, applied_at FROM schema_migrations ORDER BY version"'
```

详见 [数据库迁移与回退规范](docs/operations/database-migrations.md)。

演示数据规则：

- DEMO_SEED_ENABLED=true 才会创建内容、知识目录、项目、里程碑、申请和 Mock 同步记录。
- DEMO_SEED_MULTI_ORGANIZATION=true 会额外创建独立组织，用于验证租户隔离。
- Seed 只补充缺失记录，不覆盖人工修改；关闭 Seed 不会删除已有数据。
- 生产必须关闭两个 Seed 开关。清理演示数据应使用专用数据库或受控方案，不要在生产库执行通配 DELETE。

## 5. 日常维护

### 5.1 健康检查和日志

```powershell
Set-Location D:\qutc-platform\deploy\compose
docker compose --env-file .env ps
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
docker compose --env-file .env logs --tail=200 api
docker compose --env-file .env logs --tail=200 mysql redis
```

healthz 只证明 API 进程存活；readyz 还必须显示 MySQL 和 Redis 为 ok。排查时优先记录页面路径、HTTP 状态和 request_id，再用相同 ID 搜索 API 结构化日志。不要把 Authorization、Cookie、SMTP/AI 凭据或完整申请材料复制到 Issue。

### 5.2 备份和恢复验证

```powershell
Set-Location D:\qutc-platform
.\scripts\backup-compose.ps1
.\scripts\verify-backup-restore.ps1 -BackupPath .\backups\qutcraft-<UTC时间>
.\scripts\run-backup-restore-rehearsal.ps1
```

备份包含 MySQL、local 媒体卷、逐文件 SHA-256 和 manifest.json，恢复验证使用随机临时数据库/卷，不覆盖当前数据。Redis 是可重建缓存，不进入备份。STORAGE_DRIVER=s3 时，backup-compose.ps1 -SkipMedia 只备份数据库；媒体必须用 MinIO/S3 版本控制、复制或快照保护。完整灾备流程见 [Compose 备份与恢复手册](docs/operations/backup-restore.md)。

备份包含成员数据、Token 哈希、审计和媒体，必须放在受控、加密、权限隔离的目录中，不得提交 Git 或上传公开网盘。

### 5.3 邮件、通知和 AI Worker

- 组织未保存网页配置时沿用 EMAIL_DRIVER/STORAGE_DRIVER 等部署默认值；保存后按组织即时生效。
- 邮件关闭时邀请仍可复制链接完成加入，投递应显示 disabled/skipped，不能显示 sent。
- 审批事务先写唯一 notification_outboxes 事件，再由 API 内单机 Worker 发送；失败不回滚审批事实，管理员可以查看和重试。
- AI 运行先持久化为 queued；重启后 queued 继续领取，running 明确失败。当前不支持断点续跑、多实例公平调度或独立消息中间件。
- SMTP、AI、MinIO 故障属于外部依赖故障，先查适配器状态、超时和脱敏错误，再决定重试。

## 6. 故障排查速查

| 现象 | 检查 | 处理 |
| --- | --- | --- |
| 浏览器 Failed to fetch | Web 的 `/api` 代理、API 容器、/readyz | Compose 查看 web/api 日志；生产不应给浏览器写死 API 端口。 |
| 页面能打开但没有内容 | VITE_API_MODE、同源 `/api`、组织 slug | Compose/生产确认 remote 与同源代理；mock 模式只读前端 fixture。 |
| Admin 返回 401 | 登录、Cookie、Origin、API 地址 | Compose 远程模式使用 BOOTSTRAP_ADMIN_*；跨端口时检查 CORS 和凭据请求。 |
| Admin 返回 403 | 当前组织、成员状态、RBAC | 切换到正确组织；不要用前端改按钮绕过权限。 |
| /readyz 非 ready | MySQL/Redis 健康和网络 | 查看 docker compose ps 与依赖日志；API 容器内应使用 mysql:3306、redis:6379。 |
| API 启动失败并提示迁移 | 迁移日志、账本、旧数据卷 | 先备份，不删 volume；重点检查 014/015 外键 collation。 |
| 资源上传/下载失败 | 系统设置中的存储连接、volume 权限、MinIO profile | 页面先“保存并验证存储”；local 再查媒体卷，s3 再查 MinIO。 |
| AI 无法生成 | 智能体配置中的 Provider、Key、Base URL、模型、配额 | disabled 是关闭；mock 仅开发；保存后用页面功能验证。 |
| 邮件 disabled/failed | 系统设置中的 SMTP、连接验证、Outbox | 先保留审批事实/复制链接；页面修复 SMTP 后重试通知。 |
| 端口被占用 | .env、容器状态、宿主机进程 | 修改宿主机端口；同源代理下通常无需重建 Web。 |

## 7. 开发者工作流

### 7.1 安装依赖

```powershell
Set-Location D:\qutc-platform
git pull --ff-only origin main
Set-Location .\apps\web
pnpm install --frozen-lockfile
Set-Location ..\api
go mod download
```

开发前阅读 [贡献指南](CONTRIBUTING.md)、[安全策略](SECURITY.md) 和对应领域文档。不要在有未提交修改时执行覆盖配置或删除 volume 的操作。

### 7.2 前端

```powershell
Set-Location D:\qutc-platform\apps\web
Copy-Item .env.example .env.local
pnpm dev
```

本地 Vite 离线调试可显式设置 VITE_API_MODE=mock；生产构建会强制使用 remote。真实 Compose 联调设置：

```dotenv
VITE_API_MODE=remote
VITE_API_BASE_URL=http://127.0.0.1:8080
VITE_ORGANIZATION_SLUG=qutcraft
```

改动 VITE_* 后重启 Vite；改动 Compose 构建参数后重建 Web。前端请求集中写入 apps/web/src/api，组件不应散落 URL、Token 或组织判断。

### 7.3 API

本机运行 API 时，先启动 MySQL、Redis 和 media-init：

```powershell
Set-Location D:\qutc-platform\deploy\compose
docker compose --env-file .env up -d mysql redis media-init
Set-Location ..\..\apps\api
Copy-Item .env.example .env
go run .\cmd\server
```

常用命令：

```powershell
Set-Location D:\qutc-platform\apps\api
go test ./...
go test -race ./...
go vet ./...
go run .\cmd\server
```

### 7.4 API 变更顺序

项目采用 OpenAPI-first：

```text
OpenAPI → API 文档/Apifox → Go DTO/鉴权/测试 → TypeScript client/Mock → 页面 → 集成/e2e
```

新增接口或字段至少同步：

1. internal/model、服务层和状态机。
2. Handler 输入校验、统一响应、错误码、Request ID。
3. cmd/server/main.go 路由和服务端 RBAC。
4. OpenAPI Schema、路径、权限和示例。
5. 前端类型、client、Mock 和页面状态。
6. 单测、迁移、集成测试和专题文档。

提交前运行：

```powershell
Set-Location D:\qutc-platform
python scripts/lint-openapi.py
python scripts/check-openapi-routes.py
python scripts/check-web-api-contract.py
python scripts/check-apifox-collection.py
```

当前基线为 92 条路由、169 个 Schema、84 个前端请求和 26 个 Apifox 核心请求。

### 7.5 质量门禁和专项测试

```powershell
Set-Location D:\qutc-platform
.\scripts\run-quality-gate.ps1
.\scripts\run-quality-gate.ps1 -Integration
```

基础门禁包含 OpenAPI、Gin 路由、Web 请求、Apifox、密钥扫描、Go 测试、AI Mock 基线、前端类型检查和生产构建；-Integration 额外运行路由冒烟和 S1—S6。

| 脚本 | 用途 |
| --- | --- |
| scripts/run-route-smoke.ps1 | SPA 路由、CSP、Portal 404、liveness/readiness |
| scripts/run-s1-integration.ps1 | 内容、发布、缓存、资源下载 |
| scripts/run-s2-integration.ps1 | 邀请、注册、成员、组织切换、项目里程碑 |
| scripts/run-s3-integration.ps1 | 申请提交、筛选、审批事务、审计与已退役接口检查 |
| scripts/run-s4-integration.ps1 | 自定义 Portal、MD3 回退、安全头 |
| scripts/run-s5-observability-integration.ps1 | readiness、Request ID、审计、组织隔离 |
| scripts/run-s6-agent-integration.ps1 | AI 知识隔离、Prompt Injection、活动策划批准 |
| scripts/run-storage-integration.ps1 | MinIO/S3 上传下载往返 |
| scripts/run-ai-activity-evaluation.ps1 -Provider mock | 10 组活动场景 Mock 基线 |
| scripts/run-ai-activity-demo-rehearsal.ps1 -Rounds 3 | 真实 AI 演练，不自动评分/批准 |
| apps/web/pnpm test:e2e | Playwright 交互和移动视口回归 |

真实 AI 演练的 AI_BASE_URL、AI_API_KEY 和 AI_MODEL 只放在 Git 忽略的 Compose .env 或受控进程环境，不写入 VITE_*、代码和报告。

## 8. 代码与安全约定

- 查询必须带当前 organization_id；跨组织读取/写入必须有拒绝测试。
- 每个 Admin 写路由显式挂载 RequirePermission，服务层还要校验对象归属和状态转换。
- 发布、审批、项目批准、成员变更、通知重试、AI 配置/运行等操作必须写审计。
- Token 只存哈希或 HttpOnly Cookie；前端 Access Token 只驻留内存。
- Markdown/HTML、上传文件、AI 输出和知识资料均是不可信输入；预览需清洗，上传需限制类型/大小。
- 不记录密码、Token、SMTP/AI 凭据、完整申请材料或未脱敏模型结果。
- 外部服务失败不能伪造成功：审批、邮件 Outbox 和 AI 运行状态分别保存。
- 不为测试放宽 CORS、RBAC、命令白名单或生产 Mock 限制。

## 9. 发布前检查清单

- [ ] git diff 无无关修改、凭据、备份、运行报告或 .env。
- [ ] OpenAPI、Gin 路由、前端 client、Apifox 和文档一致。
- [ ] 新迁移在全新数据库和已有数据卷各验证一次。
- [ ] Go 测试、前端构建、密钥扫描和对应集成测试通过。
- [ ] 生产使用真实密钥、关闭 Demo Seed、禁用 Mock、限制 CORS。
- [ ] MySQL/Redis/MinIO Console 未暴露公网，Web/API 通过 HTTPS 访问。
- [ ] /healthz、/readyz 正常，日志可按 request_id 关联。
- [ ] 备份已创建并完成隔离恢复验证，记录提交版本和迁移版本。
- [ ] AI 方案有引用，人工批准只创建非公开项目/里程碑/公告草稿。
- [ ] 模型与 SMTP 故障均不会伪报成功。

## 10. 相关文档

- [README](README.md)：产品范围、页面和快速入口。
- [项目排期](schedule.md)：阶段门、完成项和延期边界。
- [完整 API 文档](docs/api/API.md)：认证、权限、字段和错误语义。
- [API 协作说明](docs/api/README.md)：OpenAPI、Swagger、Apifox 和契约流程。
- [数据库迁移规范](docs/operations/database-migrations.md)：001—018 迁移与回退边界。
- [备份与恢复手册](docs/operations/backup-restore.md)：备份格式、隔离恢复和 S3/MinIO 边界。
- [已知限制与延期项](docs/operations/known-issues.md)：AI、邮件、限流和赛后路线。
- [比赛演示运行手册](docs/product/competition-demo-runbook.md)：比赛版主路径和故障回退。
- [贡献指南](CONTRIBUTING.md)：分支、提交和 PR 约定。
- [安全策略](SECURITY.md)：漏洞报告和敏感信息处理。
