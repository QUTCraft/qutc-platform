# API 契约工作流

`openapi.yaml` 是 Portal 与 Admin API 的唯一机器可读契约源。前端的 `apps/web/src/api/portal.ts` 和 `apps/web/src/api/admin.ts` 中每个请求都必须对应一个 `operationId`；任何字段或路径改动先更新 OpenAPI，再更新前后端实现。

面向实现者的完整中文说明见 [API.md](API.md)。它解释了响应封装、认证、权限边界、字段、错误语义与调用示例；字段和路径以 `openapi.yaml` 为准。

## Apifox

1. 在 Apifox 新建项目，选择“导入数据”。
2. 导入本目录的 `openapi.yaml`（OpenAPI 3.1）。
3. 建立本地环境：`http://localhost:8080`；生产环境由部署配置提供。
4. Apifox 只负责接口协作、Mock 与测试用例；仓库中的 YAML 仍是需要评审、需要提交的事实来源。

## Swagger

后端启动后将同一份 OpenAPI 契约提供给 Swagger UI。开发期间可以用任意支持 OpenAPI 3.1 的 Swagger/Redoc 工具加载 `openapi.yaml` 预览。

## API 边界

- 所有路径以 `/api/v1/portal` 开头，且只返回已发布的公开数据。
- 列表响应统一为 `{ data: [], meta: { request_id, page, page_size, total } }`。
- 单对象响应统一为 `{ data: {}, meta: { request_id } }`。
- Portal API 无认证，只读且只返回已发布的公开数据。
- Admin API 以 `/api/v1/admin` 开头，必须携带 Bearer JWT，并由服务端以组织成员身份和 RBAC 二次授权。
- 草稿、成员隐私、审核、后台 Dashboard 和 RCON 命令只允许出现在 Admin API，绝不可从 Portal API 泄露。
- 资源下载地址由服务端返回短时受控 URL，前端不得暴露或拼接对象存储凭据。
