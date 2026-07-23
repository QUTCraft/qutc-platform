# CHANGELOG

All notable changes to the QUTCraft Platform will be documented in this file.

## [v0.4.5] - 2026-07-23

### 🎨 全局禁用误触选中文本与高发闪烁白条 (Text Selection Flickering Fix)
- **全面屏蔽背景文本选中框 (`main.css`)**:
  - 给申请页面背景全景容器 `.starlight-wrapper` 增加了全局 `user-select: none; -webkit-tap-highlight-color: transparent;`。
  - 彻底解决了在星空背景与升空粒子动画播放过程中，用户点击或拖拽误选背景空文本导致的白色发光高亮选中条闪烁 Bug。
  - 仅保留表单输入框内正常的文本光标输入体验，并将全局文本选中样式定调为相配的半透明 Monet 紫调 (`rgba(195, 192, 255, 0.35)`).

---

## [v0.4.4] - 2026-07-23

### 🎯 修复打勾图标与外围星轨绝对同心对齐
- 重构盒模型，将中心点锁定在 `(40px, 40px)` 绝对同心圆点。

---

## [v0.4.3] - 2026-07-23

### 🎨 移除打勾光晕，替换为纯净几何星轨动画
- 彻底移除打勾图标周围的绿色模糊扩散光晕，替换为旋转虚线星轨与 4 极发光星点。
