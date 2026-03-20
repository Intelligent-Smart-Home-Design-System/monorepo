# Frontend Workspace

Это стартовая точка для всех frontend-приложений в проекте. Репозиторий приведен к виду workspace-монорепы, чтобы новые приложения и общий код добавлялись предсказуемо и без споров о структуре.

## Структура

```text
.
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

Из корня workspace:

```bash
npm install
npm run dev:sim-ui
npm run lint
```

## Дальше по структуре

- Вынести переиспользуемые UI-компоненты из `apps/sim-ui` в `packages/ui`.
- Вынести типы, модели и websocket-слой в `packages/simulation`.
- Добавить общий `tsconfig.base.json`, `prettier` и при необходимости shared ESLint config в `packages/config-*` или в корень workspace.
