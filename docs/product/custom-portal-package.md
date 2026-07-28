# 自定义门户包接入指南

> 状态：Portal Manifest v1 配套规范  
> 当前范围：同源静态门户包；不提供在线代码编辑、第三方后端插件或第一方主题

## 1. 最小目录

```text
apps/web/public/portals/campus-club/
├── index.html
├── app.js
├── styles.css
└── theme.json
```

生产构建后这些文件必须保持在 `/portals/campus-club/`。Nginx 对不存在的 `/portals/` 资源返回真实 `404`，不会使用 Vue SPA 首页代替。

入口 HTML 必须声明与 Manifest 一致的门户 ID：

```html
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="qutc-portal-id" content="campus-club">
  <title>Campus Club Portal</title>
  <link rel="stylesheet" href="/portals/campus-club/styles.css">
</head>
<body>
  <main id="app"></main>
  <script type="module" src="/portals/campus-club/app.js"></script>
</body>
</html>
```

运行时会在切换前获取入口并检查：

- HTTP 状态为成功。
- `Content-Type` 包含 `text/html`。
- `qutc-portal-id` 与生效 Manifest 的 `id` 完全一致。
- 配置读取和入口探测分别不超过 1.8 秒。

任一条件失败都会保留默认 MD3，不会进入空白页。

## 2. Manifest

使用 [通用 Manifest 示例](examples/custom-portal.portal.json) 和 [主题 Token 示例](examples/custom-portal.theme.json)。Manifest 通过 Admin 设置页导入、保存为草稿，再单独启用。

入口和 Token 只能使用同源绝对路径。Manifest 不能声明外部 URL、查询参数、目录穿越、管理权限或 `server.command`。

## 3. 数据访问

自定义门户只能消费 `/api/v1/portal/organizations/{slug}` 下的公开接口。禁止：

- 携带或保存 Admin JWT、Refresh Token。
- 调用 `/api/v1/admin/*`、成员、审批、审计或命令接口。
- 拼接数据库、Redis、MinIO、RCON 或 SMTP 地址。
- 假设草稿、成员邮箱、内部项目成员等字段存在。

页面至少处理加载、空数据、`404`、网络失败和字段为 `null`。下载地址必须直接使用 Portal API 返回值。

## 4. CSP 与资源规则

Web 默认安全策略仅允许同源脚本、样式、图片、字体和连接；开发 Compose 额外允许本机 API 端口。门户包不得依赖任意远程脚本、远程字体或动态执行字符串。

- JavaScript 使用同源 ES Module 文件。
- 图片使用同源资源、Portal 下载地址或受控 `data:`/`blob:`。
- 禁止 `eval`、内联脚本、对象嵌入和跨站表单提交。
- 生产部署推荐将 API 反向代理为同源 `/api/`，避免扩大 `connect-src`。

## 5. 发布与恢复

1. 将门户包部署到 Manifest 声明的同源目录。
2. 直接访问入口，确认返回 HTML 而不是 SPA fallback。
3. 在 Admin 导入 Manifest 并保存草稿。
4. 使用“预览入口”检查页面。
5. 点击“启用此草稿”后再访问公开首页。
6. 失败时用 `/?portal=md3` 临时强制 MD3。
7. 需要永久恢复时，在 Admin 点击“恢复默认 MD3”；该操作保存并启用内置 Manifest，并留下既有配置审计。

浏览器将最近一次自动回退以不含隐私的结构化记录保存到当前标签会话，并输出 `qutc.portal.runtime_fallback` 警告。数据库 Manifest 损坏时 API 也会记录不含原文和密钥的服务端回退日志。

## 6. 发布检查表

- [ ] 入口包含正确的 `qutc-portal-id`。
- [ ] Manifest、入口、Token 与静态目录中的 ID 和版本一致。
- [ ] 只请求声明的 Portal API 能力。
- [ ] 无 Admin Token、服务端密钥、远程脚本和内联脚本。
- [ ] 入口不存在时返回真实 `404`。
- [ ] API、入口超时和字段缺失时仍可阅读回退门户。
- [ ] `/admin`、认证、邀请和申请页面不受门户切换影响。
- [ ] 临时强制回退和永久恢复默认 MD3 均已演练。
