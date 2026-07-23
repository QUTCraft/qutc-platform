# CHANGELOG

All notable changes to the QUTCraft Platform will be documented in this file.

## [v0.5.1] - 2026-07-23

### 🛠️ 后台成员管理与 Element Plus 按钮边缘样式修饰 (Admin Member Management Button Edge Fix)
- **修复文字按钮与胶囊按钮的边框轮廓问题 (`main.css`)**:
  - 为 `.el-button.is-text` / `.el-button--text` 强制设置 `border: 0 !important; border-color: transparent !important; box-shadow: none !important;`，彻底消除表格内文字按钮四周出现的暗色方框 outline。
  - 为 `.el-button.is-round` 重新设置平滑圆角 `border-radius: var(--md-shape-full)` 与 `border-color: transparent`，使得像 “+ 邀请成员” 胶囊按钮具备完美的圆润切边。
- **优化表格操作列布局 (`AdminUsersView.vue`)**:
  - 将操作列宽度微调为 `110px` 并添加 `align="right"` 与 `size="small"` 属性，使“编辑”按钮靠右优雅对齐。

---

## [v0.5.0] - 2026-07-23

### 🌟 门户首页炫酷拉风动画全面升级 (Public Portal Dynamic Animation Overhaul)
- **极光弥散网格与星尘流动粒子供应 (`HomeView.vue` & `main.css`)**:
  - 引入 14 颗纵向缓缓浮游升空的星尘微粒 (`.stardust-pixel`)，配以 16 秒缓速交错漂移的双色彩色 Monet 极光网格光斑 (`.mesh-blob`)。
- **Hero 标语炫彩渐变流动标题 (`.hero-gradient-title`)**:
  - 给 Hero 主标语添加了 `linear-gradient` 色彩渐变与 8 秒 `background-position` 平滑平移动画，随光影呼吸自然变幻。
- **主 CTA 按钮扫光光泽特效 (`.hero-apply-btn`)**:
  - 给“申请加入服务器”主按钮加入了斜向 25 度的 4 秒扫光高光 (`::after` 扫光动画)，极大增强点击引导力与科技质感。
- **Live Server Status 3D 浮跃与玻璃光芒悬浮 (`.hero-status`)**:
  - Hover 悬浮时呈 3D 抬升 (`translateY(-6px) scale(1.015)`)，自带 Monet 霓虹发光 Shadow 与高亮玻璃径向光效 (`::after`)。
- **三大 Core Pillars 支柱卡片 Hover 3D 旋转与发光 Icon (`.pillar-card`)**:
  - Pillar 卡片悬浮时向上抬升 `8px`，图标盒子 (`.pillar-icon-box`) 伴随 `rotate(6deg) scale(1.12)` 倾斜旋转与霓虹外发光。
