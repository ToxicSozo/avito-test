# Сервис назначения ревьюеров Pull Request

Микросервис автоматизирует процесс назначения ревьюверов на PR внутри команды,
а также даёт CRUD-возможности над командами, пользователями и собирает
статистику по ревью. Вся публичная контрактная часть описана в
`api/openapi.yaml`, код обработчиков сгенерирован через `oapi-codegen`.

## Возможности
- создание/обновление команд вместе с их участниками (upsert пользователей);
- хранение пользователей и переключение их активности без потери истории;
- создание PR и автоматическое назначение до двух активных тиммейтов автора;
- переназначение конкретного ревьювера под контролем доменных правил;
- идемпотентный merge;
- получение списка PR, где пользователь назначен ревьювером;
- агрегация статистики назначений по пользователям и PR.

## Технологический стек
- **Go 1.25** — основной язык;
- **PostgreSQL 16** — постоянное хранилище, миграции вшиты в бинарь;
- **pgx/v5** — драйвер и пул соединений;
- **chi/v5** — HTTP-роутер/мидлвары;
- **logrus** — структурированные JSON-логи;
- **oapi-codegen** — генерация серверной обвязки из OpenAPI;
- **testcontainers-go** — e2e-тест поверх живой базы.

## Архитектура и кодовая база
- `cmd/reviewer` — входная точка: загрузка конфига, запуск приложения.
- `internal/app` — сборка HTTP-стека, graceful shutdown.
- `internal/domain/service` — бизнес-логика (команды, PR, выбор ревьюверов, статистика).
- `internal/storage/postgres` — инициализация пула PG и применение миграций (`internal/storage/postgres/migrations/001_init.sql`).
- `internal/storage/postgres/repository` — слой работы с БД, используемый сервисом/транзакциями.
- `internal/transport/http/handlers` — ручная часть HTTP-обработчиков, которую вызывает кодген.
- `api/openapi.yaml` + `api/gen.go` — спецификация и конфиг генерации.
- `pkg/logger` — настройка JSON-логгера.

## Хранение и миграции
Миграции хранятся во вшитой директории `internal/storage/postgres/migrations`.
При создании `pgxpool` вызывается `runMigrations`, который читает файлы из `embed.FS`
и применяет их в алфавитном порядке. Таким образом, `docker-compose up`
гарантированно поднимает схему без дополнительных шагов.

## Быстрый старт (Docker Compose)
```bash
make up
```
- собирает образ приложения;
- стартует `postgres:16` и сам сервис;
- сервис доступен на `http://127.0.0.1:8080`.

Остановка и очистка данных:
```bash
make down
```

## Локальный запуск без Docker
1. Поднимите PostgreSQL (например, `reviewers`).
2. Заполните переменные окружения (можно через `.env`):
   ```bash
   export DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/reviewers?sslmode=disable"
   export PORT=8080
   export LOG_LEVEL=debug
   # опционально: READ_TIMEOUT, WRITE_TIMEOUT, IDLE_TIMEOUT, SHUTDOWN_TIMEOUT
   ```
3. Запустите сервис:
   ```bash
   make run
   ```

При старте миграции выполняются автоматически, после чего HTTP-сервер слушает
указанный порт.

## Конфигурация
`internal/config/config.go` читает переменные окружения (поддерживается `.env`
через `github.com/joho/godotenv`):

| Переменная | Значение по умолчанию | Описание |
| --- | --- | --- |
| `PORT` | `8080` | Порт HTTP-сервера |
| `DATABASE_URL` | — | DSN PostgreSQL (обязателен) |
| `READ_TIMEOUT` | `5s` | `http.Server.ReadTimeout` |
| `WRITE_TIMEOUT` | `10s` | `http.Server.WriteTimeout` |
| `IDLE_TIMEOUT` | `60s` | `http.Server.IdleTimeout` |
| `SHUTDOWN_TIMEOUT` | `10s` | Время graceful shutdown |
| `LOG_LEVEL` | `info` | Уровень логирования logrus |

