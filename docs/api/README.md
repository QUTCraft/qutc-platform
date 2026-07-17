# API 契约工作流

`openapi.yaml` 是公开门户 API 的唯一契约源。前端的 `apps/web/src/api/portal.ts` 中每个请求都必须对应一个 `operationId`；任何字段或路径改动先更新 OpenAPI，再更新前后端实现。

## Apifox

1. 在 Apifox 新建项目，选择“导入数据”。
2. 导入本目录的 `openapi.yaml`（OpenAPI 3.1）。
3. 建立本地环境：`http://localhost:8080`；生产环境由部署配置提供。
4. Apifox 只负责接口协作、Mock 与测试用例；仓库中的 YAML 仍是需要评审、需要提交的事实来源。

## Swagger

后端启动后将同一份 OpenAPI 契约提供给 Swagger UI。开发期间可以用任意支持 OpenAPI 3.1 的 Swagger/Redoc 工具加载 `openapi.yaml` 预览。

## 公开门户边界

- 所有路径以 `/api/v1/portal` 开头，且只返回已发布的公开数据。
- 列表响应统一为 `{ data: [], meta: { request_id, page, page_size, total } }`。
- 单对象响应统一为 `{ data: {}, meta: { request_id } }`。
- 认证、草稿、成员隐私、审核、后台 Dashboard 和 RCON 命令不属于 Portal API。
- 资源下载地址由服务端返回短时受控 URL，前端不得暴露或拼接对象存储凭据。
