# CHANGELOG

All notable changes to the QUTCraft Platform will be documented in this file.

## [v0.3.0] - 2026-07-22

### 🏛️ ADMIN 管理工作台与登录页面全新排版重构 (Admin Workspace & Login Overhaul)
- **全新双列双屏登录页面 (`LoginView.vue`)**:
  - 重构登录页面为现代双列布局：左侧为带有 QUTCraft Network 品牌视觉与特色标牌的渲染 Banner；右侧为宽敞极简的登录表单与高对比度输入框。
  - 彻底移除了界面上生硬冗余的开发文字（如 `JWT 与 RBAC` / `Mock 预填` 等）。
- **重构 ADMIN 响应式排版架构 (`AdminLayout.vue`)**:
  - 侧边栏 (`.admin-rail`): 拓宽至 `270px`，导航项增加充裕间距（`min-height: 48px`）。底部账号卡片 (`.account-chip`) 重构为双行布局与独立 Hover 退出按钮，解决头像、姓名与权限标签拥挤挤叠的问题。
  - 顶栏 (`.admin-topbar`): 清理了杂乱文案，保留规范的页面标题、查看门户快捷按钮与主题切换。
  - 主工作区 (`.admin-content`): 拓宽至 `1360px`，指标卡 (`.metric-grid`) 改为 `auto-fit` 弹性自适应网格，解决卡片挤压变窄的问题。
- **界面文本精简与 API 规范整合 (`docs/api/API.md`)**:
  - 彻底清理了管理页面中的技术杂音文案（如 `POST /api/v1/portal/apply` / `RCON 限制` / `认证接口将在...`）。
  - 移除了临时创建的 `docs/API_STANDARDS.md`，将 SMTP 邮件适配器与 RCON 审计规范完整补充合并至既有的 [docs/api/API.md](file:///d:/qutc-platform/docs/api/API.md) 中。

---

## [v0.2.1] - 2026-07-22

### 🐛 修复 ADMIN 管理工作台 Monet 配色遗漏与按钮文字重合 Readability 问题
- **管理工作台全面参与 Monet 取色 (Admin Area Full Monet Sync)**:
  - 修复了 Admin Rail 侧边栏、TopBar 顶部栏、Admin Panel 面板、Cards、Forms、Empty States、Tags、Code 块与表格未完整参与 Monet 动态主题色彩覆盖的问题。
- **重构按钮文字与背景色重合 bug (Element Plus Button Readability Fix)**:
  - 主按钮 (`.el-button--primary`) 的 span 文字与图标固定映射至高对比度的 `--md-sys-color-on-primary`，禁用态与默认态均具备 100% 极高可读性。

---

## [v0.2.0] - 2026-07-22

### 🎨 Material Design 3 Monet 动态四季与昼夜色彩引擎
- **严格基于本地时区与系统时间自动推演**:
  - **季节感应 (Month-based Seasons)**: 春🌸 / 夏🌿 / 秋🍁 / 冬❄️。
  - **昼夜感应 (Hour-based Day & Night)**: 日光白天 (07:00-18:59) / 静谧夜幕 (19:00-06:59)。
