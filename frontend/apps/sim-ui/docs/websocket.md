## WebSocket Protocol (Simulation UI)

Документ описывает актуальный контракт общения `sim-ui` с backend simulation. При расхождении с текущей локальной визуализацией frontend должен подстраиваться под этот контракт.

---

# Connection

В браузере frontend подключается не напрямую к simulation backend, а через gateway:

```text
ws://localhost:8090/api/v1/simulation/ws?token=<access_token>
```

Внутренний путь внутри docker-сети:

```text
frontend -> nginx -> api-gateway -> JWT validation -> simulation backend
```

`api-gateway` после проверки токена проксирует соединение в:

```text
ws://simulation:8080/ws/simulation
```

## Authentication

Браузерный `WebSocket` API не позволяет задать `Authorization` header, поэтому access token передается query-параметром `token`.

Frontend берет token из `localStorage["smart-home-auth"].tokens.access_token`.

Если token отсутствует, `sim-ui` не должен открывать WebSocket и должен перейти в локальный/disabled режим.

---

# Design Principles

## 1. UI controls ticks

Frontend управляет временем симуляции через `simulation:tick`.

Backend не должен сам продвигать UI-сессию без входящего tick. На каждый tick backend обрабатывает входные события, выполняет один шаг симуляции и возвращает `simulation:step`.

## 2. Backend is source of truth for incidents

Распространение пожара, потопа и дыма считается на backend через BFS-сетку фиксированных блоков.

Frontend не рассчитывает распространение incidents самостоятельно. Он отправляет действия пользователя и отображает `incidents[].blocks`, рассчитанные backend.

## 3. Full incident snapshot

Backend отдает полный snapshot активных incident-блоков для конкретного kind, а не только новые блоки. Frontend должен заменять слой соответствующего kind целиком.

## 4. Unified event container

Входные события на tick передаются через единый контейнер:

```json
{
  "entity_id": "string",
  "payload": {
    "kind": "string"
  }
}
```

Если событие должно попасть в конкретную сущность, `entity_id` должен быть ID этой сущности. Поле `payload.trigger` может использоваться для совместимости с текущим backend normalize logic, но новый код должен предпочитать явный `entity_id`.

---

# Message Envelope

Все сообщения кодируются как UTF-8 JSON.

```json
{
  "type": "string",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {}
}
```

Поля:

* `type` - тип сообщения.
* `ts` - timestamp события.
* `reqId` - ID симуляционной сессии.
* `payload` - тело сообщения.

---

# Known Kinds

| kind | direction | description |
| --- | --- | --- |
| `human:move` | input/state | перемещение человека |
| `human:trigger` | input | человек триггерит устройство |
| `device:trigger` | input | ручное управление устройством |
| `device:state` | state | изменение состояния устройства |
| `fire:spread` | state | распространение пожара |
| `flood:spread` | state | распространение потопа |
| `smoke:spread` | state | распространение дыма |

---

# Client -> Server

## `hello`

Frontend отправляет `hello` сразу после открытия WebSocket.

```json
{
  "type": "hello",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "client": "sim-ui",
    "version": "0.1.0",
    "features": ["multiscenario", "floor-v1", "fire", "flood", "human-move", "device-trigger"]
  }
}
```

## `simulation:start`

Запускает backend-сессию симуляции.

```json
{
  "type": "simulation:start",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "dtSim": 1,
    "apartment": {
      "meta": { "units": "mm" },
      "walls": [],
      "doors": [],
      "windows": [],
      "rooms": []
    },
    "devices": [],
    "scenarios": []
  }
}
```

`apartment` должен соответствовать backend floor DTO:

* `walls[]` - стены с `id`, `points`, `width`;
* `doors[]` - двери с `id`, `points`, `width`, `rooms`;
* `windows[]` - окна;
* `rooms[]` - комнаты с `id`, `name`, `area`, `walls`, `doors`, `windows`.

Для backend incidents в `devices` должны присутствовать incident-сущности:

```json
{
  "id": "fire",
  "type": "fire",
  "info": {
    "id": "fire",
    "cellSize": 500
  }
}
```

Аналогично:

* `{ "id": "flood", "type": "flood", ... }`
* `{ "id": "smoke", "type": "smoke", ... }`

Incident-сущности создаются при `simulation:start`, но остаются неактивными. `cellSize` задается в единицах исходного плана и определяет сторону одной BFS-клетки. Стартовая точка передается позже, когда пользователь размещает incident на плане.

## `simulation:tick`

Продвигает симуляцию на один шаг.

```json
{
  "type": "simulation:tick",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "tick": 1,
    "inputs": [
      {
        "entity_id": "resident",
        "payload": {
          "kind": "human:move",
          "to": { "x": 0.6, "y": 0.78 },
          "devices_payload": ["motion_sensor_hall"]
        }
      },
      {
        "entity_id": "lamp_hall",
        "payload": {
          "kind": "human:trigger",
          "turn_on": true
        }
      }
    ]
  }
}
```

Для включения уже созданного incident:

