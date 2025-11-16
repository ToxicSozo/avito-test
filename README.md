# PR Reviewer Assignment Service

Микросервис назначает до двух активных ревьюверов из команды автора Pull Request, поддерживает переназначение, фиксирует merge и позволяет смотреть списки/статистику назначений. HTTP-контракт полностью описан в `api/openapi.yaml` (OpenAPI 3.0.3).

## Быстрый старт
1. Требования: Docker + docker-compose, make, свободный порт 8080. Для локального запуска без контейнера нужны Go 1.25 и PostgreSQL 16.
2. Настройте окружение (пример в `.env`): `PORT`, `DATABASE_URL`, `READ_TIMEOUT`, `WRITE_TIMEOUT`, `IDLE_TIMEOUT`, `SHUTDOWN_TIMEOUT`, `LOG_LEVEL`.
3. Полный цикл в контейнерах: `make compose-up` — собирает сервис, запускает PostgreSQL и приложение на `http://localhost:8080`, миграции применяются автоматически.
4. Остановить: `make compose-down`.
5. Локальный запуск без Docker: поднимите PostgreSQL, пропишите `DATABASE_URL`, затем `make run`.

## Возможности API
| Метод и путь | Назначение | Ошибки |
| --- | --- | --- |
| `POST /team/add` | Создать команду и upsert'ить участников (пользователя нельзя перенести в другую команду). | `TEAM_EXISTS`, `USER_EXISTS` |
| `GET /team/get?team_name=` | Получить состав команды. | `NOT_FOUND` |
| `POST /users/setIsActive` | Изменить флаг активности пользователя; неактивные не назначаются на ревью. | `NOT_FOUND` |
| `POST /pullRequest/create` | Создать PR и назначить до двух случайных активных ревьюверов из команды автора (автор исключён). | `NOT_FOUND`, `PR_EXISTS` |
| `POST /pullRequest/reassign` | Заменить конкретного ревьювера на активного участника его команды (исключая автора и уже назначенных). | `NOT_FOUND`, `NOT_ASSIGNED`, `NO_CANDIDATE`, `PR_MERGED` |
| `POST /pullRequest/merge` | Пометить PR как MERGED (идемпотентно, после этого менять ревьюверов нельзя). | `NOT_FOUND` |
| `GET /users/getReview?user_id=` | Список PR'ов, где пользователь назначен ревьювером. | `NOT_FOUND` |
| `GET /stats/assignments?limit=` | Статистика назначений на пользователей (1..500, по умолчанию 50). | — |

Ошибки всегда в формате `{"error":{"code":"...", "message":"..."}}`, перечень кодов — `TEAM_EXISTS`, `USER_EXISTS`, `PR_EXISTS`, `PR_MERGED`, `NOT_ASSIGNED`, `NO_CANDIDATE`, `NOT_FOUND`.

## Архитектура и стек
- Go 1.25, chi-router, oapi-codegen и middleware валидации OpenAPI.
- PostgreSQL 16 + GORM; миграции `internal/storage/postgres/migrations` применяются при старте.
- Логирование JSON через logrus (уровень задаётся `LOG_LEVEL`).
- Слои: transport/http (chi + кодоген шаблоны), domain/service с бизнес-правилами назначения и блокировками, storage/postgres с транзакциями и выборками кандидатов.

## Проверка качества
- E2E (testcontainers, нужен запущенный Docker): `make test-e2e`.
- Линтер (`golangci-lint` и `.golangci.yml`: таймаут 5 мин, включены errcheck, sqlclosecheck, govet, staticcheck, ineffassign, unused, dupl, gocyclo, goconst, nakedret, contextcheck; максимум 50 находок на линтер и до 3 повторов одной проблемы): `make lint`.

## Линтер
- Конфигурация `.golangci.yml` включает проверки errcheck, sqlclosecheck, govet, staticcheck, ineffassign, unused, dupl, gocyclo, goconst, nakedret, contextcheck.
- Таймаут запуска — 5 минут, lint прогоняет тестовые файлы (`tests: true`).
- Ограничения на отчёт: не более 50 замечаний от каждого линтера и до трёх повторов одной и той же проблемы (настройки `max-issues-per-linter`, `max-same-issues`).

## Дополнительно
- Примеры HTTP-запросов и curl-скриптов вынесены в `requests.md`.
- Допущения: при нехватке активных кандидатов назначается доступное количество ревьюверов (0/1); после MERGED любые изменения состава ревьюверов запрещены; `POST /pullRequest/merge` идемпотентен и всегда возвращает актуальное состояние PR.

## Расхождения с исходным ТЗ
- Добавлен необязательный эндпоинт статистики `GET /stats/assignments`, который упоминался в дополнительных заданиях, но отсутствовал в OpenAPI из ТЗ.
- Введён код ошибки `USER_EXISTS`, чтобы явно запретить перенос существующего пользователя в другую команду при `POST /team/add` и сохранить консистентность данных.
