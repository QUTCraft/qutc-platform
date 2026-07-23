# CHANGELOG

All notable changes to the QUTCraft Platform will be documented in this file.

## [v0.4.0] - 2026-07-23

### ✨ 公开门户首页视觉与交互全面翻新 (Public Portal Home Page Overhaul)
- **极具视觉冲击力的 Hero 区域 (`HomeView.vue`)**:
  - 引入 `hero-glow-backdrop` 动态背景弥散光晕与 `.hero-brand-pill` 官方运行徽章。
  - 新增 `.hero-stats-strip` 横向核心数据计数条（`30+ 公开项目` · `100+ 沉淀知识` · `24/7 生态服`）。
  - 重构服务器实时状态卡片为 `.hero-status` 高级 Glassmorphic 玻璃态高透面板，自带心跳脉冲与白名单一键跳转。
- **全新三大核心支柱卡片 (`.pillars-section`)**:
  - 引入 **建筑与学术复原** (🛠️)、**开源与技术研发** (📖) 与 **知识与文化沉淀** (📓) 三大色块图标卡片，配以悬浮抬升与 MD3 容器色彩。
- **最新动态与通知展示区 (`.news-layout`)**:
  - 重构头条新闻卡片 `.lead-news` 与右侧新闻行 `.news-row`，增加分类标签、平滑位移与更充裕的文字间距。
- **共享资源与公共知识库面板 (`.split-section`)**:
  - 提升了 `.compact-item` 列表项的 Hover 交互，图标与标题平滑位移动画。

---

## [v0.3.1] - 2026-07-23

### 🎨 项目页面状态筛选栏视觉重构
- **全新 MD3 悬浮胶囊筛选栏 (`ProjectsView.vue`)**:
  - 引入 `.filter-pills` 胶囊筛选栏，内嵌各项目状态的动态彩色感应圆点（进行中🟢 / 研究中🟣 / 已完成🔵）与数量 Badge 徽章。

---

## [v0.3.0] - 2026-07-22

### 🏛️ ADMIN 管理工作台与登录页面全新排版重构
- 全新双列双屏登录页面 (`LoginView.vue`)。
- ADMIN 响应式排版架构 (`AdminLayout.vue`)。
- 界面文本精简与 API 规范整合 (`docs/api/API.md`)。
