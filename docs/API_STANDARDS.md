# QUTCraft Platform - API 规范与架构指南 (API Standards & Architecture Guide)

本文档归档了 QUTCraft 平台的后端 API 接口设计规范、认证流程以及服务器适配器 (RCON/SMTP) 规范。

---

## 1. 认证与 RBAC 权限规范 (Auth & Authorization)

### 1.1 认证接口 (Authentication API)
- **基准路径**: `/api/v1/auth`
- **机制**: 基于 JWT (JSON Web Tokens) 双令牌机制 (AccessToken + RefreshToken)。
- **客户端存储**: AccessToken 在内存或 Cookie 中保存，安全等级遵照 OAuth 2.0 规范。

### 1.2 RBAC 角色与权限级别 (Roles)
| 角色 (Role) | 识别码 | 说明 |
| :--- | :--- | :--- |
| **所有者** | `owner` | 包含全部系统配置、管理员账号管理及服务器管理权 |
| **管理员** | `administrator` | 白名单审核、内容审批、限制性 RCON 命令下发 |
| **编辑者** | `editor` | 新增与修改动态、资源及知识库草稿 |
| **成员** | `member` | 平台基础协作与内部讨论 |

---

## 2. 公开门户与申请 API (Portal API)

### 2.1 申请提交接口 (Whitelist & Member Apply)
- **路由**: `POST /api/v1/portal/apply`
- **Payload**:
  ```json
  {
    "type": "whitelist",
    "name": "张三",
    "student_id": "20240101",
    "game_id": "Steve_QUT",
    "qq": "12345678",
    "email": "steve@qutcraft.com",
    "note": "申请加入服务器建构"
  }
  ```

---

## 3. SMTP 邮箱通知与队列集成 (Email Notification Adapter)

- **服务机制**: 当玩家提交 `/api/v1/portal/apply` 成功后，后端消息队列 (Queue Consumer) 触发 SMTP 发件程序。
- **转发凭证**: SMTP 授权码与服务器私钥严格存储于受控环境变量 (`SMTP_AUTH_CODE`)，禁止向前端暴露。
- **模板样式**: 包含玩家游戏 ID、学号、QQ 账号及快速审核链接。

---

## 4. Minecraft 服务器适配器 (Server RCON Adapter)

- **隔离原则**: RCON 端口 (`25575`) 与受限管理接口仅在内部私有网络中打通。
- **审计日志 (Audit Logging)**: 每一条由管理员在后台发送的 `RCON` 指令 (如 `list`, `whitelist add`) 均产生一条包含 `operator_id` 与 `timestamp` 的审计记录。
