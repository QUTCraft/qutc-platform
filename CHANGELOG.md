# CHANGELOG

All notable changes to the QUTCraft Platform will be documented in this file.

## [v0.3.1] - 2026-07-23

### 🎨 项目页面状态筛选栏视觉重构 (Projects Filter Pills Redesign)
- **全新 MD3 悬浮胶囊筛选栏 (`ProjectsView.vue`)**:
  - 彻底移除了原先生硬简陋的 Element Plus 单选按钮组 (`el-radio-button`)。
  - 引入专为 Material Design 3 打造的 `.filter-pills` 胶囊筛选栏，内嵌各项目状态的动态彩色感应圆点（进行中🟢 / 研究中🟣 / 已完成🔵）与数量 Badge 徽章。
  - 优化了项目页头的层次布局，解决筛选按钮与大标题错位、重合的问题。

---

## [v0.3.0] - 2026-07-22

### 🏛️ ADMIN 管理工作台与登录页面全新排版重构
- **全新双列双屏登录页面 (`LoginView.vue`)**:
  - 重构登录页面为现代双列布局：左侧为带有 QUTCraft Network 品牌视觉与特色标牌的渲染 Banner；右侧为宽敞极简的登录表单与高对比度输入框。
- **重构 ADMIN 响应式排版架构 (`AdminLayout.vue`)**:
  - 侧边栏 (`.admin-rail`): 拓宽至 `270px`，导航项增加充裕间距（`min-height: 48px`）。
- **界面文本精简与 API 规范整合 (`docs/api/API.md`)**:
  - 将 SMTP 邮件适配器与 RCON 审计规范完整补充合并至既有的 [docs/api/API.md](file:///d:/qutc-platform/docs/api/API.md) 中。

---

## [v0.2.1] - 2026-07-22

### 🐛 修复 ADMIN 管理工作台 Monet 配色遗漏与按钮文字重合 Readability 问题
- **管理工作台全面参与 Monet 取色 (Admin Area Full Monet Sync)**:
  - 修复了 Admin Rail 侧边栏、TopBar 顶部栏、Admin Panel 面板、Cards、Forms、Empty States、Tags、Code 块与表格未完整参与 Monet 动态主题色彩覆盖的问题。
