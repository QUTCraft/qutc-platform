# 媒体存储适配规范

> 适用版本：`v0.1.0-mvp`
> 支持后端：本地受控目录、S3 兼容对象存储（含 MinIO）

## 1. 目标与边界

媒体上传和下载始终经过 QUTCraft API。Portal、Admin 和自定义门户只获得受控 API 地址，不获得 bucket、对象键、Access Key、Secret Key 或对象存储内部地址。

存储失败不得创建只有数据库元数据、没有文件对象的资产记录；数据库写入失败时，API 必须尽力删除已经写入的对象。

## 2. 部署默认值与网页配置

下列变量只提供部署默认值。组织管理员可在“系统设置 → 服务接入”选择服务器本地或 S3/MinIO，并保存服务地址、存储桶和凭据；保存后新上传即时使用新驱动，无需重启。Access Key 与 Secret Key 加密保存，读取接口只返回配置状态和尾号提示。本地目录始终由部署方固定，网页不能填写任意服务器路径。

| 变量 | local | s3 | 说明 |
| --- | --- | --- | --- |
| `STORAGE_DRIVER` | 必填，值为 `local` | 必填，值为 `s3` | 未设置时默认为 `local`。 |
| `STORAGE_LOCAL_ROOT` | 必填 | 仅用于旧数据兼容/迁移 | 默认 `/tmp/qutcraft-uploads`。 |
| `S3_ENDPOINT` | 不使用 | 必填 | `host:port`，不能包含 `http://` 或 `https://`。 |
| `S3_ACCESS_KEY` | 不使用 | 必填 | 仅后端读取。 |
| `S3_SECRET_KEY` | 不使用 | 必填 | 仅后端读取。 |
| `S3_BUCKET` | 不使用 | 必填 | 默认 `qutcraft-media`，不存在时由 API 创建。 |
| `S3_REGION` | 不使用 | 可选 | 默认 `us-east-1`。 |
| `S3_USE_SSL` | 不使用 | 可选 | 生产公网对象存储应使用 `true`。 |

网页接口为 `GET/PATCH /api/v1/admin/integrations`。`POST /api/v1/admin/integrations/test` 传 `{ "section": "storage" }` 时只检查本地目录或现有 S3 Bucket，不创建 Bucket、不上传或删除对象。

### 2.1 后台快捷上传入口

管理端 `/admin/assets` 是独立的资源文件工作区，适合不需要先打开 Markdown 编辑器的日常上传：

- 支持拖拽或多选文件，前端按队列逐个调用 `POST /api/v1/admin/assets`；服务端仍执行登录、组织范围、MIME 签名和 10 MiB 单文件限制。
- 上传时可选择当前组织的一条内容建立 `content_id` 引用。未关联的文件只对后台管理员可用；关联内容发布后，Portal 才能使用受控公开下载地址。
- 对未关联文件，具备 `asset:manage` 与 `content:publish` 的管理员可点击“归档到门户”。前端调用 `POST /api/v1/admin/assets/{asset_id}/publish`，服务端以单个事务创建并发布 `resource` 内容、绑定文件、写入两次内容修订和审计记录；任一步失败都会整体回滚，原文件继续保持后台私有状态。
- 列表由 `GET /api/v1/admin/assets` 提供，搜索只匹配原始文件名；下载按钮和复制按钮使用 API 返回的管理地址，不拼接 MinIO URL。
- 只有具备 `asset:manage` 且未被内容引用的文件可由 `DELETE /api/v1/admin/assets/{asset_id}` 清理。已关联文件统一返回 `409 asset.in_use`，避免误删门户正文或资源下载。

这个页面只展示存储驱动、Endpoint 和 Bucket 等非敏感状态；Access Key、Secret Key 和对象键永不下发到浏览器。MinIO 已由 1Panel 等外部系统部署时，只需要确保 API 容器可访问该 Endpoint，然后在“系统设置 → 服务接入”保存并验证，不要让浏览器直接访问 MinIO API。

生产环境拒绝 MinIO 示例账号和包含 `change-me` 的示例密钥。

## 3. 对象与元数据

对象键由服务端生成，格式为：

```text
{organization_id}/{asset_uuid}.{safe_extension}
```

`media_assets.storage_driver` 记录对象所属驱动，`storage_path` 只在服务端内部保存对象键或兼容旧版的本地绝对路径。两者均不会进入 Portal API。

切换存储驱动不会自动搬迁已有对象。资产记录会保留实际驱动，API 下载时按记录解析本地或 S3，因此切换后旧对象仍可读取；如果旧驱动的服务或凭据已不可用，则返回 `503 asset.storage_driver_unavailable`，不会尝试把本地路径当成 S3 Key。

## 4. 上传事务边界

1. 在解析 multipart 前限制请求体为 11 MiB。
2. 校验单文件 10 MiB、文件签名 MIME、文件名和内容组织归属。
3. 写入当前存储驱动。
4. 写入资产元数据。
5. 元数据失败时补偿删除对象。
6. 成功后失效组织 Portal 缓存。

存储不可用返回 `503 asset.storage_unavailable`。输入非法仍使用 `400/413`，不会被错误归类为存储故障。

## 5. 下载授权

- Admin 下载必须具备 `asset:read`，且资产属于当前组织。
- Portal 下载要求资产已关联当前组织中处于 `published` 状态的内容。
- 公开图片使用 `inline`；其他文件使用 `attachment`。
- API 流式读取对象，不签发长期对象存储直链。

## 6. 本地 MinIO 验证

在 `deploy/compose/.env` 中设置：

```dotenv
STORAGE_DRIVER=s3
S3_ENDPOINT=minio:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin-change-me
S3_BUCKET=qutcraft-media
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin-change-me
```

启动：

```powershell
docker compose -f deploy/compose/docker-compose.yml --profile storage up -d --build
.\scripts\run-storage-integration.ps1
# 或连同常规门禁执行：
.\scripts\run-quality-gate.ps1 -Integration -StorageIntegration
```

集成脚本执行真实 `Put → Open → Delete → 404`，测试对象在结束时清理。
