# CHANGELOG

All notable changes to the QUTCraft Platform will be documented in this file.

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

---

## [v0.4.5] - 2026-07-23

### 🎨 全局禁用误触选中文本与高发闪烁白条
- 给申请页面背景全景容器 `.starlight-wrapper` 增加了全局 `user-select: none`。
