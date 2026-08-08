# frontend-api

Read-only-ish HTTP API for the frontend contract.

It serves:

- `GET /api/v1/device-types`
- `GET /api/v1/ecosystems`
- `GET /api/v1/presets`
- `GET /api/v1/catalog/categories`
- `GET /api/v1/catalog/products`
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

With the Compose `test` profile, `catalog-db-seed` loads the manual-selection
catalog from `services/frontend-api/test.sql`. Manual catalog endpoints and plan
validation use the rows marked with `taxonomy_version = 'test'`.

Data is stored in `catalog-postgresql`. Metadata and plan tables are created by `db/catalog/migrations/000002_frontend_api.*.sql`.
