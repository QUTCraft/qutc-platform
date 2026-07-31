# 媒体存储适配规范

> 适用版本：`v0.1.0-mvp`
> 支持后端：本地受控目录、S3 兼容对象存储（含 MinIO）

## 1. 目标与边界

媒体上传和下载始终经过 QUTCraft API。Portal、Admin 和自定义门户只获得受控 API 地址，不获得 bucket、对象键、Access Key、Secret Key 或对象存储内部地址。

存储失败不得创建只有数据库元数据、没有文件对象的资产记录；数据库写入失败时，API 必须尽力删除已经写入的对象。

## 2. 驱动配置

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

生产环境拒绝 MinIO 示例账号和包含 `change-me` 的示例密钥。

## 3. 对象与元数据

对象键由服务端生成，格式为：

```text
{organization_id}/{asset_uuid}.{safe_extension}
```

`media_assets.storage_driver` 记录对象所属驱动，`storage_path` 只在服务端内部保存对象键或兼容旧版的本地绝对路径。两者均不会进入 Portal API。

切换存储驱动不会自动搬迁已有对象。迁移时必须先复制对象并更新 `storage_driver`/`storage_path`；在对应驱动未启用时读取旧对象返回 `503 asset.storage_driver_unavailable`，不会尝试把本地路径当成 S3 Key。

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
