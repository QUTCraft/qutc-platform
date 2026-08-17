# 比赛演示运行手册

## 1. 演示定位

主叙事使用“社团智枢 Commons Agent：面向校园组织的 AI 活动策划与运营服务平台”。活动策划智能体是比赛主服务，CMS、项目、知识库、RBAC 和审计是它可靠执行的底座。QUTCraft 是真实落地案例，Minecraft 文化背景不进入通用管理能力。

## 2. 环境准备

通用比赛资料建议：

```dotenv
DEFAULT_ORGANIZATION_SLUG=campus-commons
VITE_ORGANIZATION_SLUG=campus-commons
DEMO_SEED_ENABLED=true
DEMO_SEED_PROFILE=generic
DEMO_SEED_MULTI_ORGANIZATION=true
AI_PROVIDER=openai_compatible
AI_BASE_URL=https://provider.example/v1
AI_API_KEY=<由部署环境注入>
AI_MODEL=<已验证模型>
```

正式环境还必须替换 JWT、数据库、Bootstrap Admin、MinIO 和 SMTP 凭据，关闭公开端口暴露，并使用 HTTPS。启动后依次检查 `/healthz`、`/readyz`、Portal 首页和 `/admin`。

## 3. 推荐演示路径

1. 从 Campus Commons 门户进入登录页，确认门户、登录页和 Admin 均采用通用组织叙事；在顶栏确认当前为 Campus Commons。需要说明真实落地时可切换到 QUTCraft，组织角色、知识、项目和内容不会串租户，Minecraft 服务器卡片与白名单同步仅在 QUTCraft 场景出现。
2. 进入“活动策划”，填写校园活动目标、受众、时间、场地、预算和风险约束。
3. 检索并显式选择当前组织的活动规范与历史资料，说明跨组织内容不会进入模型上下文。
4. 使用真实模型生成带引用方案，展示模型模式、固定引用版本和生成状态。
5. 由评审人完成五维人工评分，展示组织级质量概览和真实模型/Prompt 分组；说明自动结构检查不能替代人工质量判断。
6. 审查建议操作，只批准项目、准备/宣传/执行/复盘里程碑和公告草稿中的需要项。
7. 打开项目管理和内容编辑器，验证创建结果；强调项目默认非公开，公告仍是草稿。
8. 人工检查公告后发布，在 Portal 查看，再下线，证明 AI 没有绕过 CMS 状态机。
9. 打开审计页，用 Request ID 回溯活动策划、评分、项目、里程碑和草稿创建动作。
10. 时间允许时展示多组织切换、批量邀请与预创建账户、组织模板、申请审批通知队列、内容修订恢复、资源下载统计和自定义 Portal 回退，作为平台完整度而非 AI 主线。

## 4. 故障回退

- 真实模型不可用：切换 `AI_PROVIDER=disabled`，展示已完成的接口治理和确定性录屏，不把 Mock 称作真实模型。
- API 在任务处理中重启：未领取的 `queued` 任务会在恢复后继续执行；被中断的 `running` 任务会明确标记为 `failed`，重新创建任务，不宣称断点续跑。
- 自定义门户异常：访问 `/?portal=md3` 临时回退，随后在 Admin 执行“恢复默认 MD3”。
- SMTP 不可用：复制邀请链接；邀请投递和审批通知保持 `failed` 或 `disabled`，审批事实不回滚，可在 Admin 查看并在恢复后重试通知。
- 数据异常：停止写入，按备份恢复手册从升级前快照恢复。

## 5. 演示前门禁

```powershell
.\scripts\run-quality-gate.ps1 -Integration
.\scripts\run-ai-activity-demo-rehearsal.ps1 -Rounds 3
.\scripts\run-multi-organization-demo-rehearsal.ps1 -RoundsPerOrganization 1
cd apps/web
pnpm test:e2e
```

稳定性脚本只验证真实 API 生成、Prompt v2、固定引用和六类建议操作，不自动评分或批准；报告默认进入被忽略的 `tmp/agent-rehearsal`，不保存生成正文。演示人员仍需在管理端完成一次真实人工评分和建议批准。

2026-08-02 使用 `agnes-2.0-flash` 的首轮真实 API 演练为 3/3，平均 13.2 秒，最慢 21.6 秒。该结果只证明当次技术链路稳定，答辩前仍须在最终网络环境重跑，并由演示人员完成评分与批准。

2026-08-05 使用同一真实模型完成双组织演练：QUTCraft 1/1（15.8 秒）与 Campus Commons 1/1（10.0 秒）均生成 Prompt v2、至少一条组织内引用和 6 类建议操作。脚本未自动评分、批准或发布，报告不保存生成正文；该结果证明组织切换后的模型、知识和方案链路可以分别工作，不替代最终人工验收。

同时完成一次全新数据库迁移和一次旧数据卷升级；确认 `schema_migrations` 从 001 连续到 019，既有内容已由 015 幂等补齐 v1 修订基线，媒体资产已由 018 补齐外部提供方字段，组织已由 019 补齐备案与官网 Logo 字段。最终答辩前查看 [已知限制](../operations/known-issues.md)，确保口径与实际实现一致。
