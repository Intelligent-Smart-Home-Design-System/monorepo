Temporal UI: http://localhost:8080
Grafana: http://localhost:3000
Jaeger: http://localhost:16686
login: `admin`
password: `admin`

Старт:
1.Поднятие контейнеров:
docker compose up -d --build

2.Запуск активити:
docker compose --profile tools run --rm temporal-client

3.Далее можно посмотреть трейсы в джагере
