<<<<<<< HEAD
## WebSocket Protocol (Simulation UI)

Документ описывает протокол взаимодействия между UI и backend для запуска и выполнения симуляции квартиры в режиме lockstep.

---

# Connection

* Endpoint: `/ws/simulation`
* Protocol: WebSocket
* Encoding: UTF-8 JSON

---

# Design principles

### 1. UI is the source of truth for simulation control

UI управляет временем симуляции через `simulation:tick`. Backend не продвигает симуляцию самостоятельно.

### 2. Lockstep execution

Каждый `tick` — атомарный шаг симуляции. Backend обязан обработать входные события и вернуть результат до следующего tick.

### 3. Backpressure через протокол

UI обязан ждать `simulation:step` перед отправкой следующего `tick`.

### 4. Stateless backend per session

Backend не хранит UI state квартиры. Все данные приходят в `simulation:start`.

### 5. Unified event model

Все события используют единый контейнер:
**entityId + payload (внутри payload находится kind)**

---

# Message Envelope

Все сообщения оборачиваются в стандартный WebSocket envelope:

```json
{
  "type": "string",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "optional-client-generated-id",
  "payload": {}
}
```

### Fields

* `type` — тип сообщения
* `ts` — timestamp события
* `reqId` — ID симуляционной сессии (обязателен для simulation lifecycle)
* `payload` — тело сообщения

---

# Unified Event Model

Все входные и выходные события используют единый формат:

```json
{
  "entityId": "string",
  "payload": {}
}
```

## Payload contract

`payload` всегда содержит:

* `kind` — тип команды или события
* дополнительные поля, специфичные для типа события

---

## Why kind is inside payload

* routing выполняется по `entityId`
* `kind` используется внутри entity для выбора команды
* payload остаётся self-contained domain object

---

# Known kinds

| kind                | direction | description                    |
| ------------------- | --------- | ------------------------------ |
| `human:move`        | input     | перемещение человека           |
| `human:interaction` | input     | взаимодействие с устройствами  |
| `device:trigger`    | input     | управление устройством         |
| `human:move`        | state     | обновление позиции человека    |
| `human:interaction` | state     | подтверждение взаимодействия   |
| `device:state`      | state     | изменение состояния устройства |

---

# Client → Server

---

## `hello`

Handshake после установления соединения.

```json
{
  "type": "hello",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "client": "sim-ui",
    "version": "1.0.0",
    "features": ["multiscenario"]
  }
}
```

---

## `simulation:start`

Инициализация симуляции.

```json
{
  "type": "simulation:start",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "dtSim": 1.0,
    "apartment": {
      "id": "apt_1",
      "floor": {}
    },
    "devices": [
      {
        "id": "lamp_hall",
        "type": "lamp",
        "roomId": "hall",
        "x": 0.52,
        "y": 0.78,
        "state": {
          "turned_on": false
        }
      }
    ],
    "scenarios": [
      {
        "id": "motion_light",
        "edges": [
          {
            "to": "lamp_hall",
            "action": "turn_on"
          }
        ]
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
        "entityId": "player_1",
        "payload": {
          "kind": "human:move",
          "to": {
            "x": 0.60,
            "y": 0.78
          }
        }
      },
      {
        "entityId": "lamp_hall",
        "payload": {
          "kind": "device:state",
          "turn_on": true
        }
      }
    ]
  }
}
```

---

## `simulation:stop`

Остановка симуляции.

