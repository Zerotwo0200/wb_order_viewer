## WB Order Service

Демо‑сервис для приёма, сохранения и просмотра заказов из NATS Streaming.

### Возможности
- ✅ Хранение в PostgreSQL (JSONB)
- ✅ In‑memory кэш с восстановлением из БД при старте
- ✅ Подписка на NATS Streaming с durable‑очередью
- ✅ HTTP API: `GET /api/order/{id}`
- ✅ Простой веб‑интерфейс на `http://localhost:8080/`

### Требования
- Docker + Docker Compose
- Go 1.21+

### Быстрый старт (Windows CMD)

1. Запустить инфраструктуру:
   ```cmd
   docker compose up -d
   ```

2. Запустить сервер:
   ```cmd
   go run ./cmd/server
   ```

3. Отправить тестовый заказ (в новом окне CMD):
   ```cmd
   type model.json | go run ./cmd/publisher
   ```

4. Открыть браузер и перейти на `http://localhost:8080/`.
   Ввести UID: `b563feb7b2b84b6test`.

### Структура проекта
```
wb/
├── cmd/
│   ├── server/      # Основной сервис (подписчик NATS + HTTP API)
│   └── publisher/   # Публикатор тестового сообщения
├── internal/
│   ├── domain/      # Сущности и порты (интерфейсы хранилищ/кэша)
│   ├── usecase/     # Юзкейсы: загрузка кэша, получение заказа, обработка входа
│   └── adapter/
│       ├── repo/    # Адаптер репозитория: PostgreSQL (JSONB upsert/load)
│       └── cache/   # Адаптер кэша: in-memory
├── web/
│   └── index.html   # Веб‑UI
├── model.json       # Пример данных заказа
├── docker-compose.yml
└── README.md
```

### API
- `GET /api/order/{id}` — получить заказ по его UID

### Конфигурация
Переменные окружения:
- `DATABASE_URL` → `postgres://wbuser:wbpass@localhost:5433/wborders`
- `NATS_URL` → `nats://localhost:4223`
- `STAN_CLUSTER_ID` → `wb-cluster`
- `STAN_SUBJECT` → `orders`

### Архитектура

```
NATS Streaming → Service → PostgreSQL (JSONB‑хранилище)
                         → In‑Memory Cache (быстрые чтения)
```

Слои :
- Domain: `internal/domain` — сущности и порты (интерфейсы)
- UseCase: `internal/usecase` — сценарии работы с заказами
- Adapters: `internal/adapter/*` — реализация портов (PostgreSQL, кэш)
- Adapters: `internal/adapter/httpapi`, `internal/adapter/natsstan`, `internal/adapter/repo`, `internal/adapter/cache`
- Composition: `cmd/server` — только сборка зависимостей и запуск

Жизненный цикл:
- При старте: юзкейс `LoadCache` восстанавливает кэш из PostgreSQL.  
- При сообщении: юзкейс `ProcessIncomingOrder` сохраняет в БД, обновляет кэш, затем ACK.

### Команды (Windows CMD)

```cmd
:: Запустить Docker
docker compose up -d

:: Запустить сервис
go run ./cmd/server

:: Отправить тестовый заказ (в новом окне)
type model.json | go run ./cmd/publisher

:: Запустить тесты
go test ./cmd/server -v

:: Остановить Docker
docker compose down
```

### Автотесты, бенчмарки и стресс‑тесты (Windows CMD)
```cmd
:: Тесты
go test ./cmd/server -v
go test ./cmd/server -cover

:: Бенчмарки
go test ./cmd/server -bench=.

:: Стресс‑тесты (Vegeta)

go install github.com/tsenart/vegeta@latest

echo GET http://localhost:8080/api/order/b563feb7b2b84b6test | vegeta attack -rate=100 -duration=10s | vegeta report

:: Стресс‑тесты (WRK)

docker run --rm williamyeh/wrk -t4 -c100 -d30s http://host.docker.internal:8080/order/b563feb7b2b84b6test
```

### Пояснение по файлам (.go)
- `cmd/server/main.go` — точка входа; сборка зависимостей, запуск HTTP и подписчика NATS.
- `cmd/server/main_test.go` — юнит‑тесты HTTP/юзкейсов через адаптеры.
- `cmd/server/bench_test.go` — бенчмарки обработчика и кэша.
- `cmd/publisher/main.go` — утилита публикации тестового заказа в NATS Streaming (читает JSON из stdin).
- `internal/domain/order.go` — доменная сущность заказа.
- `internal/domain/ports.go` — доменные порты: репозиторий, кэш, подписчик; общие ошибки.
- `internal/usecase/order_usecases.go` — юзкейсы: `LoadCache`, `GetOrderByID`, `ProcessIncomingOrder`.
- `internal/adapter/repo/postgres_repo.go` — адаптер репозитория для PostgreSQL (upsert/load, EnsureSchema).
- `internal/adapter/cache/memory_cache.go` — адаптер in‑memory кэша заказов.
- `internal/adapter/httpapi/server.go` — HTTP‑сервер (маршрут `GET /api/order/{id}` + статика `web/`).
- `internal/adapter/natsstan/subscriber.go` — адаптер подписки NATS Streaming (STAN) с ручным ACK.


