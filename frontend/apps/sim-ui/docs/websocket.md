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

Разрыв WebSocket немедленно останавливает и удаляет связанную backend-сессию. Автоматического восстановления состояния симуляции нет: после установления нового соединения требуется новый `simulation:start`.

## Authentication

Браузерный `WebSocket` API не позволяет задать `Authorization` header, поэтому access token передается query-параметром `token`.

Frontend берет token из `localStorage["smart-home-auth"].tokens.access_token`.

Если token отсутствует, `sim-ui` не должен открывать WebSocket и должен перейти в локальный/disabled режим.

---

# Design Principles

## 1. UI controls ticks

Frontend управляет временем симуляции через `simulation:tick`.

Backend не должен сам продвигать UI-сессию без входящего tick. На каждый tick backend обрабатывает входные события, выполняет один шаг симуляции и возвращает `simulation:step`.

Один шаг frontend-сессии равен `dtSim = 0.05` секунды симуляционного времени. При скорости `1.0x` frontend отправляет tick каждые 50 мс. Множитель скорости изменяет период отправки (`50 / speed` мс), но не `dtSim`: например, при `2.0x` tick отправляется каждые 25 мс. Frontend держит не более одного неподтверждённого tick: следующий отправляется только после `simulation:step`.

## 2. Backend is source of truth for incidents

Распространение пожара, потопа и дыма считается на backend через BFS-сетку фиксированных блоков.

Frontend не рассчитывает распространение incidents самостоятельно. Он отправляет действия пользователя и отображает `incidents[].blocks`, рассчитанные backend.

Пользовательские действия не создают дополнительные `simulation:tick`. Frontend накапливает запуск и сброс incidents, управление устройствами и `human:move`, а затем прикладывает их к следующему плановому tick. События датчиков движения после принятого перемещения создаёт backend. Только периодический таймер отправляет `simulation:tick`, поэтому частота действий пользователя не изменяет `simTime` и скорость распространения incidents.

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

Для `simulation:start`, `simulation:tick` и `simulation:stop` поле `reqId` обязательно. `dtSim` и номер `tick` должны быть больше нуля. Backend отвечает с тем же `reqId`; frontend игнорирует ответы другой сессии и уже применённые номера тиков.

---

# Known Kinds

| kind | direction | description |
| --- | --- | --- |
| `human:move` | input/state | перемещение человека |
| `human:route` | input/state | маршрут человека и управление им |
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
    "dtSim": 0.05,
    "apartment": {
      "meta": { "units": "mm" },
      "walls": [],
      "doors": [],
      "windows": [],
      "rooms": []
    },
    "devices": [
      {
        "id": "resident",
        "type": "human",
        "info": {
          "id": "resident",
          "x": 1200,
          "y": 800,
          "roomID": "room_1"
        }
      }
    ],
    "scenarios": []
  }
}
```

`apartment` должен соответствовать backend floor DTO:

* `walls[]` - стены с `id`, `points`, `width`;
* `doors[]` - двери с `id`, `points`, `width`, `rooms`;
* `windows[]` - окна;
* `rooms[]` - комнаты с `id`, `name`, `area`, `walls`, `doors`, `windows`.

Каждый `scenarios[].id` обязан ссылаться на существующий `devices[].id`, как и каждый `scenarios[].edges[].to`. Backend валидирует обе стороны связи до инициализации движка и отклоняет `simulation:start`, если source или target отсутствует.

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

Житель также создаётся при `simulation:start` как сущность типа `human`. Его начальные `x`, `y` передаются в координатах исходного floor, а `roomID` должен указывать на комнату, полигон которой содержит эту точку.

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
        "entity_id": "resident_1",
        "payload": {
          "kind": "human:move",
          "to": { "x": 1500, "y": 800 }
        }
      },
      {
        "entity_id": "lamp_hall",
        "payload": {
          "kind": "device:trigger",
          "turn_on": true
        }
      }
    ]
  }
}
```

Ручная команда всегда адресуется самому устройству через `entity_id`; frontend не меняет его состояние до ответа backend. Управляемые устройства принимают независимое поле `turn_on`. Устройства с уровнем (умная лампа, диммер, шторы) дополнительно принимают `percents` в диапазоне 0...100: изменение уровня не включает и не выключает устройство. Датчики инцидентов, движения и камеры вручную не переключаются: их состояние рассчитывается backend по наблюдаемым событиям.

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

Frontend не удаляет incident-полигоны сразу после отправки reset. Слой остается видимым до `stateChanges` с пустым backend snapshot соответствующего kind.

