# URL Shortener

Сервис для сокращения ссылок на Go: сохраняет длинный URL и отдаёт короткий алиас, по которому потом делает редирект.

## Запуск

клонируем
```bash
git clone 
cp .env.example .env  
docker compose up 
```
сервис работает

## Миграции

Схема БД (`url`: таблица и индекс по `alias`) описана в SQL-файлах и **не создаётся приложением при старте** —
`postgres.New` только открывает пул соединений. Файлы:

```
migrations/
├── 000001_init.up.sql     # применение схемы (формат golang-migrate)
├── 000001_init.down.sql   # откат схемы
└── init/
    └── 000001_init.sql    # тот же DDL для docker-entrypoint-initdb.d
```

При `docker compose up` файлы из `migrations/init` монтируются в `/docker-entrypoint-initdb.d`,
и postgres применяет их при инициализации **пустого** volume `postgres-data`.
Если volume уже существует, схема не обновится — пересоздайте окружение:

```bash
docker compose down -v
docker compose up
```

Применение миграций вручную (например, на существующем volume) через `migrate`:

```bash
docker run --rm \
  -v "$PWD/migrations:/migrations" \
  migrate/migrate \
  -path=/migrations \
  -database "${DATABASE_URL}" up
```

> `DATABASE_URL` из `.env` указывает на хост `postgres`, который резолвится только внутри
> docker-сети. Для запуска миграций с хоста замените его на `localhost:5432`.

Новую миграцию добавляйте парой файлов `migrations/<номер>_<имя>.up.sql` и `.down.sql`,
а `up`-часть дублируйте в `migrations/init/` под следующим номером — либо применяйте
её через `migrate` (тогда дублирование не нужно).

## API

Все запросы к `/url` защищены Basic Auth (`AUTH_USER` / `AUTH_PASSWORD` из `.env`).

| Метод    | Путь           | Описание                       |
|----------|----------------|--------------------------------|
| `POST`   | `/url`         | Сохранить URL и получить алиас |
| `DELETE` | `/url/{alias}` | Удалить ссылку по алиасу       |
| `GET`    | `/{alias}`     | Редирект (302) на исходный URL |
| `PATCH`  | `/url/{alias}` | Обновить URL и при необходимости alias |

в методах POST и PATCH поле `alias` необязательно: если его не передать, в POST алиас сгенерируется случайно, в PATCH останется прежним
