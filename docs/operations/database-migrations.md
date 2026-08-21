# 数据库迁移与回退规范

## 1. 运行机制

`apps/api/migrations` 是数据库结构的唯一事实来源。SQL 文件按三位数字前缀排序，并通过 Go `embed` 编译进 API 二进制。API 启动时：

1. 创建 `schema_migrations` 账本；
2. 检查已登记版本；
3. 顺序执行尚未登记的 SQL；
4. 每个文件完整执行成功后写入版本号；
5. 任一语句失败时终止 API 启动，不继续 seed，也不把失败版本登记为成功。

不得重新引入启动时 `AutoMigrate`，不得在生产库手工修改表后不补迁移文件。Compose 不再把 SQL 挂载到 MySQL 的一次性初始化目录，因此全新数据库与已有数据卷使用同一条迁移路径。

## 2. 旧数据卷兼容

001—008 发布期间 API 使用 GORM `AutoMigrate`。检测到已有 `organizations` 表但迁移账本为空时，迁移器会登记 001—008 为历史基线，再从 009 开始执行。这一兼容分支只用于升级旧版数据卷；全新数据库从 001 开始执行。

当前仓库最新版本为 `020_content_review_workflow.sql`。升级完成后，账本应连续包含 001—020；不能只根据某个新增列存在就跳过后续数据回填。

升级后应检查：

```sql
SELECT version, applied_at FROM schema_migrations ORDER BY version;
```

## 3. 新增迁移

- 文件名使用 `NNN_short_description.sql`，编号只能递增。
- 已进入 `main` 或已被任一共享环境执行的文件禁止修改；修正必须追加新版本。
- SQL 必须兼容 MySQL 8.4，不得依赖本地客户端命令。
- 大表变更需先评估锁表时间；数据回填与结构变更可拆为多个版本。
- 新字段必须同步模型、OpenAPI、测试、备份清单和敏感数据分级。

## 4. 发布前升级演练

1. 使用当前线上版本创建数据库和测试数据。
2. 运行备份脚本并校验清单与 SHA-256。
3. 用待发布 API 启动同一数据卷。
4. 检查 `/readyz`、`schema_migrations`、核心闭环和审计。

010 的 `activity_plans` 迁移不新增数据库外键。原因是 001—008 期间的历史 AutoMigrate 数据卷可能同时存在 `utf8mb4_unicode_ci` 与 `utf8mb4_0900_ai_ci` 标识列，直接建立跨表外键会使旧卷无法升级。组织归属、运行、项目和内容引用由服务端事务、组织过滤与集成测试强制校验；新迁移不得为了补外键原地修改历史表 collation。

011 是数据型策略迁移，把已有组织的 `activity-planner` 定义升级为 `activity-planner/v2`。v2 将结构化活动简报与知识来源统一编码为不可信 JSON 数据，并强化 Prompt Injection、密钥泄露、工具越权和绕过人工批准的系统策略。启动 seed 只同步定义的静态元数据，不覆盖组织级启停与配额配置。

012 新增 `activity_plan_evaluations`，以 `(plan_id, reviewer_user_id)` 唯一约束保存五维人工评分；`organization_id` 同时参与所有服务端查询，防止跨组织读取。该表沿用 010 的旧数据卷兼容策略，不新增跨表外键；删除测试方案或执行数据清理时必须先删除对应评分。

013 为 `refresh_tokens` 新增 `organization_id` 与索引。迁移前签发的历史 Refresh Token 使用空字符串兼容值，并在首次轮换时回退到用户最早的 active 组织；迁移后签发和轮换的令牌始终记录当前组织。该列刻意不建立组织外键，以允许历史空值安全过渡；服务端每次 Refresh 和组织切换仍会实时校验用户与目标组织的 active 成员关系。

014 是延期增强的结构基础：

- `users.default_organization_id` 保存用户显式切换后的默认组织偏好；登录时仍会实时验证 active 成员关系，无效偏好自动回退。
- `organizations` 增加邀请邮件主题/正文模板；模板只允许受控变量，不保存 SMTP 凭据。
- `media_assets` 增加下载次数与最近下载时间。
- `agent_citations.source_body` 保存创建运行时的引用正文快照，使 queued 任务可在进程重启后重建上下文。
- 新增 `content_revisions` 和 `notification_outboxes`，分别承载不可变内容版本与申请审批通知队列。

014 在兼容旧 AutoMigrate 数据卷时必须注意标识列 collation：历史 `contents.id` 可能为 `utf8mb4_0900_ai_ci`，组织和用户标识可能为 `utf8mb4_unicode_ci`。迁移文件已对外键列显式声明匹配 collation；不要删除这些声明或用 `AutoMigrate` 重建表。

015 是幂等数据回填：仅为尚无任何修订的既有 `contents` 写入 version 1 快照，保留原作者、状态、正文、发布时间和创建时间。已有修订的内容不会被重复写入。该迁移必须在 014 完整成功并登记后执行。

016 为组织智能体配置增加模型提供方、兼容 API 地址、加密 API Key 和模型名；017 增加组织级门户、SMTP 与媒体存储运行时配置。两者的密钥列只保存服务端加密后的密文，不能通过查询接口返回明文。

018 为 `media_assets` 增加 `provider` 与 `external_url`，用于兼容 Superbed 等外部图床与本地/对象存储回退。既有资产以 `provider=local` 安全补齐；该迁移修复了模型字段已更新但旧数据卷缺列时上传和集成测试失败的问题。

019 为 `organizations` 增加备案号字符串和官网 Logo 资产引用。Logo 引用刻意不建立跨表外键：媒体资产删除流程会在同一应用事务中清空引用，既兼容历史数据卷的 collation，也允许存储对象与元数据按现有补偿边界清理。

020 新增 `content_review_requests`，持久化发布审核与下线申请的提交版本、说明、反馈、处理人和处理时间；同时把通知 Outbox 唯一键扩展为“事件 + 目标 + 收件邮箱”，允许一次内容审核安全通知多个审核者。`content_id` 与 `revision_id` 由服务事务和删除清理逻辑维护，不建立跨表外键，以兼容旧 AutoMigrate 数据卷中 `contents.id` 的历史 collation；审核请求仍固定保存不可变修订 ID，且 `review` 状态禁止改稿。

5. 在空数据库再次启动，验证 001 至最新版本可完整执行。

## 5. 回退边界

MySQL DDL 可能隐式提交，本项目不承诺对任意结构迁移执行自动反向 SQL。发布回退采用“应用兼容优先，备份恢复兜底”：

- 扩展型迁移优先新增可空字段或带安全默认值，先发布兼容旧字段的应用。
- 删除、重命名和不可逆数据转换必须拆成跨版本流程，在确认旧版本不再使用后执行。
- 迁移失败或发布后发现数据不兼容时，停止写入并按 [备份恢复手册](backup-restore.md) 恢复升级前快照。
- 不得只删除 `schema_migrations` 记录来伪造回退；账本必须与真实结构一致。
