# Подробная логика работы `main-pipeline`

1. Пользователь запускает систему командой `docker compose up --build`, это описано в инструкции запуска: `services/main-pipeline/README.md:15`.
2. Docker Compose читает список сервисов из секции `services`: `services/main-pipeline/docker-compose.yml:1`.
3. Для Temporal поднимается PostgreSQL-хранилище `temporal-postgresql`: `services/main-pipeline/docker-compose.yml:2`.
4. Temporal Server стартует как сервис `temporal` и ждёт готовности `temporal-postgresql`: `services/main-pipeline/docker-compose.yml:16`.
5. Temporal Server публикует порт `7233`, через который к нему подключаются worker и gateway: `services/main-pipeline/docker-compose.yml:27`.
6. Отдельно стартует `temporal-ui`, чтобы можно было смотреть workflow execution в браузере: `services/main-pipeline/docker-compose.yml:30`.
7. Для каталога устройств стартует `catalog-postgresql`: `services/main-pipeline/docker-compose.yml:39`.
8. Одноразовый job `catalog-db-migrate` применяет SQL-миграции каталога: `services/main-pipeline/docker-compose.yml:55`.
9. Одноразовый job `catalog-db-seed` заполняет тестовый каталог устройств seed-данными: `services/main-pipeline/docker-compose.yml:65`.
10. Docker собирает образ `main-pipeline` из текущей директории: `services/main-pipeline/docker-compose.yml:77`.
11. В Dockerfile сначала собирается бинарник workflow-worker `main-pipeline`: `services/main-pipeline/Dockerfile:6`.
12. В том же Dockerfile собирается второй бинарник `api-gateway`: `services/main-pipeline/Dockerfile:7`.
13. В runtime-образ копируется бинарник `main-pipeline`: `services/main-pipeline/Dockerfile:11`.
14. В runtime-образ копируется бинарник `api-gateway`: `services/main-pipeline/Dockerfile:12`.
15. По умолчанию контейнер этого образа запускает entrypoint `main-pipeline`: `services/main-pipeline/Dockerfile:14`.
16. Сервис `main-pipeline` в compose зависит от Temporal и трёх activity-worker’ов: `services/main-pipeline/docker-compose.yml:80`.
17. Для `main-pipeline` задаётся адрес Temporal `temporal:7233`: `services/main-pipeline/docker-compose.yml:86`.
18. Для `main-pipeline` задаётся namespace Temporal `default`: `services/main-pipeline/docker-compose.yml:87`.
19. Процесс `main-pipeline` стартует с функции `main`: `services/main-pipeline/cmd/main-pipeline/main.go:19`.
20. `main-pipeline` читает адрес Temporal из `TEMPORAL_ADDRESS`: `services/main-pipeline/cmd/main-pipeline/main.go:24`.
21. `main-pipeline` читает namespace из `TEMPORAL_NAMESPACE`: `services/main-pipeline/cmd/main-pipeline/main.go:25`.
22. `main-pipeline` создаёт Temporal client через `client.Dial`: `services/main-pipeline/cmd/main-pipeline/main.go:31`.
23. `main-pipeline` создаёт Temporal workflow-worker на task queue `main-pipeline`: `services/main-pipeline/cmd/main-pipeline/main.go:41`.
24. `main-pipeline` регистрирует workflow `MainPipelineWorkflow`: `services/main-pipeline/cmd/main-pipeline/main.go:42`.
25. `main-pipeline` запускает worker polling через `workflowWorker.Run(...)`: `services/main-pipeline/cmd/main-pipeline/main.go:44`.
26. С этого момента `main-pipeline` не выполняет pipeline сам по себе, а постоянно poll’ит Temporal task queue: `services/main-pipeline/cmd/main-pipeline/main.go:41`.
27. Параллельно compose запускает `api-gateway` как отдельный сервис: `services/main-pipeline/docker-compose.yml:95`.
28. `api-gateway` использует тот же Docker image, но переопределяет entrypoint на `api-gateway`: `services/main-pipeline/docker-compose.yml:98`.
29. `api-gateway` зависит от Temporal и `main-pipeline`: `services/main-pipeline/docker-compose.yml:99`.
30. `api-gateway` получает адрес Temporal через `TEMPORAL_ADDRESS`: `services/main-pipeline/docker-compose.yml:103`.
31. `api-gateway` слушает HTTP внутри контейнера на `:8080`: `services/main-pipeline/docker-compose.yml:105`.
32. `api-gateway` получает auth token из `API_GATEWAY_TOKEN`, по умолчанию `dev-token`: `services/main-pipeline/docker-compose.yml:107`.
33. `api-gateway` публикуется наружу как `localhost:8090`: `services/main-pipeline/docker-compose.yml:110`.
34. Процесс `api-gateway` стартует с функции `main`: `services/main-pipeline/cmd/api-gateway/main.go:23`.
35. `api-gateway` создаёт Temporal client через `client.Dial`: `services/main-pipeline/cmd/api-gateway/main.go:28`.
36. `api-gateway` создаёт HTTP server на адресе из `HTTP_ADDRESS`: `services/main-pipeline/cmd/api-gateway/main.go:45`.
37. В HTTP server передаётся router, собранный функцией `buildAPI`: `services/main-pipeline/cmd/api-gateway/main.go:48`.
38. Router регистрирует healthcheck `GET /healthz`: `services/main-pipeline/cmd/api-gateway/main.go:68`.
39. Router регистрирует основную ручку запуска `POST /start`: `services/main-pipeline/cmd/api-gateway/main.go:72`.
40. Пользователь отправляет JSON через `POST http://localhost:8090/start`: `services/main-pipeline/README.md:37`.
41. Пользователь передаёт `Content-Type: application/json`: `services/main-pipeline/README.md:38`.
42. Пользователь передаёт auth header `Authorization: Bearer dev-token`: `services/main-pipeline/README.md:39`.
43. Пользователь передаёт тело запроса из JSON-файла через `--data-binary`: `services/main-pipeline/README.md:40`.
44. `api-gateway` сначала проверяет авторизацию для `/start`: `services/main-pipeline/cmd/api-gateway/main.go:73`.
45. Функция `authorized` разрешает запрос без token только если token не задан: `services/main-pipeline/cmd/api-gateway/main.go:173`.
46. Функция `authorized` принимает auth через header `X-API-Key`: `services/main-pipeline/cmd/api-gateway/main.go:177`.
47. Функция `authorized` также принимает auth через `Authorization: Bearer ...`: `services/main-pipeline/cmd/api-gateway/main.go:180`.
48. Если auth не прошёл, gateway возвращает `401 unauthorized`: `services/main-pipeline/cmd/api-gateway/main.go:75`.
49. Если auth прошёл, gateway создаёт переменную запроса типа `pipeline.PipelineRequest`: `services/main-pipeline/cmd/api-gateway/main.go:79`.
50. Gateway ограничивает размер body до `10 MiB`: `services/main-pipeline/cmd/api-gateway/main.go:80`.
51. Gateway запрещает неизвестные поля JSON через `DisallowUnknownFields`: `services/main-pipeline/cmd/api-gateway/main.go:81`.
52. Gateway декодирует JSON body в `PipelineRequest`: `services/main-pipeline/cmd/api-gateway/main.go:82`.
53. Структура входного запроса содержит `request_id`: `services/main-pipeline/internal/pipeline/types.go:6`.
54. Структура входного запроса содержит `floor_plan`: `services/main-pipeline/internal/pipeline/types.go:7`.
55. Структура входного запроса содержит `selected_levels`: `services/main-pipeline/internal/pipeline/types.go:8`.
56. Структура входного запроса содержит `device_selection`: `services/main-pipeline/internal/pipeline/types.go:9`.
57. Gateway вызывает бизнес-валидацию входного запроса через `validate(req)`: `services/main-pipeline/cmd/api-gateway/main.go:87`.
58. `validate` требует наличие `floor_plan`: `services/main-pipeline/cmd/api-gateway/main.go:161`.
59. `validate` требует наличие непустого `selected_levels`: `services/main-pipeline/cmd/api-gateway/main.go:164`.
60. `validate` требует наличие `device_selection`: `services/main-pipeline/cmd/api-gateway/main.go:167`.
61. Если `request_id` не передан, gateway генерирует его автоматически: `services/main-pipeline/cmd/api-gateway/main.go:92`.
62. Gateway стартует workflow через Temporal client методом `ExecuteWorkflow`: `services/main-pipeline/cmd/api-gateway/main.go:96`.
63. Workflow ID формируется как `main-pipeline-` + `request_id`: `services/main-pipeline/cmd/api-gateway/main.go:97`.
64. Workflow отправляется в task queue `main-pipeline`: `services/main-pipeline/cmd/api-gateway/main.go:98`.
65. В Temporal стартует workflow-функция `MainPipelineWorkflow`: `services/main-pipeline/cmd/api-gateway/main.go:99`.
66. Если Temporal не смог стартовать workflow, gateway возвращает `500`: `services/main-pipeline/cmd/api-gateway/main.go:102`.
67. Если workflow успешно стартовал, gateway выставляет JSON response header: `services/main-pipeline/cmd/api-gateway/main.go:108`.
68. Gateway возвращает HTTP `202 Accepted`: `services/main-pipeline/cmd/api-gateway/main.go:109`.
69. Gateway возвращает клиенту `workflow_id` и `run_id`: `services/main-pipeline/cmd/api-gateway/main.go:110`.
70. Temporal кладёт workflow task в task queue `main-pipeline`, имя queue задано константой: `services/main-pipeline/workflows/main_pipeline.go:12`.
71. Ранее запущенный worker `main-pipeline` забирает workflow task, потому что он poll’ит эту же queue: `services/main-pipeline/cmd/main-pipeline/main.go:41`.
72. Начинается выполнение `MainPipelineWorkflow`: `services/main-pipeline/workflows/main_pipeline.go:21`.
73. Workflow создаёт retry policy для activity: `services/main-pipeline/workflows/main_pipeline.go:25`.
74. Retry policy задаёт начальный интервал retry `1s`: `services/main-pipeline/workflows/main_pipeline.go:26`.
75. Retry policy задаёт максимум `3` попытки: `services/main-pipeline/workflows/main_pipeline.go:29`.
76. Workflow готовит переменную для результата `floor-parser`: `services/main-pipeline/workflows/main_pipeline.go:32`.
77. Workflow создаёт activity context для `floor-parser`: `services/main-pipeline/workflows/main_pipeline.go:33`.
78. Для `floor-parser` используется task queue `floor-parser`: `services/main-pipeline/workflows/main_pipeline.go:34`.
79. Для `floor-parser` задаётся timeout `2 minutes`: `services/main-pipeline/workflows/main_pipeline.go:35`.
80. Workflow вызывает activity `parse_floor_json`: `services/main-pipeline/workflows/main_pipeline.go:38`.
81. В `parse_floor_json` передаётся `request_id`: `services/main-pipeline/workflows/main_pipeline.go:39`.
82. В `parse_floor_json` передаётся исходный `floor_plan`: `services/main-pipeline/workflows/main_pipeline.go:40`.
83. Workflow ждёт завершения `parse_floor_json` через `.Get(...)`: `services/main-pipeline/workflows/main_pipeline.go:41`.
84. Если `parse_floor_json` завершился ошибкой после retry, workflow возвращает ошибку и падает: `services/main-pipeline/workflows/main_pipeline.go:42`.
85. Если `parse_floor_json` успешен, workflow готовит переменную результата `layout`: `services/main-pipeline/workflows/main_pipeline.go:45`.
86. Workflow создаёт activity context для `layout`: `services/main-pipeline/workflows/main_pipeline.go:46`.
87. Для `layout` используется task queue `layout`: `services/main-pipeline/workflows/main_pipeline.go:47`.
88. Для `layout` задаётся timeout `2 minutes`: `services/main-pipeline/workflows/main_pipeline.go:48`.
89. Workflow вызывает activity `place_devices`: `services/main-pipeline/workflows/main_pipeline.go:51`.
90. В `place_devices` передаётся тот же `request_id`: `services/main-pipeline/workflows/main_pipeline.go:52`.
91. В `place_devices` передаётся уже обработанный `floor_plan` из результата `floor-parser`: `services/main-pipeline/workflows/main_pipeline.go:53`.
92. В `place_devices` передаётся `selected_levels` из исходного запроса: `services/main-pipeline/workflows/main_pipeline.go:54`.
93. Workflow ждёт завершения `place_devices` через `.Get(...)`: `services/main-pipeline/workflows/main_pipeline.go:55`.
94. Если `place_devices` завершился ошибкой после retry, workflow возвращает ошибку и падает: `services/main-pipeline/workflows/main_pipeline.go:56`.
95. Если `place_devices` успешен, workflow готовит переменную результата `device-selection`: `services/main-pipeline/workflows/main_pipeline.go:59`.
96. Workflow создаёт activity context для `device-selection`: `services/main-pipeline/workflows/main_pipeline.go:60`.
97. Для `device-selection` используется task queue `device-selection`: `services/main-pipeline/workflows/main_pipeline.go:61`.
98. Для `device-selection` задаётся timeout `3 minutes`: `services/main-pipeline/workflows/main_pipeline.go:62`.
99. Workflow вызывает activity `select_devices_json`: `services/main-pipeline/workflows/main_pipeline.go:65`.
100. В `select_devices_json` передаётся исходный блок `device_selection`: `services/main-pipeline/workflows/main_pipeline.go:66`.
101. В `select_devices_json` передаётся результат расстановки `layout`: `services/main-pipeline/workflows/main_pipeline.go:67`.
102. Workflow ждёт завершения `select_devices_json` через `.Get(...)`: `services/main-pipeline/workflows/main_pipeline.go:68`.
103. Если `select_devices_json` завершился ошибкой после retry, workflow возвращает ошибку и падает: `services/main-pipeline/workflows/main_pipeline.go:69`.
104. Если все три activity успешны, workflow собирает финальный `PipelineResult`: `services/main-pipeline/workflows/main_pipeline.go:73`.
105. В финальный результат записывается `request_id`: `services/main-pipeline/workflows/main_pipeline.go:74`.
106. В финальный результат записывается `parsed_floor_plan`: `services/main-pipeline/workflows/main_pipeline.go:75`.
107. В финальный результат записывается `layout`: `services/main-pipeline/workflows/main_pipeline.go:76`.
108. В финальный результат записывается `device_selection`: `services/main-pipeline/workflows/main_pipeline.go:77`.
109. Workflow возвращает финальный результат в Temporal history: `services/main-pipeline/workflows/main_pipeline.go:73`.
110. Структура финального результата описана как `PipelineResult`: `services/main-pipeline/internal/pipeline/types.go:40`.
111. Поле финального результата `parsed_floor_plan` описано в типах pipeline: `services/main-pipeline/internal/pipeline/types.go:42`.
112. Поле финального результата `layout` описано в типах pipeline: `services/main-pipeline/internal/pipeline/types.go:43`.
113. Поле финального результата `device_selection` описано в типах pipeline: `services/main-pipeline/internal/pipeline/types.go:44`.
114. После завершения workflow worker `main-pipeline` не останавливается, а продолжает poll’ить новые workflow tasks: `services/main-pipeline/cmd/main-pipeline/main.go:44`.
115. Чтобы получить результат, пользователь вызывает `GET /result/{workflow_id}`: `services/main-pipeline/README.md:53`.
116. Альтернативно пользователь может вызвать `GET /result?workflow_id=...`: `services/main-pipeline/README.md:60`.
117. Gateway регистрирует handler для `GET /result`: `services/main-pipeline/cmd/api-gateway/main.go:155`.
118. Gateway регистрирует handler для `GET /result/{workflow_id}`: `services/main-pipeline/cmd/api-gateway/main.go:156`.
119. Result handler снова проверяет auth: `services/main-pipeline/cmd/api-gateway/main.go:113`.
120. Result handler берёт `workflow_id` из path: `services/main-pipeline/cmd/api-gateway/main.go:118`.
121. Если path-параметра нет, result handler берёт `workflow_id` из query: `services/main-pipeline/cmd/api-gateway/main.go:120`.
122. Result handler берёт optional `run_id` из query: `services/main-pipeline/cmd/api-gateway/main.go:122`.
123. Если `workflow_id` не передан, gateway возвращает `400`: `services/main-pipeline/cmd/api-gateway/main.go:124`.
124. Gateway запрашивает описание workflow execution через `DescribeWorkflowExecution`: `services/main-pipeline/cmd/api-gateway/main.go:128`.
125. Если workflow не найден, gateway возвращает `404`: `services/main-pipeline/cmd/api-gateway/main.go:130`.
126. Gateway читает текущий Temporal status workflow: `services/main-pipeline/cmd/api-gateway/main.go:134`.
127. Если workflow ещё не `COMPLETED`, gateway возвращает `202` и текущий статус: `services/main-pipeline/cmd/api-gateway/main.go:135`.
128. В ответе для незавершённого workflow gateway возвращает `workflow_id`: `services/main-pipeline/cmd/api-gateway/main.go:139`.
129. В ответе для незавершённого workflow gateway возвращает фактический `run_id`: `services/main-pipeline/cmd/api-gateway/main.go:140`.
130. В ответе для незавершённого workflow gateway возвращает `status`: `services/main-pipeline/cmd/api-gateway/main.go:141`.
131. Если workflow завершён, gateway создаёт переменную результата типа `pipeline.PipelineResult`: `services/main-pipeline/cmd/api-gateway/main.go:146`.
132. Gateway получает результат workflow через `GetWorkflow(...).Get(...)`: `services/main-pipeline/cmd/api-gateway/main.go:147`.
133. Если получение результата завершилось ошибкой, gateway возвращает `500`: `services/main-pipeline/cmd/api-gateway/main.go:148`.
134. Если результат получен, gateway выставляет `Content-Type: application/json`: `services/main-pipeline/cmd/api-gateway/main.go:152`.
135. Gateway сериализует `PipelineResult` в HTTP response: `services/main-pipeline/cmd/api-gateway/main.go:153`.
136. На этом конкретный pipeline полностью завершён: workflow result лежит в Temporal, gateway может его читать повторно, а worker-процессы остаются запущенными и ждут новые задачи: `services/main-pipeline/README.md:9`.
