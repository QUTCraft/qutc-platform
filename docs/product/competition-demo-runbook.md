# 比赛演示运行手册

## 1. 演示定位

主叙事使用“面向校园社团与民间组织的数字化管理与公共内容平台”。QUTCraft 是真实落地案例，不把 Minecraft 或 RCON 作为通用产品的必要前提。门户承担宣传和公开内容分发，Admin 承担内容、成员、项目、审批、配置、审计及 AI 辅助生产。

## 2. 环境准备

通用比赛资料建议：

```dotenv
DEFAULT_ORGANIZATION_SLUG=campus-commons
DEMO_SEED_ENABLED=true
DEMO_SEED_PROFILE=generic
AI_PROVIDER=openai_compatible
AI_BASE_URL=https://provider.example/v1
AI_API_KEY=<由部署环境注入>
AI_MODEL=<已验证模型>
```

正式环境还必须替换 JWT、数据库、Bootstrap Admin、MinIO 和 SMTP 凭据，关闭公开端口暴露，并使用 HTTPS。启动后依次检查 `/healthz`、`/readyz`、Portal 首页和 `/admin`。

## 3. 推荐演示路径

1. 打开 Portal，说明组织公开资料、动态、项目、资源和知识库全部来自只读 Portal API。
2. 登录 Admin，在系统设置修改组织标语并保存；刷新 Portal 展示即时生效和缓存失效。
3. 创建 Markdown 内容，插入图片或附件，实时预览；保持草稿时 Portal 不可见。
4. 发布内容并在 Portal 查看；随后下线，证明公开边界和状态机有效。
5. 创建成员邀请，说明 SMTP 关闭时可复制链接、启用时记录真实投递状态和失败重试。
6. 展示项目成员和里程碑协作。
7. 从 Portal 提交申请，在 Admin 审批；展示业务决定、独立 ServerAdapter 同步结果和审计。明确当前适配器为 Mock。
8. 在内容编辑器调用“从知识生成”，选择组织知识、查看引用、人工确认草稿，再由人发布。
9. 打开审计页，用 Request ID 关联管理动作和服务日志。
10. 展示自定义 Portal Manifest 的草稿、启用、自动回退和永久恢复默认 MD3。

## 4. 故障回退

- 真实模型不可用：切换 `AI_PROVIDER=disabled`，展示已完成的接口治理和确定性录屏，不把 Mock 称作真实模型。
- 自定义门户异常：访问 `/?portal=md3` 临时回退，随后在 Admin 执行“恢复默认 MD3”。
- SMTP 不可用：复制邀请链接，投递状态保持 `failed` 或 `disabled`。
- ServerAdapter 失败：审批事实保持成功，同步记录显示失败并可安全重试。
- 数据异常：停止写入，按备份恢复手册从升级前快照恢复。

## 5. 演示前门禁

```powershell
.\scripts\run-quality-gate.ps1 -Integration
cd apps/web
pnpm test:e2e
```

同时完成一次全新数据库迁移和一次旧数据卷升级；确认 `schema_migrations` 从 001 连续到最新版本。最终答辩前查看 [已知限制](../operations/known-issues.md)，确保口径与实际实现一致。
