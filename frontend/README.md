# Frontend Workspace

Это стартовая точка для всех frontend-приложений в проекте. Репозиторий приведен к виду workspace-монорепы, чтобы новые приложения и общий код добавлялись предсказуемо и без споров о структуре.

## Структура

```text
.
├── package.json
├── apps/
│   ├── sim-ui/          # текущий симулятор
│   ├── web/             # основной продуктовый интерфейс
│   └── apartment-ui/    # интерфейс квартиры
├── packages/
│   ├── ui/              # общий UI-kit и primitive-компоненты
│   └── simulation/      # доменные типы, модели, websocket-клиент, render/model logic
└── docs/                # общая документация workspace
```

## Правила workspace

- Все FE-приложения живут в `apps/*`.
- Общий код живет в `packages/*`.
- App-specific документация лежит в `apps/<name>/docs`.
- Общая документация лежит в корневом `docs/`.
- `.gitignore` должен быть один, в корне workspace.

## Единый стек

- Package manager: `npm`.
- Framework для приложений: `Next.js`.
- Язык: `TypeScript` в `strict`-режиме.
- Линтинг и форматирование: `ESLint` + `Prettier`.
- Импорт-алиасы и базовые настройки TypeScript должны быть согласованы между приложениями и пакетами.

## Текущее состояние

- Текущее приложение перенесено в `apps/sim-ui`.
- `packages/ui` и `packages/simulation` созданы как точки расширения под общий код.
- `apps/web` и `apps/apartment-ui` добавлены как заготовки под будущие приложения.

## Команды

Корень Workspace — это папка `./frontend/`

Из корня workspace:

```bash
npm ci
npm run dev:sim-ui
npm run lint
```

Если в dev-режиме страница открылась, но клики не меняют состояние, запусти
production-preview:

```bash
npm run preview:sim-ui
```

## Симуляция и WebSocket backend

Для полной проверки симуляции нужно отдельно поднять Go-сервис:

```bash
cd ../services/simulation
go run cmd/simulation/main.go
```

Затем из `frontend`:

```bash
npm ci
npm run dev:sim-ui
```

Для финальной ручной проверки можно использовать production-preview:

```bash
npm run preview:sim-ui
```

Симуляция откроется на `http://127.0.0.1:3000/simulation` и подключится к
`ws://127.0.0.1:8080`. Для быстрой проверки WebSocket-контракта:

```bash
npm run test:ws --workspace @smart-home/sim-ui
```

Страница симуляции сохраняет расстановку устройств в `localStorage`
(`simulation-plan-layout`) и автоматически активирует сценарии по устройствам, которые
реально перетащены на план. Пользователь на странице симуляции сценарии вручную не
собирает и не выбирает. Если WebSocket-бэк временно недоступен, интерфейс явно пишет это
в консоль событий и продолжает работать в локальном режиме.

## Дальше по структуре

- Вынести переиспользуемые UI-компоненты из `apps/sim-ui` в `packages/ui`.
- Вынести типы, модели и websocket-слой в `packages/simulation`.
- Добавить общий `tsconfig.base.json`, `prettier` и при необходимости shared ESLint config в `packages/config-*` или в корень workspace.
