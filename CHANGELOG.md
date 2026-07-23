# CHANGELOG

All notable changes to the QUTCraft Platform will be documented in this file.

## [v0.4.3] - 2026-07-23

### 🎨 移除打勾光晕，替换为纯净几何星轨动画 (Constellation Orbit Animation)
- **星轨与 4 极闪烁星点 (`ApplyView.vue` & `main.css`)**:
  - 彻底移除打勾图标周围的绿色模糊扩散光晕。
  - 替换为极具几何精细感与科技韵味的 `.constellation-orbit` 360 度旋转虚线星轨，外围环绕 4 颗不同主题色（青绿/星紫/翡翠/月白）的微型发光星点 (`.orbit-dot`)。

---

## [v0.4.2] - 2026-07-23

### 🐛 修复提交成功打勾图标光晕动画对齐与边缘切割问题
- 精准居中与多阶气场光圈 (`ApplyView.vue` & `main.css`)。

---

## [v0.4.1] - 2026-07-23

### ✨ “踏星汉而至，赴方块之约” 提交动画影院级重构
- 引入 16 颗动态升空的 3D 悬浮方块粒子 (`.ascension-cubes`)。
- 优化了 850ms 节奏控制与全息认证存根卡片 (`.passport-card`)。
