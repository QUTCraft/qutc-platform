# CHANGELOG

All notable changes to the QUTCraft Platform will be documented in this file.

## [v0.2.1] - 2026-07-22

### 🐛 修复 ADMIN 管理工作台 Monet 配色遗漏与按钮文字重合 Readability 问题 (Urgent Admin Monet Readability Fix)
- **管理工作台全面参与 Monet 取色 (Admin Area Full Monet Sync)**:
  - 修复了 Admin Rail 侧边栏、TopBar 顶部栏、Admin Panel、Cards、Forms、Empty States、Tags、Code 块与表格未完整参与 Monet 动态主题色彩覆盖的问题。
  - 确保所有管理工作台区域完全响应 MD3 Monet 动态四季与昼夜调色盘。
- **重构按钮文字与背景色重合 bug (Element Plus Button Readability Fix)**:
  - 彻底解除了 Element Plus 默认 `is-disabled` / `plain` / `text` 按钮在暗色模式或 Monet 盛夏/春樱/枫秋色彩下出现的文字与背景色沉底重合 bug。
  - 主按钮 (`.el-button--primary`) 的 span 文字与图标固定映射至高对比度的 `--md-sys-color-on-primary`，禁用态与默认态均具备 100% 极高可读性。

---

## [v0.2.0] - 2026-07-22

### 🎨 Material Design 3 Monet 动态四季与昼夜色彩引擎 (Seasonal & Diurnal Monet Color Engine)
- **严格基于本地时区与系统时间自动推演 (Strict Local Auto-Detection)**:
  - 移除前端手动切换胶囊，全站主题调色完全基于本地时间（`new Date()`）自动推演。
  - **季节感应 (Month-based Seasons)**:
    - 🌸 **春 · 樱纷复苏 (3-5月)**: 樱粉与新芽柔绿，搭配暖桃基调 (`#D81B60` / `#2E7D32`)。
    - 🌿 **夏 · 碧海潮生 (6-8月)**: 碧海青绿与海浪湛蓝，搭配冰霜青雾 (`#00897B` / `#0288D1`)。
    - 🍁 **秋 · 枫丹金秋 (9-11月)**: 赭石枫红与麦浪琥珀，搭配暖金麦香 (`#D84315` / `#F57F17`)。
    - ❄️ **冬 · 霜雪微芒 (12-2月)**: 霜雪冰蓝与夜幕紫罗兰，搭配冰银月白 (`#1E88E5` / `#5E35B1`)。
  - **昼夜感应 (Hour-based Day & Night)**:
    - ☀️ **日光白天 (07:00 - 18:59)**: 明亮清爽、高饱和度 MD3 日间光感。
    - 🌙 **静谧夜幕 (19:00 - 06:59)**: 优雅柔和的深色高对比度夜间模式。
- **暗色高对比度文字与极致可读性 (High-Contrast Dark Readability)**:
  - 重构了暗色模式下的文字 contrast ratio 至 `12:1` 以上，解决死板硬切与暗光下文字模糊刺眼的问题。
- **全站 450ms 平滑缓动过渡 (Silky Transitions)**:
  - `transition: background-color 450ms cubic-bezier(0.2, 0, 0, 1), color 450ms cubic-bezier(0.2, 0, 0, 1)`，昼夜交替与主题色彩切换时极为平滑自然。

---

### 🚀 二级申请页面与全屏宇宙入场动画 (Apply View & Page Transitions)
- **按钮中心原点黑色扩散跳转 (`usePageTransition.ts`)**:
  - “加入我们” / “申请加入服务器” 按钮点击时，以鼠标点击坐标为原点向外扩散圆形黑幕覆盖全屏（计算至最远屏幕角，动态放大），动画放完后方切换路由至 `/apply`，实现零闪烁平滑过渡。
- **渐进出现繁星与沉浸背景 (`ApplyView.vue`)**:
  - 进入申请页面后，60 颗高密度繁星按时间差 (`delay: i * 24ms`) 一颗接一颗闪烁诞生，并永久留存于深空背景中。
  - 两个核心点光晕 (`.nebula-glow`) 从点光源 (`scale(0.01)`) 历经 2.2 秒缓缓向外扩散为 600px 的弥散星云。
- **提交表单后的色彩爆发动画 (Post-Submit Bloom)**:
  - 极度震撼的提交成功动画：黑白星空表单瞬间绽放 3D 冲击波环、旋转光束与全彩超新星爆发辉光。

---

### 🏛️ MD3 统一设计系统与整体视觉提升 (MD3 Design System Overhaul)
- 引入规范的 MD3 色阶、Shape 圆角级 (8px/12px/16px/28px/Full)、Elevation 阴影与 Typography 文字排版层级。
- Element Plus 组件全局与 MD3 Design Tokens (`tokens.css`) 深度绑定映射。
