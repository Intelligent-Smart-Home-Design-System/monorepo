<<<<<<< HEAD
# apps/web

Основной продуктовый интерфейс интеллектуального планировщика умного дома.

## Возможности

- Приветственная страница для запуска нового подбора.
- Страница ввода настроек: бюджет, экосистема, хабы, треки и требования к устройствам.
- Загрузка плана квартиры в форматах DXF и PNG.
- Интерактивная страница плана с устройствами, карточкой устройства, аналогами и маркетплейсами.
- Страницы аналитики и сценариев для демонстрации пользовательского флоу.
- Основной UI в `apps/web` подключён к backend API для загрузки справочников, создания плана и просмотра результатов.
- Добавлены страницы регистрации и входа, хранение `access_token`/`refresh_token` и отправка JWT во frontend-запросах.

## Текущее сетевое взаимодействие

Сетевые вызовы собраны в `src/app/lib/api.ts` и идут через `fetch`.

Сейчас используются такие endpoints backend:

- `GET /api/v1/plans` — список планов для главной страницы.
- `GET /api/v1/ecosystems` — список экосистем для страницы настроек.
- `GET /api/v1/presets` — пресеты требований.
- `GET /api/v1/device-types` — типы устройств и допустимые фильтры.
- `POST /api/v1/plans` — создание нового плана.
- `GET /api/v1/plans/{plan_id}/status` — статус генерации плана.
- `GET /api/v1/plans/{plan_id}` — готовый результат с bundles и listings.

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

## Запуск

Из директории `frontend`:

```bash
npm install
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
=======
# apps/web

Основной продуктовый интерфейс интеллектуального планировщика умного дома.

## Возможности

- Приветственная страница для запуска нового подбора.
- Страница ввода настроек: бюджет, экосистема, хабы, треки и требования к устройствам.
- Загрузка плана квартиры в форматах DXF и PNG.
- Интерактивная страница плана с устройствами, карточкой устройства, аналогами и маркетплейсами.
- Страницы аналитики и сценариев для демонстрации пользовательского флоу.
- Основной UI в `apps/web` уже подключён к backend API для загрузки справочников, создания плана и просмотра результатов.

## Текущее сетевое взаимодействие

Сетевые вызовы собраны в `src/app/lib/api.ts` и идут через `fetch`.

Сейчас используются такие endpoints backend:

- `GET /api/v1/plans` — список планов для главной страницы.
- `GET /api/v1/ecosystems` — список экосистем для страницы настроек.
- `GET /api/v1/presets` — пресеты требований.
- `GET /api/v1/device-types` — типы устройств и допустимые фильтры.
- `POST /api/v1/plans` — создание нового плана.
- `GET /api/v1/plans/{plan_id}/status` — статус генерации плана.
- `GET /api/v1/plans/{plan_id}` — готовый результат с bundles и listings.

Базовый URL задаётся через переменную окружения `NEXT_PUBLIC_API_BASE_URL`.

Если переменная не задана, frontend обращается по относительным путям, например `/api/v1/plans`. Это подходит только если backend проксируется тем же origin.

## Токены, auth и регистрация

На текущий момент в `apps/web` не реализованы:

- токены доступа;
- заголовок `Authorization`;
- cookie/session auth;
- login/registration flows;
- отдельные запросы на регистрацию, логин или обновление токена.

Во frontend есть `localStorage`, но он используется только для локального UI-состояния:

- `planner-uploaded-plan` — превью загруженного пользователем плана;
- данные для demo-страниц аналитики и сценариев.

## Запуск

Из директории `frontend`:

```bash
npm install
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
>>>>>>> 4bf54f8 (hz)
