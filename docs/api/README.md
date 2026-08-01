# API 契约工作流

`openapi.yaml` 是 Auth、Portal 与 Admin API 的唯一机器可读契约源。前端的 `apps/web/src/api/auth.ts`、`apps/web/src/api/portal.ts` 和 `apps/web/src/api/admin.ts` 中每个请求都必须对应一个 `operationId`；任何字段或路径改动先更新 OpenAPI，再更新前后端实现。

面向实现者的完整中文说明见 [API.md](API.md)。它解释了响应封装、认证、权限边界、字段、错误语义与调用示例；字段和路径以 `openapi.yaml` 为准。

申请审批、服务器同步、失败重试、Mock 语义和受限命令的完整状态规范见 [申请审批与 ServerAdapter API 规范](server-adapter.md)。

成员邀请的 SMTP 配置、投递状态、token 轮换重试和凭据边界见 [邀请邮件适配器规范](email-adapter.md)。

Request ID、结构化日志、存活/就绪探针和组织隔离审计查询见 [API 可观测性与审计规范](observability.md)。

组织运营智能体的 7 条已实现接口、组织配置页、RBAC、异步状态、模型供应商配置、错误码与审计见 [组织运营智能体 API 规范](ai-agent.md)。

门户 Manifest 的字段、安全边界、草稿/启用 API 与默认 MD3 回退规则见 [自定义门户 Manifest v1](../product/portal-manifest-v1.md)。

## Apifox

1. 在 Apifox 新建项目，选择“导入数据”。
2. 导入本目录的 `openapi.yaml`（OpenAPI 3.1）。
3. 导入 `apifox/core-smoke.postman_collection.json`，格式选择 Postman Collection 2.1。
4. 导入 `apifox/local.postman_environment.json`，选择该环境后在本机填写 `adminEmail` 与 `adminPassword`；模板不会保存真实凭据。
5. Compose 默认 API 地址为 `http://localhost:18080`。每次完整运行前清空 `runId`、`contentId`、`applicationId`、`knowledgeSourceId`、`agentRunId` 和 `accessToken`，然后按集合顺序执行。
6. Apifox 只负责接口协作、Mock 与测试用例；仓库中的 YAML 仍是需要评审、需要提交的事实来源。

核心集合会创建一篇带唯一 `runId` 的新闻草稿，验证“草稿不可见 → 发布可见 → 归档不可见”；随后验证申请审批，保存一组安全的默认智能体策略，并使用已有知识资料创建一条明确标识 Mock/真实模式的智能体运行。它会留下已归档内容、已拒绝申请、组织智能体配置和相关审计记录，因此只应在开发、测试或明确允许写入的演示环境运行。

## Swagger

后端启动后将同一份 OpenAPI 契约提供给 Swagger UI。开发期间可以用任意支持 OpenAPI 3.1 的 Swagger/Redoc 工具加载 `openapi.yaml` 预览。

## 契约检查

提交前运行以下脚本：

```bash
python scripts/lint-openapi.py
python scripts/check-openapi-routes.py
python scripts/check-web-api-contract.py
python scripts/check-apifox-collection.py
```

- `lint-openapi.py` 检查 OpenAPI 3.1、引用、Schema、operationId、tag、响应和 Admin/Portal 安全边界。
- `check-openapi-routes.py` 对照 Gin 与 OpenAPI 的方法、路径、operationId 和路径参数。
- `check-web-api-contract.py` 确认四个真实 TypeScript API client 的请求都存在于 OpenAPI。
- `check-apifox-collection.py` 确认核心集合的请求仍匹配 OpenAPI，且环境模板不包含密码或 Token。
- `scan-secrets.py` 扫描所有已跟踪及未忽略文件中的高置信度密钥格式，并阻止服务端密钥配置进入前端源码。

这些脚本依赖 PyYAML；如果本机尚未安装，执行 `python -m pip install pyyaml`。存活检查 `/healthz` 与 MySQL/Redis 就绪检查 `/readyz` 均纳入契约，以便 Compose 冒烟和文档检查使用同一套入口。

也可以在 PowerShell 中运行统一门禁：

```powershell
.\scripts\run-quality-gate.ps1
```

需要连同 S1—S6 的真实 Compose/MySQL/Redis 集成套件一起执行时使用：

```powershell
.\scripts\run-quality-gate.ps1 -Integration
```

只检查当前 Compose 的 Web 路由、安全头、自定义门户 404 与 API 健康状态时使用：

```powershell
.\scripts\run-quality-gate.ps1 -Runtime
```

仓库的 `.github/workflows/quality-gate.yml` 会在 push 与 pull request 上执行无外部服务的契约、Go 测试、前端类型检查和生产构建。真实 MySQL/Redis 集成与路由冒烟仍由本地或部署流水线的 `-Integration` 执行，避免 CI 用 Mock 掩盖服务工况。

## API 边界

- 所有路径以 `/api/v1/portal` 开头，且只返回已发布的公开数据。
- 列表响应统一为 `{ data: [], meta: { request_id, page, page_size, total } }`。
- 单对象响应统一为 `{ data: {}, meta: { request_id } }`。
- Portal API 无认证，只读且只返回已发布的公开数据。
- Auth API 负责注册、登录、刷新令牌、退出和当前会话；刷新令牌通过 HttpOnly Cookie 投递、在服务端只保存哈希并于刷新时轮换，Access Token 只驻留于前端内存。
- Admin API 以 `/api/v1/admin` 开头，必须携带 Bearer JWT，并由服务端以组织成员身份和 RBAC 二次授权。
- 草稿、成员隐私、审核、后台 Dashboard 和 RCON 命令只允许出现在 Admin API，绝不可从 Portal API 泄露。
- 资源下载地址由服务端返回受控 Portal 路径（未来可替换为短时签名 URL）；没有关联文件时返回 `null`，前端不得暴露或拼接对象存储凭据。
- 媒体存储通过服务端 `local`/`s3` 驱动切换，MinIO/S3 bucket、对象键和凭据不属于 API 契约；配置与迁移规则见 [媒体存储适配规范](storage-adapter.md)。