## `simulation:stop`

Останавливает backend-сессию.

```json
{
  "type": "simulation:stop",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

Frontend сохраняет текущее отображение до ответа `simulation:stopped`. Incident-слои и остальные подтверждённые состояния очищаются только после этого ответа.

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
    "dtSim": 0.05,
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
    "simTime": 0.05,
    "stateChanges": [],
    "triggeredEdges": [
      {
        "from": "motion_sensor_1",
        "to": "lamp_hall",
        "action": "trigger"
      }
    ],
    "humans": []
  }
}
```

`stateChanges` - фактические изменения состояния сущностей. Это основной источник для UI. Для управляемых устройств frontend считает активным только подтверждённое backend поле `turn_on: true`; `percents` и другие числовые настройки отображаются независимо. Локальная отправка команды сама по себе не меняет визуальное состояние.

`triggeredEdges` содержит уникальные постоянные сценарные связи, которые фактически передали событие в рамках этого tick. Backend заполняет `from`, `to`, `action` и, если они заданы, `data`. Frontend использует массив для визуализации пути срабатывания, но фактическое состояние сущностей по-прежнему берёт только из `stateChanges`.

Разовые действия человека (`human:interaction`) и внутренние уведомления комнатных observers не входят в `triggeredEdges`, потому что они не являются постоянными связями из `simulation:start.payload.scenarios`.

Frontend не выполняет цепочки сценариев локально и не назначает им режимы «параллельно» или «по очереди». Он передаёт постоянный граф зависимостей при `simulation:start`, затем отправляет только плановые тики и пользовательские inputs. Порядок, задержки и фактические срабатывания определяет backend. Для заметной визуализации frontend может кратковременно удерживать уже полученную связь на экране, но это не должно создавать события или менять состояние устройств.

## Human movement

Frontend добавляет движение жителя в ближайший плановый `simulation:tick`:

```json
{
  "entity_id": "resident",
  "payload": {
    "kind": "human:move",
    "to": { "x": 1500, "y": 800 }
  }
}
```

Backend проверяет выход маршрута за границу полигона `room.area`. Пересечь границу можно только через дверь, которая связывает текущую и соседнюю комнаты в `floor.Adjacency`; необязательный или неполный список `room.walls` не используется как единственный источник коллизий. После проверки backend обновляет комнату и возвращает фактическую позицию:

```json
{
  "entity_id": "resident",
  "payload": {
    "kind": "human:move",
    "to": { "x": 1498.4, "y": 800 },
    "roomID": "room_1",
    "status": "moved"
  }
}
```

Frontend не проверяет стены и не активирует датчики движения самостоятельно. Он держит не более одного неподтверждённого `human:move`, а каждый шаг стрелкой размером `0.015` в нормализованных координатах рассчитывает от последней подтверждённой backend-позиции. Маркер обновляется только по `stateChanges`, поэтому команды не накапливаются за заблокированной границей и одного обратного шага достаточно для отхода от стены. Backend после принятого движения уведомляет наблюдателей текущей комнаты. Массив `humans` в `simulation:step` пока зарезервирован и может быть пустым.

Для маршрута frontend отправляет только выбранные пользователем опорные точки:

```json
{
  "entity_id": "resident",
  "payload": {
    "kind": "human:route",
    "action": "start",
    "speed": 1,
    "route": [
      { "x": 1500, "y": 800 },
      { "x": 2200, "y": 1400 }
    ]
  }
}
```

Backend самостоятельно разбивает движение на шаги, продвигает маршрут только на плановых tick и проверяет стены и двери на каждом шаге. Команды `pause`, `resume` и `stop` передаются тем же kind в поле `action`. Frontend получает статусы `running`, `paused`, `completed`, `blocked` или `stopped`.

После `simulation:start` размещение устройств фиксируется. До `simulation:stop` frontend запрещает добавлять, перемещать и удалять устройства, чтобы координаты UI и зарегистрированных backend observers не расходились.

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
* считать сообщением полного snapshot только payload с массивом `incidents`; события датчиков с тем же `kind`, но без `incidents`, не должны менять слой;
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

`TICK_FAILED` означает, что backend не смог применить входное событие текущего тика, например из-за неизвестного `entity_id` или некорректного payload устройства. Такая сессия немедленно останавливается, чтобы frontend и backend не продолжали работу с разными состояниями; для продолжения требуется новый `simulation:start` с новым `reqId`.

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
