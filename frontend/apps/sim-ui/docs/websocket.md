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

## `log:event`

```json
{
  "type": "log:event",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "level": "INFO",
    "message": "Rule triggered"
  }
}
```


# Summary

* routing: `entityId`
* command dispatch: `payload.kind`
* payload: domain data only
* system is lockstep and deterministic
* UI fully controls simulation progression
