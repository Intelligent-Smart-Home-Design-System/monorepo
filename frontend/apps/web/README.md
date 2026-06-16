# apps/web

Основной продуктовый интерфейс интеллектуального планировщика умного дома.

## Возможности

- Приветственная страница для запуска нового подбора.
- Страница ввода настроек: бюджет, экосистема, хабы, треки и требования к устройствам.
- Загрузка плана квартиры в формате DXF.
- Интерактивная страница плана с устройствами, карточкой устройства, аналогами и маркетплейсами.
- Страницы аналитики и сценариев для демонстрации пользовательского флоу.
- Основной UI в `apps/web` подключён к backend API для загрузки справочников, запуска pipeline и просмотра результатов.
- Добавлены страницы регистрации и входа, хранение `access_token`/`refresh_token` и отправка JWT во frontend-запросах.

## Текущее сетевое взаимодействие

Сетевые вызовы собраны в `src/app/lib/api.ts` и идут через `fetch`.

Сейчас используются такие endpoints backend:

- `GET /api/v1/plans` — список планов для главной страницы.
- `GET /api/v1/ecosystems` — список экосистем для страницы настроек.
- `GET /api/v1/device-types` — типы устройств и допустимые фильтры.
- `POST /start` — запуск построения pipeline через api-gateway.
- `GET /result/{workflow_id}` — получение статуса или результата pipeline.
- `GET /api/v1/plans/{plan_id}/status` — legacy-статус старого плана.
- `GET /api/v1/plans/{plan_id}` — legacy-результат с bundles и listings.

Базовый URL задаётся через переменную окружения `NEXT_PUBLIC_API_BASE_URL`.

Если переменная не задана, frontend обращается по относительным путям, например `/api/v1/plans`. Это подходит только если backend проксируется тем же origin.

## Токены, auth и регистрация

В `apps/web` реализованы:

- страницы `/login` и `/register`;
- сохранение `access_token` и `refresh_token` в `localStorage`;
- добавление заголовка `Authorization: Bearer <access_token>` во все запросы к backend;
- попытка обновления access token через refresh endpoint при ответе `401`.

Пути auth endpoints можно переопределить переменными окружения:

- `NEXT_PUBLIC_AUTH_LOGIN_PATH`, по умолчанию `/api/v1/auth/login`;
- `NEXT_PUBLIC_AUTH_REGISTER_PATH`, по умолчанию `/api/v1/auth/register`;
- `NEXT_PUBLIC_AUTH_REFRESH_PATH`, по умолчанию `/api/v1/auth/refresh`.

Frontend ожидает от backend поля `access_token` и `refresh_token`. Также поддерживаются camelCase-поля `accessToken`/`refreshToken` и вложенный объект `tokens`.

## Переход в симуляцию

На странице готового плана есть кнопка «Открыть в симуляции». Она сериализует выбранный
bundle устройств и передает его в `apps/sim-ui` через query-параметр `devices`.

URL симуляции можно задать переменной:

```bash
NEXT_PUBLIC_SIM_UI_URL=http://127.0.0.1:3000/simulation
```

Если переменная не задана, используется `http://127.0.0.1:3000/simulation`.

## Запуск

Из директории `frontend`:

```bash
npm ci
npm run dev --workspace @smart-home/web
```

Пример запуска с backend:

```bash
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080 npm run dev --workspace @smart-home/web
```

Приложение откроется на `http://localhost:3000` или на следующем свободном порте.

## Проверка

```bash
npm run build --workspace @smart-home/web
```

App-specific документацию и макеты можно хранить рядом в `apps/web/docs`.
