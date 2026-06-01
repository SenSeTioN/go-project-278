### Hexlet tests and linter status:
[![Actions Status](https://github.com/SenSeTioN/go-project-278/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/SenSeTioN/go-project-278/actions)
[![CI](https://github.com/SenSeTioN/go-project-278/actions/workflows/ci.yml/badge.svg)](https://github.com/SenSeTioN/go-project-278/actions/workflows/ci.yml)

## Demo

<!-- TODO: добавить ссылку на развёрнутое приложение -->
URL: https://TODO.onrender.com

## Переменные окружения

| Имя | Описание | Обязательная |
|---|---|---|
| `PORT` | Порт HTTP-сервера (по умолчанию `8080`) | нет |
| `DATABASE_URL` | DSN PostgreSQL для миграций `goose` | для запуска миграций |
| `SENTRY_DSN` | DSN проекта Sentry для отправки ошибок | нет |

## Эндпоинты

- `GET /ping` → `pong`
- `GET /debug-sentry` — генерирует panic для проверки интеграции с Sentry
