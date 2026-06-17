## WebSocket Protocol (Simulation UI)

Документ описывает протокол взаимодействия между UI и backend для запуска и выполнения симуляции квартиры в режиме lockstep.

---

# Connection

* Endpoint: `/ws/simulation`
* Local service: `ws://127.0.0.1:8080/ws/simulation`
* Protocol: WebSocket
* Encoding: UTF-8 JSON

UI автоматически переподключается при закрытии сокета. Если backend недоступен во время запуска, страница остается рабочей и явно пишет в консоль событий, что симуляция продолжена локально.

---

# UI State

* Устройства с продуктовой страницы читаются из query-параметра `devices`, затем кэшируются в `localStorage` под ключом `simulation-devices`.
* Размещение устройств на плане сохраняется в `localStorage` под ключом `simulation-plan-layout`.
* Реальный план квартиры может быть передан в `localStorage` через `simulation-floor`, `planner-floor-json`, `parsed-floor` или `floor-json`.
* Сценарии на странице симуляции не выбираются пользователем вручную: UI выводит доступные сценарии из устройств, размещенных на плане.
* Если backend или внешний план недоступны, UI использует локальный fallback, но сообщает об этом в логах.

---

# Design Principles

### 1. UI is the source of truth for simulation control

UI управляет временем симуляции через `simulation:tick`. Backend не продвигает симуляцию самостоятельно.

### 2. Lockstep execution

Каждый `tick` — атомарный шаг симуляции. Backend обрабатывает входные события и возвращает результат через `simulation:step`.

### 3. Stateless backend per session

Backend не хранит UI state квартиры. Все данные приходят в `simulation:start`.

### 4. Unified event model

Входные события используют единый контейнер:

```json
{
  "entity_id": "string",
  "payload": {
    "trigger": "optional-device-id"
  }
}
```

`payload.kind` содержит тип команды или события. Если действие должно попасть в конкретное устройство, его ID передается в `payload.trigger`.

---

# Message Envelope

```json
{
  "type": "string",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "optional-client-generated-id",
  "payload": {}
}
```

### Fields

* `type` — тип сообщения.
* `ts` — timestamp события.
* `reqId` — ID симуляционной сессии.
* `payload` — тело сообщения.

---

# Known Kinds

| kind | direction | description |
| --- | --- | --- |
| `human:move` | input/state | перемещение человека |
| `human:trigger` | input | человек триггерит устройство |
| `device:trigger` | input | ручное управление устройством |
| `device:state` | state | изменение состояния устройства |
| `environment:trigger` | input | пожар, потоп или другое событие среды |

---

# Client → Server

## `hello`

```json
{
  "type": "hello",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "client": "sim-ui",
    "version": "0.1.0",
    "features": ["multiscenario", "floor-v1", "fire", "flood", "human-move", "device-trigger"]
  }
}
```

---

## `simulation:start`

UI отправляет это сообщение и ждет `simulation:started`. Если подтверждение не пришло за 2 секунды, frontend продолжает локальную симуляцию.

```json
{
  "type": "simulation:start",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "dtSim": 1,
    "apartment": {
      "id": "apt_1",
      "floor": {}
    },
    "devices": [
      {
        "id": "motion_sensor_hall",
        "type": "lampSwitcher",
        "info": {
          "id": "motion_sensor_hall",
          "delay": 0,
          "turned_on": false,
          "x": 0.5,
          "y": 0.8
        }
      },
      {
        "id": "lamp_hall",
        "type": "lamp",
        "info": {
          "id": "lamp_hall",
          "delay": 0,
          "turned_on": false,
          "x": 0.3,
          "y": 0.62
        }
      }
    ],
    "scenarios": [
      {
        "id": "motion_sensor_hall",
        "edges": [{ "to": "lamp_hall", "action": "trigger" }]
      }
    ]
  }
}
```

---

## `simulation:tick`

Продвижение симуляции на один шаг.

```json
{
  "type": "simulation:tick",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "tick": 1,
    "inputs": [
      {
        "entity_id": "player_1",
        "payload": {
          "kind": "human:move",
          "to": {
            "x": 0.6,
            "y": 0.78
          },
          "devices_payload": ["motion_sensor_hall"]
        }
      },
      {
        "entity_id": "player_1",
        "payload": {
          "kind": "human:trigger",
          "trigger": "lamp_hall",
          "turn_on": true
        }
      }
    ]
  }
}
```

### Input fields

* `entity_id` — сущность, которая совершает действие.
* `payload.trigger` — ID устройства, которое было триггернуто человеком или другим событием.
* `payload.kind` — тип события.
* `payload.devices_payload` — массив ID устройств, которые попали в контекст события, например датчики движения рядом с человеком.

---

## `simulation:stop`

```json
{
  "type": "simulation:stop",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

---

# Server → Client

## `simulation:started`

```json
{
  "type": "simulation:started",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "status": "running"
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
    "stateChanges": [
      {
        "entity_id": "lamp_hall",
        "payload": {
          "kind": "device:state",
          "turn_on": true
        }
      }
    ],
    "events": []
  }
}
```

## `log:event`

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

Сначала запустить backend:

```bash
cd services/simulation
go run cmd/simulation/main.go
```

Потом из `frontend`:

```bash
npm run test:ws --workspace @smart-home/sim-ui
```
