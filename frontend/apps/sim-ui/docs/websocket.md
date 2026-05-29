## WebSocket Protocol (Simulation UI)

Документ описывает двунаправленный WebSocket протокол между UI и бэкендом для запуска и визуализации симуляций квартиры.

## Connection

- Endpoint: `/ws/simulation`
- Transport: WebSocket (JSON messages)
- Encoding: UTF-8 JSON

## Design notes (important)

- Бэкенд **не получает** план квартиры из других модулей. UI уже имеет план квартиры, устройства и их позиции, полученные на предыдущих шагах пайплайна, и передаёт всё необходимое в `simulation:start`.
- UI управляет симуляцией в режиме **lockstep**: каждый "тик" UI — это сообщение `simulation:tick`. Это обеспечивает детерминированность симуляции.
- **Pause / resume** — забота UI. Если UI перестаёт слать тики, бэкенд естественным образом перестаёт продвигать симуляцию.
- Все входящие события и исходящие изменения состояний используют единый конверт (`kind`, `entityId`, `payload`). Поле `payload` специфично для каждой сущности и парсится бэкендом на основе `kind`.

## Message Envelope

Каждое сообщение использует общий конверт:

```json
{
  "type": "string",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "optional-client-generated-id",
  "payload": {}
}
```

Поля:
- `type`: название сообщения (см. ниже)
- `ts`: ISO timestamp (время клиента или сервера)
- `reqId`: опциональный ID запроса для сопоставления ответов. Клиент генерирует его сам и передаёт в `simulation:start`. Все последующие сообщения в рамках этой сессии (тики, события, ошибки) содержат тот же `reqId`
- `payload`: данные специфичные для типа сообщения

## Unified event envelope

Все входящие события (client → server внутри `simulation:tick`) и все изменения состояний (server → client внутри `simulation:step`) используют единую структуру:

```json
{
  "kind": "string",
  "entityId": "string",
  "payload": {}
}
```

Поля:
- `kind`: описывает природу события (например `human:move`, `human:interaction`, `device:trigger`)
- `entityId`: стабильный уникальный ID сущности над которой выполняется действие
- `payload`: данные специфичные для сущности и типа события, парсятся бэкендом или UI на основе `kind`

### Known `kind` values

| kind | направление | форма payload |
|---|---|---|
| `human:move` | input | `{ "x": 0.60, "y": 0.78 }` — целевые координаты движения |
| `human:interaction` | input | `{ "device_id": "lamp_hall", "payload": { ... } }` — взаимодействие с устройством |
| `device:trigger` | input | устройство-специфичный патч, например `{ "turn_on": true }` |
| `human:move` | state change | `{ "x": 0.60, "y": 0.78, "roomId": "hall", "status": "moved" }` — новая позиция и комната |
| `human:interaction` | state change | `{ "entity_id": "lamp_hall", "status": "triggered" }` — подтверждение взаимодействия |
| `device:state` | state change | устройство-специфичный патч, например `{ "turn_on": true }` |

## Client -> Server

### `hello`

Первоначальный handshake. Клиент сообщает свои возможности. Должен быть первым сообщением после установки соединения.

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

### `simulation:start`

Запуск сессии симуляции. UI передаёт все необходимые данные: план квартиры, устройства с их ID/типами/позициями и граф сценариев (смежность устройств).

Важно:
- `dtSim` — количество **симуляционного времени** продвигаемого каждым `simulation:tick`. UI контролирует реальную скорость частотой отправки тиков
- `devices[].id` должен быть стабильным и уникальным в рамках сессии. Сценарии ссылаются на устройства по этим ID

```json
{
  "type": "simulation:start",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "dtSim": 1.0,
    "apartment": {
      "id": "apt_1",
      "floor": { "...": "ui floor plan payload (existing UI format)" }
    },
    "devices": [
      {
        "id": "lamp_hall",
        "type": "lamp",
        "roomId": "hall",
        "x": 0.52,
        "y": 0.78,
        "state": { "turned_on": false }
      }
    ],
    "scenarios": [
      {
        "id": "motion_light_hall",
        "edges": [
          { "to": "lamp_hall", "action": "turn_on" }
        ]
      }
    ]
  }
}
```

### `simulation:tick`

Продвигает симуляцию на один lockstep тик. UI должен дождаться соответствующего ответа `simulation:step` перед отправкой следующего тика — это обеспечивает детерминированность и backpressure.

Все входящие события собранные с предыдущего тика передаются вместе. Бэкенд применяет их атомарно перед продвижением симуляционного времени.

```json
{
  "type": "simulation:tick",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "tick": 1,
    "inputs": [
      {
        "kind": "human:move",
        "entityId": "player_1",
        "payload": { "x": 0.60, "y": 0.78 }
      },
      {
        "kind": "device:trigger",
        "entityId": "lamp_hall",
        "payload": { "turn_on": true }
      },
      {
        "kind": "human:interaction",
        "entityId": "player_1",
        "payload": { "device_id": "lamp_hall", "payload": { "turn_on": true } }
      }
    ]
  }
}
```

### `simulation:stop`

Останавливает текущую сессию симуляции. Бэкенд освобождает все ресурсы связанные с `reqId`.

```json
{
  "type": "simulation:stop",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

## Server -> Client

### `hello:ack`

Подтверждение handshake. Сервер сообщает свою версию.

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

### `simulation:started`

Подтверждение успешного запуска симуляции. Сервер возвращает эффективные параметры сессии.

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

### `simulation:status`

Текущее состояние симуляции. Может быть запрошено клиентом или отправлено сервером по своей инициативе.

```json
{
  "type": "simulation:status",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "state": "running",
    "dtSim": 1.0,
    "tick": 1
  }
}
```

### `simulation:step`

Один lockstep апдейт для визуализации. Все изменения состояний за тик передаются в одном сообщении. Каждое изменение использует единый конверт события.

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
        "kind": "device:state",
        "entityId": "lamp_hall",
        "payload": { "turn_on": true }
      },
      {
        "kind": "human:move",
        "entityId": "player_1",
        "payload": { "x": 0.60, "y": 0.78, "roomId": "hall", "status": "moved" }
      },
      {
        "kind": "human:interaction",
        "entityId": "player_1",
        "payload": { "entity_id": "lamp_hall", "status": "triggered" }
      }
    ]
  }
}
```

### `log:event`

Лог событий симуляции для отображения в консоли UI. Отправляется бэкендом при значимых событиях внутри симуляции.

```json
{
  "type": "log:event",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "level": "INFO",
    "device": "hub",
    "message": "Rule matched"
  }
}
```

### `error`

Ошибка обработки запроса. Содержит `reqId` запроса который вызвал ошибку.

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