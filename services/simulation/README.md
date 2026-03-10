# Сервис симуляции умного дома

Сервис для симуляции поведения устройств и сущностей умного дома с использованием дискретно-событийного моделирования.

## Архитектура

```
cmd/simulation/main.go     <- Точка входа
internal/
├── api/                   <- обмен данными с внешними сервисами (DTO/...)
├── client/                <- Клиент для взаимодействия с внешними сервисами
├── entities/              <- Бизнес-логика сущностей
│   ├── actors/            <- Акторы (пожарный, сантехник, ...)
│   ├── devices/           <- Устройства (свет, климат, безопасность, ...)
│   └── field/             <- Поле симуляции (план квартиры)
├── processing/            <- Обработка данных
│   ├── converter/         <- Конвертация DTO в сущности
│   ├── engine/            <- Движок симуляции
│   ├── fetcher/           <- Получение данных
│   └── sender/            <- Отправка событий
└── simulations/           <- Управление симуляциями
```

### наглядное представление архитектуры: [ссылка](https://miro.com/app/board/uXjVGb1bTTs=/?share_link_id=166215177234)

## Основные компоненты

### Simulations

Главный компонент, который управляет всеми симуляциями. Связывает логику компонентов (fetcher, sender, engine) между собой.

```go
type Simulations struct {
    fetcher          fetcher.Fetcher
    sender           sender.Sender
    IDToEngine       map[string]engine.Engine        // engineID <-> engine
    IDToEventInChan  map[string]chan api.EventInDTO  // engineID <-> канал входящих событий
    IDToEventOutChan map[string]chan api.EventOutDTO // engineID <-> канал исходящих событий
    IDToDependencies map[string][]api.ActionDTO      // engineID <-> зависимости
}
```

### Engine (Движок)

Движок симуляции использует библиотеку [simgo](https://github.com/fschuetz04/simgo) для дискретно-событийного моделирования.

**Основные функции:**
- `InitEntities` — инициализация сущностей и их зависимостей
- `InitProcesses` — запуск процессов для каждой сущности
- `CheckCircleDependencies` — проверка циклических зависимостей (DFS алгоритм)
- `HandleEvent` — обработка событий и триггеринг зависимых сущностей
- `UpdateField` — обновление состояния ячеек на поле

### Сущности (Entities)

#### Интерфейс Entity
```go
type Entity interface {
    GetID() string
    GetReceiversID() []string           // сущности, которых триггерит данная сущность
    SetReceivers(actions []api.ActionDTO)
}
```

#### Интерфейс EntityWithProcess
```go
type EntityWithProcess interface {
    Entity
    HandleInDTO(dto []byte) error       // обработка входящих данных
    HandleOutDTO(out any) error         // обработка исходящих данных
    GetProcessFunc() func(process simgo.Process)
    Process(process simgo.Process)      // функция процесса устройства
    GetOutCh() chan []byte
}
```

#### Типы устройств
| Тип | Константа | Описание |
|-----|-----------|----------|
| Лампа | `lamp` | Управляемый источник света |
| Переключатель лампы | `lamp_switcher` | Выключатель для лампы |

### Поле (Field)

Представляет план квартиры для симуляции.

```go
type Field struct {
    Width  int
    Height int
    Cells  [][]*Cell
}

type Cell struct {
    Condition    bool // true - сгорело; false - дефолт
    IsHiddenWall bool // невидимая стенка для пожара
}
```

## DTO структуры

### EventInDTO / EventOutDTO
```go
type EventInDTO struct {
    EntityID string          `json:"entityID"`
    Info     json.RawMessage `json:"info"`
}
```

### EntityDTO
```go
type EntityDTO struct {
    ID        string          `json:"id"`
    Receivers []string        `json:"receivers"` // те, кого данная сущность тригерит
    Info      json.RawMessage `json:"info"`
}
```

### ActionDTO
```go
type ActionDTO struct {
    ID         string        `json:"id"`
    ActionName string        `json:"action_name"`
    Data       []interface{} `json:"data"`
}
```

## Правила и логика работы

### 1. Жизненный цикл симуляции

1. **Инициализация** (`Init`)
   - Запуск sender
   - Инициализация движков для каждой симуляции
   - Загрузка полей и сущностей
   - Проверка циклических зависимостей
   - Запуск процессов

2. **Работа** (`Run`)
   - Получение событий от fetcher (блокирующая операция)
   - Распределение событий по каналам соответствующих движков
   - Обработка событий в движках

3. **Остановка** (`Stop`)
   - Закрытие каналов входящих событий
   - Graceful shutdown через контекст

### 2. Зависимости между сущностями

- Сущности могут триггерить другие сущности через `Receivers` (слайс зависимостей)
- При получении события, движок автоматически отправляет события зависимым сущностям
- **Циклические зависимости запрещены** — проверяются через DFS алгоритм при инициализации

### 3. Обработка событий

1. Событие поступает в канал `eventsInChan`
2. `HandleEvent` обрабатывает событие:
   - Получает сущность по `EntityID`
   - Триггерит все зависимые сущности (`Receivers`)
   - Вызывает `HandleInDTO` для сущностей с процессами
3. Выполняется шаг симуляции (`simStep = 1.0`)

### 4. Правила для устройств

- Каждое устройство имеет уникальный ID формата `{type}_{identifier}` (например, `lamp_1`)
- Устройства могут иметь задержку реакции (`Delay`)
- Входящие данные сохраняются в `Store` и обрабатываются в процессе

### 5. Поле симуляции

- Координаты ячеек проверяются на валидность (x: 0..Height, y: 0..Width)
- Состояние ячеек можно обновлять через `UpdateField`

## Запуск

```bash
go run cmd/simulation/main.go
```

## Зависимости

- `github.com/fschuetz04/simgo` — дискретно-событийное моделирование
- `golang.org/x/sync` — errgroup для управления горутинами