```json
{
  "type": "simulation:tick",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "tick": 2,
    "inputs": [
      {
        "entity_id": "fire",
        "payload": {
          "kind": "fire:spread",
          "turn_on": true,
          "x": 1200,
          "y": 800,
          "roomID": "room_1"
        }
      }
    ]
  }
}
```

`x` и `y` передаются в системе координат исходного `apartment`, а не в экранном диапазоне 0...1. `roomID` должен совпадать с `rooms[].id`. Backend использует эти данные для начальной клетки и не требует координат в `simulation:start`.

При размещении пожара frontend активирует только `fire` с kind `fire:spread`. Дым запускается независимо отдельным событием для entity `smoke` с kind `smoke:spread`. Затопление активирует `flood` с kind `flood:spread`. После активации frontend не отправляет incident повторно на каждом tick: backend сам выполняет следующий BFS-шаг при последующих `simulation:tick`.

Для сброса incident frontend отправляет сущности payload `{ "reset": true }`. Backend очищает BFS-сетку, выключает затронутые датчики через пустой список блоков и возвращает snapshot с `"incidents": []`. После сброса ту же сущность можно снова активировать новым `turn_on` с другими координатами.

## `simulation:stop`

Останавливает backend-сессию.

```json
{
  "type": "simulation:stop",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

---

# Server -> Client

## `hello:ack`

```json
{
  "type": "hello:ack",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "server": "sim-backend",
    "version": "1.0.0"
  }
}
```

## `simulation:started`

```json
{
  "type": "simulation:started",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "dtSim": 1,
    "state": "running"
  }
}
```

## `simulation:step`

```json
{
  "type": "simulation:step",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "tick": 1,
    "simTime": 1,
    "stateChanges": [],
    "triggeredEdges": [],
    "humans": []
  }
}
```

`stateChanges` - фактические изменения состояния сущностей. Это основной источник для UI.

`triggeredEdges` - задел для трассировки сценариев. Сейчас backend его почти не заполняет, frontend не должен зависеть от этого поля.

## Incident state change

Incident приходит внутри `stateChanges[]`.

```json
{
  "entity_id": "fire",
  "payload": {
    "kind": "fire:spread",
    "incidents": [
      {
        "roomID": "room_1",
        "blocks": [
          {
            "id": "room_1:2:3",
            "roomID": "room_1",
            "x": 1200,
            "y": 800,
            "size": 500,
            "points": [
              [950, 550],
              [1450, 550],
              [1450, 1050],
              [950, 1050]
            ]
          }
        ]
      }
    ]
  }
}
```

Frontend должен:

* читать `payload.kind`;
* поддерживать `fire:spread`, `flood:spread`, `smoke:spread`;
* брать `incidents[].blocks[].points`;
* рисовать каждый block как polygon;
* заменять весь слой этого kind новым snapshot-ом;
* не использовать старую локальную радиусную визуализацию, если backend blocks уже пришли.

`points` могут быть обрезаны backend-ом по стенам. Frontend не должен повторно обрезать block, он только отображает polygon.

## `simulation:stopped`

```json
{
  "type": "simulation:stopped",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

## `log:event`

Зарезервировано для backend-логов в UI.

```json
{
  "type": "log:event",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "level": "INFO",
    "device": "lamp_hall",
    "message": "Свет включен"
  }
}
```

## `error`

```json
{
  "type": "error",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "code": "INVALID_PAYLOAD",
    "message": "cannot parse simulation:start payload"
  }
}
```

---

# Local Smoke Check

Через прямое подключение к backend:

```bash
cd services/simulation
go run cmd/simulation/main.go
```

```bash
cd frontend
npm run test:ws --workspace @smart-home/sim-ui
```

Через полный docker path:

```bash
make up-test
```

Полный E2E-маршрут через nginx, API Gateway и JWT проверяется Playwright-тестом. Если локальный PostgreSQL уже занимает `5432`, опубликуй catalog DB на другом host-порту:

```bash
CATALOG_DB_HOST_PORT=5433 docker compose -f docker-compose.apps.yaml --profile test up -d --build
cd frontend
npm run test:e2e:simulation
```

Тест проверяет `401` без токена и с неверным токеном, затем запускает simulation с валидным JWT, размещает пожар, ожидает backend polygons и отсутствие активации датчика дыма, выполняет reset и повторную активацию.

Открыть:

```text
http://localhost:8090/sim-ui/simulation
```

Путь `/sim-ui` проксируется nginx в отдельное Next.js-приложение sim-ui. Страница входа, конфигуратор и симуляция благодаря этому имеют один browser origin `localhost:8090` и используют общий `localStorage`, включая `smart-home-auth`, `simulation-floor` и `simulation-devices`.

### Проверка соединения

В состоянии `running` сообщения `simulation:tick` и ответы `simulation:step` подтверждают, что соединение работает. До запуска и во время паузы frontend раз в 25 секунд отправляет `ping`, backend отвечает `pong` с тем же `reqId`. Heartbeat не изменяет tick или `simTime`. Если ответ на tick либо `pong` отсутствует 60 секунд, frontend закрывает зависшее соединение и выполняет reconnect. Nginx использует `proxy_read_timeout 75s`, поэтому исправный WebSocket может работать без ограничения общей длительности.
