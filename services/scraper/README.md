# Scraper service

Сервис отвечает за скачивание ресурсов маркетплейсов/документации экосистем и их сохранение в [БД каталога](../../db/catalog/README.md), в бронзовый слой.  

### Usage 
```
cd services/scraper

go build -o scraper

./mytool scrape --config ./cmd/scraper/config.local.toml

# Запустить несколько источников
./mytool scrape --config ./cmd/scraper/config.local.toml --sources printer,wildberries
```

Конфиг - config.toml, пример есть в cmd/scraper. В нем необходимо указать параметры для подключения к БД. Пароль задается переменной окружения:
`SCRAPER_DATABASE_PASSWORD`

При запуске сервис читает все страницы, необходимые для скрейпинга, из БД, скрейпит их (скачивает ресурсы) и пишет снепшот ресурсов для каждой страницы в БД.

