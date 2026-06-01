# frontend-api

Read-only-ish HTTP API for the frontend contract.

It serves:

- `GET /api/v1/device-types`
- `GET /api/v1/ecosystems`
- `GET /api/v1/presets`
- `GET /api/v1/plans`
- `POST /api/v1/plans`
- `GET /api/v1/plans/{plan_id}`
- `GET /api/v1/plans/{plan_id}/status`

Data is stored in `catalog-postgresql`. Metadata and plan tables are created by `db/catalog/migrations/000002_frontend_api.*.sql`.
