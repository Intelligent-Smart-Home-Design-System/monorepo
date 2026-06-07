# main-pipeline

Temporal workflow-orchestrator for floor plan processing:

1. `floor-parser` activity `parse_floor_json`
2. `layout` activity `place_devices`
3. `device-selection` activity `select_devices`

Workers start once with Docker Compose and continuously poll Temporal task queues. A concrete pipeline starts only by `POST /start` through `api-gateway`, runs the three activities, then completes.

## Run

```bash
cd services/main-pipeline
docker compose up --build
```

Open:

- API Gateway (nginx -> Go gateway): `http://localhost:8090`
- Frontend API: `http://localhost:8090/api/v1`
- Temporal UI: `http://localhost:8088`
- Prometheus: `http://localhost:9092`
- Jaeger: `http://localhost:16686`
- Grafana: `http://localhost:3000` (`admin` / `admin`)


Optional (env):
В docker-compose.yml у каждого сервиса в env прописаны порты для метрик в METRICS_PORT. (Main-pipeline - 2112, api-gateway - 2116, floor-parser-worker - 2113, layout-worker - 2114, device-selection-worker - 2115)

В Grafana по умолчанию user:admin, pasw:admin. (Лучше поменять)


Метрики, логи и трейсы.

Для промсотра логов заходим в dashboard, создаем новый, выбираем Loki в качестве source. Далее в select label выбираем откуда мы хотим брать логи (из контейнера, всего сервиса).

Для метрик есть опять же Grafana или http://localhost:3100/metrics.

Трейсы тоже есть в Grafana ила напрямую http://localhost:16686

## Test examples

Register a user:

```bash
curl -X POST http://localhost:8090/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"demo-password"}'
```

Login and copy `access_token` from the response:

```bash
curl -X POST http://localhost:8090/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"demo-password"}'
```

Start workflow with the JWT token:

```bash
curl -X POST http://localhost:8090/start \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  --data-binary @examples/security_basic.json

curl -X POST http://localhost:8090/start \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  --data-binary @examples/lighting_plus_security.json
```

`nginx` routes public HTTP requests to `api-gateway`. `api-gateway` accepts the request, validates required fields, checks `Authorization: Bearer <JWT>`, and starts `MainPipelineWorkflow` via Temporal client.

Optional password reset flow for local/dev usage:

```bash
curl -X POST http://localhost:8090/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com"}'

curl -X POST http://localhost:8090/auth/reset-password \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","reset_token":"<reset_token>","new_password":"new-demo-password"}'
```

Users and password reset tokens are stored in `catalog-postgresql` through the `api_users` table, so registrations survive `api-gateway` restarts.

Frontend contract endpoints are routed through nginx to `frontend-api`:

```bash
curl http://localhost:8090/api/v1/device-types
curl http://localhost:8090/api/v1/ecosystems
curl http://localhost:8090/api/v1/presets
curl http://localhost:8090/api/v1/plans
```

To fetch a completed workflow result:

```bash
curl -X GET http://localhost:8090/result/main-pipeline-lighting-plus-security \
  -H "Authorization: Bearer <access_token>"

curl -X GET http://localhost:8090/result/{workflow-id} \
  -H "Authorization: Bearer <access_token>"

```

Or with query parameters:

```bash
curl -X GET "http://localhost:8090/result?workflow_id=main-pipeline-lighting-plus-security" \
  -H "Authorization: Bearer <access_token>"
```

If the workflow is still running or failed, the endpoint returns the current Temporal status instead of the final JSON.

Watch workflow status in Temporal UI. Logs are emitted to container stdout, metrics are exposed on ports `2112`-`2116`, and Jaeger is available for OTLP traces.

## Potential Bottlenecks

1. `device-selection-worker` is the most likely bottleneck: it loads the catalog from PostgreSQL and runs the solver. See `services/device-selection/src/device_selection/temporal/activities.py`.
2. `catalog-postgresql` can become a shared pressure point when many device-selection workers refresh catalog cache at once.
3. `layout-worker` can become CPU-bound on large floor plans or complex rule sets. See `services/layout/internal/temporalworker/activities.go`.
4. Each activity type currently uses one task queue (`floor-parser`, `layout`, `device-selection`), so there is no priority split between light and heavy requests. See `services/main-pipeline/workflows/main_pipeline.go`.
5. `GET /result` polling can add load to `api-gateway` and Temporal if many clients poll frequently. See `services/main-pipeline/cmd/api-gateway/main.go`.
6. Final JSON is stored in Temporal workflow history, which is not ideal for large or long-lived payloads. For production, store large results in DB/S3/MinIO and keep only a reference in workflow result.
7. `api-gateway` currently has JWT auth and validation, but no rate limiting or queue-load based backpressure.
8. Docker Compose does not define replicas or CPU/RAM limits; local scaling is done manually with `docker compose up --scale ...`.
