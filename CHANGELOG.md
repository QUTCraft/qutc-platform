# CHANGELOG

All notable changes to the QUTCraft Platform will be documented in this file.

## [v0.4.4] - 2026-07-23

### 🎯 修复打勾图标与外围星轨绝对同心对齐 (100% Strict Concentric Alignment)
- **精准像素计算与同心定位 (`ApplyView.vue` & `main.css`)**:
  - 重构了 `.success-icon-wrapper` 盒模型架构（`84px * 84px`），将绿色打勾徽章 (`.check-badge`) 绝对定位为 `top: 10px; left: 10px;`（`64px * 64px`）。
  - 将 `.constellation-orbit` 星轨绝对定位为 `inset: 0`（`84px * 84px`），两者的几何中心点均精准锁定在 `(40px, 40px)` 绝对同心圆点，彻底消除了旋转与渲染时的任何偏心或晃动感。

---

## [v0.4.3] - 2026-07-23

### 🎨 移除打勾光晕，替换为纯净几何星轨动画
- 彻底移除打勾图标周围的绿色模糊扩散光晕，替换为旋转虚线星轨与 4 极发光星点。

---

## [v0.4.2] - 2026-07-23

### 🐛 修复提交成功打勾图标光晕动画对齐与边缘切割问题
- 精准居中与多阶气场光圈 (`ApplyView.vue` & `main.css`)。
