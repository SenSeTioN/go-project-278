### Hexlet tests and linter status:

[![Actions Status](https://github.com/SenSeTioN/go-project-278/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/SenSeTioN/go-project-278/actions)
[![CI](https://github.com/SenSeTioN/go-project-278/actions/workflows/ci.yml/badge.svg)](https://github.com/SenSeTioN/go-project-278/actions/workflows/ci.yml)

## Demo

**Live:** https://go-project-278-3hc9.onrender.com

## Переменные окружения

| Имя            | Описание                                   | Обязательная                          |
| -------------- | ------------------------------------------ | ------------------------------------- |
| `PORT`         | Порт HTTP-сервера (по умолчанию `8080`)    | нет                                   |
| `BASE_URL`     | Базовый URL для формирования `short_url`   | нет (дефолт `http://localhost:$PORT`) |
| `DATABASE_URL` | DSN PostgreSQL (для миграций и приложения) | для `/api/links`                      |
| `SENTRY_DSN`   | DSN проекта Sentry для отправки ошибок     | нет                                   |

Локальная разработка: скопируй `.env.example` → `.env`, подставь значения.

## Эндпоинты

| Метод  | Путь                | Описание                                                                       |
| ------ | ------------------- | ------------------------------------------------------------------------------ |
| GET    | `/ping`             | `pong`                                                                         |
| GET    | `/debug-sentry`     | сгенерировать panic для проверки Sentry                                        |
| GET    | `/r/:code`          | редирект (302) на `original_url`, запись посещения                             |
| GET    | `/api/links`        | список ссылок (пагинация через `?range=[from,to]`)                             |
| POST   | `/api/links`        | создать ссылку (`short_name` опционален)                                       |
| GET    | `/api/links/:id`    | получить ссылку                                                                |
| PUT    | `/api/links/:id`    | обновить ссылку                                                                |
| DELETE | `/api/links/:id`    | удалить ссылку                                                                 |
| GET    | `/api/link_visits`  | список посещений (пагинация через `?range=[from,to]` или заголовок `Range`)    |

Пагинация: ответ содержит заголовки `Content-Range: <resource> {from}-{to}/{total}` и `Accept-Ranges: <resource>`.

## Стек

**Backend (Go):**
- Gin — HTTP-фреймворк
- gin-contrib/cors — CORS для запросов с фронтенда
- go-playground/validator — валидация тел запросов
- Sentry (`sentry-go` + `sentrygin`) — мониторинг ошибок
- PostgreSQL + sqlc (типобезопасные запросы) + goose (миграции)
- godotenv — загрузка `.env` локально

**Frontend:** `@hexlet/project-url-shortener-frontend` (npm-пакет с готовым `dist/`).

**Инфраструктура:**
- Docker (multi-stage build)
- Caddy — reverse proxy и раздача статики на проде
- Render — деплой
- GitHub Actions — CI (lint + test + build)

## Команды

### Go / Makefile

```bash
make run            # запустить приложение
make dev            # фронт + бэк одновременно (npm run dev)
make air            # бэк с автоперезагрузкой через Air
make build          # собрать бинарь в bin/app
make test           # go test -race -v
make lint           # golangci-lint
make lint-fix       # golangci-lint с автофиксом
make fmt            # go fmt ./...
make tidy           # go mod tidy
make ci             # lint + test + build (то же, что в GitHub Actions)
make clean          # rm -rf bin

make sqlc           # перегенерировать internal/db по запросам db/queries

make migrate-up      # применить миграции goose
make migrate-down    # откатить последнюю миграцию
make migrate-status  # показать статус миграций
make migrate-create NAME=add_users  # создать новую миграцию
```

### Frontend (npm-скрипты)

```bash
npm run backend   # только Go-сервер
npm run frontend  # только Vite preview на :5173
npm run dev       # оба процесса параллельно через concurrently
```
