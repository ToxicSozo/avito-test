# Примеры HTTP-запросов

Все запросы предполагают запущенный сервис на `http://localhost:8080`.

```bash
# Создать команду и участников
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alice", "is_active": true},
      {"user_id": "u2", "username": "Bob", "is_active": true},
      {"user_id": "u3", "username": "Eve", "is_active": true}
    ]
  }'

# Создать PR и получить автоматически назначенных ревьюверов
curl -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1001","pull_request_name":"Add search","author_id":"u1"}'

# Переназначить ревьювера u2 на другого участника его команды
curl -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1001","old_user_id":"u2"}'

# Пометить PR как MERGED
curl -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1001"}'

# Обновить флаг активности пользователя
curl -X POST http://localhost:8080/users/setIsActive \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u2","is_active":false}'

# Получить PR'ы, где пользователь ревьювер
curl "http://localhost:8080/users/getReview?user_id=u2"
```
