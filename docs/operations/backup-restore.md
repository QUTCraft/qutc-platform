# Compose 备份与恢复手册

> 适用范围：QUTCraft Platform 单服务器 Docker Compose 部署
> 当前实现：MySQL 8.4 + `STORAGE_DRIVER=local` 媒体卷
> 备份格式：`qutc.backup/v1`

## 1. 目标与边界

一次完整备份包含：

- `database.sql`：使用 `mysqldump --single-transaction` 导出的业务数据库；
- `media.tar.gz`：本地 `media_data` Docker 卷中的媒体对象；
- `media-files.sha256`：每个媒体文件的 SHA-256；
- `manifest.json`：创建时间、源数据库、存储驱动、逐表行数以及所有备份文件的大小和 SHA-256。

备份目录包含成员数据、内容、Refresh Token 哈希、审计记录和媒体文件，必须按敏感数据管理。仓库已忽略 `backups/`，但运维人员仍应使用受控目录、最小权限、磁盘加密和异地副本；不得上传到公开网盘、Issue 或 Git。

Redis 只保存可重建缓存，不进入备份。恢复后可以使用空 Redis，由 API 从 MySQL 重建公开缓存。

## 2. 创建备份

先确认 Compose 正常运行：

```powershell
cd D:\qutc-platform
docker compose -f deploy/compose/docker-compose.yml ps
```

执行：

```powershell
.\scripts\backup-compose.ps1
```

默认写入 `backups/qutcraft-<UTC时间>/`。脚本会短暂暂停 API 容器，依次导出 MySQL 与媒体卷，再恢复 API，以避免数据库元数据和本地文件在备份窗口内继续变化。暂停不影响 MySQL/Redis，通常只持续数秒；应在维护窗口执行。

只有已经通过外部只读/维护措施阻止写入时，才允许：

```powershell
.\scripts\backup-compose.ps1 -NoPauseWrites
```

数据库临时 dump 位于 MySQL 容器 `/tmp`，无论成功或失败都会尝试删除。成功目录没有 `.incomplete`；失败目录保留该标记，恢复脚本会拒绝使用。

## 3. 隔离恢复验证

每个准备保留的备份都应立即演练：

```powershell
.\scripts\verify-backup-restore.ps1 -BackupPath .\backups\qutcraft-20260730T081809Z
```

验证脚本不会覆盖当前数据库或媒体卷。它会：

1. 校验 manifest 中所有文件的大小与 SHA-256；
2. 创建随机名称的临时 MySQL 数据库并导入 `database.sql`；
3. 比较完整表集合与每张表的行数；
4. 创建随机名称的临时 Docker 卷并解压媒体；
5. 比较每个媒体文件的 SHA-256；
6. 在 `finally` 中删除临时 SQL、数据库和卷。

成功标志为 `BACKUP_RESTORE_VERIFY_OK`。任何一步失败都必须把该备份标记为不可用并调查原因，不能只因为 `mysqldump` 命令退出成功就认为可恢复。

完整自动演练：

```powershell
.\scripts\run-backup-restore-rehearsal.ps1

# 或连同全部质量门禁执行
.\scripts\run-quality-gate.ps1 -Integration -BackupRestore
```

自动演练使用系统临时目录，并仅在恢复验证成功后删除备份数据。

## 4. 正式灾难恢复

当前 MVP 不提供“覆盖正在运行的源数据库”按钮。正式恢复必须使用新的数据库/卷，验证后再切换服务，避免误操作破坏最后一份可用数据：

1. 停止 Web/API 写入，保留原 MySQL 和媒体卷，不执行 `down -v`。
2. 将备份复制到受控恢复主机并再次运行隔离验证。
3. 建立新的 MySQL 数据库和新的媒体卷。
4. 导入 `database.sql`，将 `media.tar.gz` 解压到新卷。
5. 按 manifest 复查表集合、行数和媒体 SHA-256。
6. 更新 Compose 目标或卷映射，启动 API，等待 `/readyz` 返回 `ready`。
7. 执行 S1—S6 集成与 Portal/Admin 冒烟。
8. 验证完成前不得删除旧数据库、旧卷和原始备份。

生产恢复应由两人复核目标数据库与卷名称。任何包含 `docker compose down -v`、删除源数据库或清空现有媒体卷的操作都不属于本手册的自动流程。

## 5. S3/MinIO 模式

当 `STORAGE_DRIVER=s3` 时，本脚本拒绝把本地卷冒充完整媒体备份。可使用：

```powershell
.\scripts\backup-compose.ps1 -SkipMedia
```

这只生成数据库备份。媒体必须使用对象存储提供方的 bucket versioning、复制或快照能力单独保护，并记录与数据库备份对应的时间点。MinIO 单机演示环境至少应备份 `minio_data`，生产环境应优先使用对象锁、版本控制和异地复制；未验证 bucket 恢复前，S3 备份不能计为通过。

## 6. 频率与保留

- 开发/比赛环境：重要演示前、迁移前和每日结束时至少一次。
- 真实社团环境：每日全量，重大发布或迁移前额外一次。
- 至少保留“本机近期副本 + 异地加密副本”，并定期验证最旧与最新备份。
- 备份删除按数据保留策略执行，不能把“磁盘不足”作为未经验证就删除唯一可恢复副本的理由。

每次演练记录：备份时间、版本/提交、表数量、媒体文件数量、验证结果、耗时、操作者和异常；记录本身不得包含 SQL、文件内容或凭据。
