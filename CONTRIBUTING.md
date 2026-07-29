# 贡献指南

感谢你参与 QUTC 项目。

## 开始之前

在开始开发前，请先：

1. 搜索是否已经存在相关 Issue。
2. 对较大的功能先创建 Feature Issue。
3. 等维护者确认方案后再开始实现。
4. 不要在 Issue、代码或日志中提交密码和敏感信息。

## 开发流程

1. 从最新的 `main` 分支创建新分支。
2. 在新分支上完成修改。
3. 执行项目要求的测试和代码检查。
4. 提交 Pull Request。
5. 根据评审意见修改。
6. CI 通过并获得批准后合并。

## 分支命名

使用以下格式：

- `feat/123-member-profile`
- `fix/208-login-timeout`
- `docs/contribution-guide`
- `chore/update-dependencies`
- `refactor/project-service`

如果存在对应 Issue，请在分支名中包含 Issue 编号。

## Commit 建议

推荐使用 Conventional Commits：

- `feat: add member profile`
- `fix: prevent duplicate project members`
- `docs: update deployment guide`
- `refactor: simplify permission checks`
- `test: add project service tests`
- `chore: update dependencies`

一次提交应尽量只完成一类修改。

## Pull Request 要求

Pull Request 应当：

- 说明为什么需要这项修改
- 说明具体修改内容
- 关联对应 Issue
- 提供测试结果
- 涉及界面时提供截图
- 不包含无关格式化或重构
- 不包含密码、Token、私钥或真实用户数据

## 代码评审

未经评审，不应直接向 `main` 推送代码。

评审重点包括：

- 功能是否正确
- 权限检查是否完整
- 是否引入安全风险
- 数据库迁移是否安全
- API 是否保持兼容
- 是否有必要的测试和文档

## 安全问题

安全漏洞不要通过公开 Issue 报告，请按照 `SECURITY.md` 中的方式联系维护团队。