```json
{
  "type": "simulation:stop",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

---

# Server → Client

---

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

---

## `simulation:started`

```json
{
  "type": "simulation:started",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "dtSim": 1.0,
    "state": "running"
  }
}
```

---

## `simulation:step`

Основной кадр симуляции.

```json
{
  "type": "simulation:step",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "tick": 1,
    "simTime": 1.0,
    "stateChanges": [
      {
        "entityId": "lamp_hall",
        "payload": {
          "kind": "device:state",
          "turn_on": true
        }
      },
      {
        "entityId": "player_1",
        "payload": {
          "kind": "human:move",
          "to": {
            "x": 0.60,
            "y": 0.78
          }
          "roomId": "hall",
          "status": "moved"
        }
      }
    ]
  }
}
```

---

## `simulation:status`

```json
{
  "type": "simulation:status",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "state": "running",
    "tick": 1
  }
}
```


---

## `error`

```json
{
  "type": "error",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "code": "INVALID_REQUEST",
    "message": "Invalid scenario"
  }
}
```

---


# Summary

* routing: `entityId`
* command dispatch: `payload.kind`
* payload: domain data only
* system is lockstep and deterministic
* UI fully controls simulation progression
=======
# WebSocket Protocol (Simulation UI)

This document defines the bidirectional WebSocket protocol between the UI and the backend
for running and visualizing apartment simulations.

## Connection

- Endpoint: `/ws/simulation`
- Transport: WebSocket (JSON messages)
- Encoding: UTF-8 JSON

## Message Envelope

Every message uses this common envelope:

```json
{
  "type": "string",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "optional-client-generated-id",
  "payload": {}
}
```

Fields:
- `type`: message name (see below)
- `ts`: ISO timestamp (client or server time)
- `reqId`: optional request id to match responses
- `payload`: message-specific data

## Client -> Server

### `hello`
Initial handshake. Client capabilities.

```json
{
  "type": "hello",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "client": "sim-ui",
    "version": "1.0.0",
    "features": ["multiscenario", "floor-v1"]
  }
}
```

### `floor:get`
Request current floor plan.

```json
{
  "type": "floor:get",
  "ts": "2026-02-18T12:00:00.000Z"
}
```

### `scenario:list`
Request available scenarios.

```json
{
  "type": "scenario:list",
  "ts": "2026-02-18T12:00:00.000Z"
}
```

### `simulation:start`
Start simulation.

```json
{
  "type": "simulation:start",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "mode": "parallel",
    "scenarioIds": ["scn_1", "scn_2"],
    "speed": 1.0
  }
}
```

### `simulation:pause`

```json
{
  "type": "simulation:pause",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

### `simulation:resume`

```json
{
  "type": "simulation:resume",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

### `simulation:stop`

```json
{
  "type": "simulation:stop",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

### `devices:update`
Client-side override for device positions (optional future feature).

```json
{
  "type": "devices:update",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "devices": [
      { "id": "motion_sensor_hall", "x": 0.52, "y": 0.78, "roomId": "hall" }
    ]
  }
}
```

## Server -> Client

### `hello:ack`

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

### `floor`
Returns current floor plan (see `floor.json` format).

```json
{
  "type": "floor",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": { "...": "floor.json content" }
}
```

### `scenario:list`

```json
{
  "type": "scenario:list",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "scenarios": [
      { "id": "scn_1", "title": "Движение → включить свет", "chain": ["motion_sensor_hall", "hub", "lamp_hall"] }
    ]
  }
}
```

### `simulation:status`
Current state of the simulation.

```json
{
  "type": "simulation:status",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "state": "running",
    "mode": "parallel",
    "speed": 1.0,
    "scenarioIds": ["scn_1", "scn_2"]
  }
}
```

### `simulation:step`
One step update for visualization.

```json
{
  "type": "simulation:step",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "scenarioId": "scn_1",
    "stepIndex": 2,
    "activeDevice": "lamp_hall",
    "activeEdge": ["hub", "lamp_hall"]
  }
}
```

### `device:state`
Device state update.

```json
{
  "type": "device:state",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "id": "lamp_hall",
    "state": "active"
  }
}
```

### `log:event`
Console/event feed.

```json
{
  "type": "log:event",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "level": "INFO",
    "device": "hub",
    "message": "Rule matched"
  }
}
```

### `error`

```json
{
  "type": "error",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "code": "INVALID_REQUEST",
    "message": "Scenario not found"
  }
}
```

>>>>>>> 4bf54f8 (hz)
