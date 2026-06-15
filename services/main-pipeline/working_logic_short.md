# Краткая логика работы `main-pipeline`

1. Клиент отправляет JSON-запрос в `api-gateway` через `POST /start`: `services/main-pipeline/README.md:37`.
2. В запросе передаётся auth header `Authorization: Bearer dev-token`: `services/main-pipeline/README.md:39`.
3. В теле запроса передаётся JSON с планом, выбранными уровнями и параметрами подбора устройств: `services/main-pipeline/README.md:40`.
4. `api-gateway` проверяет авторизацию запроса: `services/main-pipeline/cmd/api-gateway/main.go:73`.
5. `api-gateway` декодирует тело запроса в `PipelineRequest`: `services/main-pipeline/cmd/api-gateway/main.go:79`.
6. `PipelineRequest` содержит `request_id`: `services/main-pipeline/internal/pipeline/types.go:6`.
7. `PipelineRequest` содержит входной `floor_plan`: `services/main-pipeline/internal/pipeline/types.go:7`.
8. `PipelineRequest` содержит выбранные уровни сервисов в `selected_levels`: `services/main-pipeline/internal/pipeline/types.go:8`.
9. `PipelineRequest` содержит параметры подбора устройств в `device_selection`: `services/main-pipeline/internal/pipeline/types.go:9`.
10. `api-gateway` проверяет, что в запросе есть `floor_plan`: `services/main-pipeline/cmd/api-gateway/main.go:161`.
11. `api-gateway` проверяет, что в запросе есть `selected_levels`: `services/main-pipeline/cmd/api-gateway/main.go:164`.
12. `api-gateway` проверяет, что в запросе есть `device_selection`: `services/main-pipeline/cmd/api-gateway/main.go:167`.
13. Если `request_id` не передан, `api-gateway` генерирует его автоматически: `services/main-pipeline/cmd/api-gateway/main.go:92`.
14. `api-gateway` запускает Temporal workflow `MainPipelineWorkflow`: `services/main-pipeline/cmd/api-gateway/main.go:96`.
15. Workflow получает ID вида `main-pipeline-<request_id>`: `services/main-pipeline/cmd/api-gateway/main.go:97`.
16. Workflow отправляется в task queue `main-pipeline`: `services/main-pipeline/cmd/api-gateway/main.go:98`.
17. Клиент сразу получает `workflow_id` и `run_id`: `services/main-pipeline/cmd/api-gateway/main.go:110`.
18. `main-pipeline` worker забирает workflow из task queue `main-pipeline`: `services/main-pipeline/cmd/main-pipeline/main.go:41`.
19. Начинается выполнение `MainPipelineWorkflow`: `services/main-pipeline/workflows/main_pipeline.go:21`.
20. Workflow вызывает первый шаг `parse_floor_json`: `services/main-pipeline/workflows/main_pipeline.go:38`.
21. В `parse_floor_json` передаётся исходный `floor_plan`: `services/main-pipeline/workflows/main_pipeline.go:40`.
22. После успешного парсинга workflow получает `parsed.FloorPlan`: `services/main-pipeline/workflows/main_pipeline.go:41`.
23. Workflow вызывает второй шаг `place_devices`: `services/main-pipeline/workflows/main_pipeline.go:51`.
24. В `place_devices` передаётся результат `floor-parser`: `services/main-pipeline/workflows/main_pipeline.go:53`.
25. В `place_devices` передаётся `selected_levels`: `services/main-pipeline/workflows/main_pipeline.go:54`.
26. После успешной расстановки workflow получает `placed.Layout`: `services/main-pipeline/workflows/main_pipeline.go:55`.
27. Workflow вызывает третий шаг `select_devices_json`: `services/main-pipeline/workflows/main_pipeline.go:65`.
28. В `select_devices_json` передаётся блок `device_selection`: `services/main-pipeline/workflows/main_pipeline.go:66`.
29. В `select_devices_json` передаётся результат `layout`: `services/main-pipeline/workflows/main_pipeline.go:67`.
30. После успешного подбора workflow получает `selected.Result`: `services/main-pipeline/workflows/main_pipeline.go:68`.
31. Workflow собирает итоговый `PipelineResult`: `services/main-pipeline/workflows/main_pipeline.go:73`.
32. В итоговый результат входит `request_id`: `services/main-pipeline/workflows/main_pipeline.go:74`.
33. В итоговый результат входит `parsed_floor_plan`: `services/main-pipeline/workflows/main_pipeline.go:75`.
34. В итоговый результат входит `layout`: `services/main-pipeline/workflows/main_pipeline.go:76`.
35. В итоговый результат входит `device_selection`: `services/main-pipeline/workflows/main_pipeline.go:77`.
36. Итоговая структура результата описана как `PipelineResult`: `services/main-pipeline/internal/pipeline/types.go:40`.
37. Клиент может запросить результат через `GET /result/{workflow_id}`: `services/main-pipeline/README.md:53`.
38. Клиент может запросить результат через `GET /result?workflow_id=...`: `services/main-pipeline/README.md:60`.
39. `api-gateway` проверяет статус workflow через Temporal: `services/main-pipeline/cmd/api-gateway/main.go:128`.
40. Если workflow ещё не завершён, `api-gateway` возвращает текущий статус: `services/main-pipeline/cmd/api-gateway/main.go:135`.
41. Если workflow завершён, `api-gateway` получает финальный результат: `services/main-pipeline/cmd/api-gateway/main.go:147`.
42. `api-gateway` возвращает финальный JSON клиенту: `services/main-pipeline/cmd/api-gateway/main.go:153`.
