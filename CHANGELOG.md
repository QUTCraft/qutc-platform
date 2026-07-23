# CHANGELOG

All notable changes to the QUTCraft Platform will be documented in this file.

## [v0.4.2] - 2026-07-23

### 🐛 修复提交成功打勾图标光晕动画对齐与边缘切割问题 (Green Halo Animation Fix)
- **精准居中与多阶气场光圈 (`ApplyView.vue` & `main.css`)**:
  - 彻底修复了原先 `.icon-ring-pulse` 尺寸不匹配（`90px` vs `64px`）导致的绿色光晕不居中、边界截断与放慢失真 Bug。
  - 重构为基准 `64px` 居中双阶发光气场光环 (`.ring-1` & `.ring-2`)，结合 2.2 秒交错缓动动画，由圆心向外柔和扩散至 2.2 倍，并带有翡翠绿弥散晕染。

---

## [v0.4.1] - 2026-07-23

### ✨ “踏星汉而至，赴方块之约” 提交动画影院级重构
- 引入 16 颗动态升空的 3D 悬浮方块粒子 (`.ascension-cubes`)。
- 优化了 850ms 节奏控制与全息认证存根卡片 (`.passport-card`)。

---

## [v0.4.0] - 2026-07-23

### ✨ 公开门户首页视觉与交互全面翻新
- 引入 `hero-glow-backdrop` 动态背景弥散光晕与 `.hero-brand-pill` 官方运行徽章。
