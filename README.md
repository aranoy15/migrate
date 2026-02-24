# Database Migration Tool

Универсальная утилита для управления миграциями базы данных PostgreSQL.

## Установка

```bash
go build -o migrate main.go
```

## Использование

### Переменные окружения

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=database_name
DB_SSLMODE=disable
```

### Команды

```bash
# Применить все миграции
./migrate -command=up -schema=my_schema -path=./migrations

# Применить N миграций
./migrate -command=up -steps=1 -schema=my_schema -path=./migrations

# Откатить все миграции
./migrate -command=down -schema=my_schema -path=./migrations

# Откатить N миграций
./migrate -command=down -steps=1 -schema=my_schema -path=./migrations

# Показать текущую версию
./migrate -command=version -schema=my_schema -path=./migrations

# Принудительно установить версию
./migrate -command=force -version=1 -schema=my_schema -path=./migrations
```

### Параметры

- `-command` - команда: `up`, `down`, `force`, `version` (обязательно)
- `-schema` - имя схемы PostgreSQL (обязательно)
- `-path` - путь к папке с миграциями (обязательно)
- `-steps` - количество шагов для up/down (опционально, 0 = все)
- `-version` - версия для force команды (обязательно для force)

## Формат миграций

Миграции должны следовать формату golang-migrate:
- `{version}_{name}.up.sql` - применение миграции
- `{version}_{name}.down.sql` - откат миграции

Пример:
- `000001_create_users_table.up.sql`
- `000001_create_users_table.down.sql`