## API
Полная спецификация лежит в `api/openapi.yaml`. Основные эндпоинты:
- `POST /team/add` — создать/обновить команду с участниками;
- `GET /team/get` — получить команду по имени;
- `POST /users/setIsActive` — включить/выключить пользователя;
- `POST /pullRequest/create` — создать PR и автоматически назначить ревью;
- `POST /pullRequest/merge` — отметить PR как MERGED (идемпотентно);
- `POST /pullRequest/reassign` — переназначить одного ревьювера;
- `GET /users/getReview` — вывести PR, где пользователь ревьювер;
- `GET /stats/assignments` — агрегированная статистика назначений.

Для обновления кодогенерации:
```bash
make gen
```

## Пример полного сценария
```bash
# 1. Создать команду и участников
curl -s -X POST http://127.0.0.1:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{"team_name":"backend","members":[
        {"user_id":"u1","username":"Alice","is_active":true},
        {"user_id":"u2","username":"Bob","is_active":true},
        {"user_id":"u3","username":"Charlie","is_active":true},
        {"user_id":"u4","username":"Diana","is_active":true}
      ]}'

# 2. Создать PR и получить автоприсвоенных ревьюверов
curl -s -X POST http://127.0.0.1:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1001","pull_request_name":"Add search","author_id":"u1"}'

# 3. Переназначить ревьювера
curl -s -X POST http://127.0.0.1:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1001","old_user_id":"u2"}'

# 4. Замержить PR
curl -s -X POST http://127.0.0.1:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1001"}'

# 5. Посмотреть, где u2 ревьювер
curl -s "http://127.0.0.1:8080/users/getReview?user_id=u2"

# 6. Проверить статистику назначений
curl -s http://127.0.0.1:8080/stats/assignments
```

## Makefile

| Команда | Описание |
| --- | --- |
| `make build` | Сборка Linux/amd64 бинаря в `bin/reviewer` |
| `make run` | Локальный запуск (нужен `DATABASE_URL`) |
| `make test-e2e` | E2E сценарий через Testcontainers (`test/e2e`) |
| `make lint` | Запуск `golangci-lint` |
| `make gen` | Перегенерация API-кода |
| `make up` / `make down` | Поднять/остановить docker-compose |

*Примечание:* unit-тесты запускаются стандартно (`go test ./...`); отдельная
цель в Makefile не нужна.

## Тестирование
- `go test ./...` — быстрые тесты (репозитории/сервисы);
- `make test-e2e` — поднимает Postgres в контейнере и гоняет сценарий через HTTP API;
- `make lint` — статический анализ по `.golangci.yml`.

## Дополнительные задания
Реализовано:
1. **Эндпоинт статистики** `/stats/assignments` с агрегатами по пользователям
   и PR (`api/openapi.yaml`, `internal/storage/postgres/repository/stats.go`,
   `internal/transport/http/handlers/stats.go`).
2. **E2E-тест** `TestEndToEndScenario` на Testcontainers
   (`test/e2e/e2e_test.go`), доступен командой `make test-e2e`.
3. **Конфигурация линтера** — `.golangci.yml` с набором включённых чекеров и
   единым временем выполнения.

Не реализовано (оставлено как возможное развитие):
- нагрузочное тестирование и публикация метрик;
- массовая деактивация команды с безопасной переназначаемостью открытых PR.

## Известные ограничения и допущения
- При создании команды текущая реализация допускает повторное добавление уже
  существующей команды (идёт upsert без отдельной ошибки); контракт можно
  ужесточить по требованию.
  При попытке добавить участника из другой команды вернётся ошибка `TEAM_EXISTS`.

## Логгирование и мониторинг
  structured logging, Recoverer, Timeout).
- JSON-логи пишутся через `pkg/logger` с выбранным уровнем.

Приятной проверки!
