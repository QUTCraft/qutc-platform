# QUTCraft Platform

Repository scaffold for the QUTCraft platform.

The current layout follows the project structure proposal:

```text
apps/
├── api/
│   ├── cmd/server/
│   ├── internal/
│   │   ├── config/
│   │   ├── handler/
│   │   ├── middleware/
│   │   ├── model/
│   │   ├── platform/{cache,database}/
│   │   ├── repository/
│   │   └── service/
│   └── migrations/
└── web/src/
    ├── api/
    ├── assets/
    ├── components/
    ├── layouts/
    ├── router/
    ├── stores/
    ├── styles/
    ├── types/
    ├── utils/
    └── views/
deploy/{compose,openresty}/
docs/{adr,api,architecture,product}/
scripts/
tests/integration/
```

## 当前可运行模块

- `apps/web`：Vue 3 + Vite + TypeScript + Element Plus 的 MD3 公共门户。
- `docs/api/openapi.yaml`：公开门户的 OpenAPI 3.1 契约，可直接导入 Apifox 或 Swagger 工具。

## 本地启动前端

```bash
cd apps/web
pnpm install
pnpm dev
```

默认使用契约 Mock 数据。复制 `.env.example` 为 `.env.local`，并设置 `VITE_API_MODE=remote`、`VITE_API_BASE_URL` 后，即可接入后端。
