# frontend-api

Read-only-ish HTTP API for the frontend contract.

It serves:

- `GET /api/v1/device-types`
- `GET /api/v1/ecosystems`
- `GET /api/v1/presets`
- `GET /api/v1/plans`
- `POST /api/v1/plans`
- `POST /api/v1/plans/manual`
- `GET /api/v1/plans/{plan_id}`
- `GET /api/v1/plans/{plan_id}/status`

`POST /api/v1/plans/manual` accepts a budget, a parsed `floor_plan`, and concrete
catalog selections:

```json
{
  "budget": 50000,
  "floor_plan": {},
  "selections": [
    {
      "device_id": 1,
      "listing_id": 10,
      "quantity": 2
    }
  ]
}
```

The API derives each selection's device type and current price from the catalog.
It rejects unavailable listings, duplicate device types, and selections whose
current total cost exceeds the budget.

Data is stored in `catalog-postgresql`. Metadata and plan tables are created by `db/catalog/migrations/000002_frontend_api.*.sql`.
