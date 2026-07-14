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

This README is intentionally a placeholder and can be expanded as implementation begins.
