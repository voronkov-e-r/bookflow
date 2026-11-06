# BookFlow — Микросервисная платформа для бронирования книг (Docker)

**BookFlow** — это учебный пет-проект, демонстрирующий микросервисную архитектуру на Go. 
Проект включает в себя 4 независимых сервиса, API Gateway, асинхронную обработку событий и распределенные транзакции (Saga).

> 🐳 **Примечание:** В данной ветке проект полностью контейнеризирован. Вся инфраструктура (базы данных, брокер сообщений и сами сервисы) поднимается одной командой через `docker-compose`. Вам не нужно устанавливать Go или БД на локальную машину.

## 🏗 Архитектура

Проект состоит из следующих компонентов:

1. **API Gateway (`gateway/`)** — точка входа для клиентов (HTTP/REST). Проксирует запросы в gRPC-сервисы, обрабатывает JWT-аутентификацию.
2. **Auth Service (`auth/`)** — управление пользователями (регистрация, логин). Использует PostgreSQL и выдает JWT.
3. **Catalog Service (`catalog/`)** — управление книгами. Использует MongoDB (гибкая схема, вложенные документы).
4. **Order Service (`order/`)** — оформление заказов (аренда книг). Использует PostgreSQL. Реализует **Saga-паттерн** для распределенных транзакций между Catalog и Order.
5. **Notification Worker (`notification/`)** — фоновый consumer, слушает RabbitMQ и обрабатывает события `order.created` (симуляция email/analytics).

### Схема взаимодействия (внутри Docker-сети):

    [Client (curl/Postman)]
           │ HTTP (порт 8080)
           ▼
    [ API Gateway ]
           │ gRPC (внутренняя сеть Docker)
           ├─────────────► [ Auth Service ] ───────────► [ PostgreSQL ]
           │
           ├─────────────► [ Catalog Service ] ────────► [ MongoDB ]
           │
           └─────────────► [ Order Service ] ──────────► [ PostgreSQL ]
                                  │
                                  │ (Publish Event)
                                  ▼
                            [ RabbitMQ ] (Fanout Exchange: events)
                                  │
                                  ▼
                        [ Notification Worker ]
                        (Email / Analytics)

## 🛠 Стек технологий

*   **Язык:** Go (1.22+)
*   **API:** REST (HTTP), gRPC (Protobuf)
*   **Базы данных:** PostgreSQL (транзакции, ACID), MongoDB (документы, агрегации)
*   **Брокер сообщений:** RabbitMQ (Fanout, DLQ)
*   **Аутентификация:** JWT (JSON Web Tokens)
*   **Логирование:** `log/slog` (структурированные логи)
*   **Инфраструктура:** Docker, Docker Compose

## 🚀 Запуск проекта (Docker)

Для запуска вам понадобятся только **Docker** и **Docker Compose**.

### 1. Запуск всей инфраструктуры
В корне проекта выполните:

    docker compose up --build -d

*(Флаг `-d` запускает контейнеры в фоновом режиме. Если вы запускаете впервые, сборка образов займет пару минут).*

### 2. Проверка статуса
Убедитесь, что все контейнеры запустились и прошли healthcheck:

    docker compose ps

Вы должны увидеть `Up` и `healthy` для `postgres`, `mongodb`, `rabbitmq` и запущенные сервисы `auth`, `catalog`, `order`, `notification`, `gateway`.

### 3. Инициализация Базы Данных (только первый раз)
Так как база данных PostgreSQL создается с нуля, вам нужно создать таблицы. Выполните команду:

    docker exec -it postgres psql -U default -d bookflow -c "
    CREATE TABLE users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        email TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL,
        name TEXT NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    CREATE TABLE orders (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        user_id UUID NOT NULL,
        book_id TEXT NOT NULL,
        book_title TEXT NOT NULL,
        status TEXT NOT NULL DEFAULT 'created',
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );"

### 4. Остановка проекта

    docker compose down

*(Если вы хотите удалить данные баз данных и начать с чистого листа, используйте `docker compose down -v`).*

## 🧪 Проверка работоспособности (через API Gateway)

После запуска API доступен на `http://localhost:8080` (или по IP вашей Docker-сети).

**1. Регистрация пользователя:**

    curl.exe -i -X POST localhost:8080/api/auth/register -H "Content-Type: application/json" -d '{"email":"test@test.com","password":"secret123","name":"Test User"}'

**2. Логин (получение JWT):**

    curl.exe -i -X POST localhost:8080/api/auth/login -H "Content-Type: application/json" -d '{"email":"test@test.com","password":"secret123"}'

**3. Создание книги:**

    curl.exe -i -X POST localhost:8080/api/books -H "Content-Type: application/json" -d '{"title":"1984","author":"Orwell","year":1949}'

**4. Список книг:**

    curl.exe -i localhost:8080/api/books

**5. Создание заказа:**
*(Используйте `user_id` из шага 2 и `book_id` из шага 3)*

    curl.exe -i -X POST localhost:8080/api/orders -H "Content-Type: application/json" -d '{"user_id":"<USER_ID>","book_id":"<BOOK_ID>"}'

**6. Возврат книги (Saga):**
*(Используйте `id` заказа из шага 5)*

    curl.exe -i -X POST localhost:8080/api/orders/<ORDER_ID>/return

## 📂 Структура проекта

    bookflow/
    ├── auth/            # gRPC сервис аутентификации
    ├── catalog/         # gRPC сервис каталога книг
    ├── gateway/         # HTTP REST API шлюз
    ├── notification/    # RabbitMQ consumer для уведомлений
    ├── order/           # gRPC сервис заказов (Saga)
    ├── proto/           # Protobuf контракты
    │   ├── auth.proto
    │   ├── catalog.proto
    │   └── order.proto
    ├── gen/             # Сгенерированный Go код из proto
    ├── cmd/             # Клиенты для ручного тестирования (опционально)
    ├── Dockerfile       # Мультистейдж сборка для всех Go-сервисов
    ├── docker-compose.yml # Конфигурация всей инфраструктуры и сервисов
    ├── go.mod
    └── go.sum

## ✨ Реализованные паттерны и решения
*   **Мультистейдж Docker-сборка** (бинарники собираются в `golang:alpine`, а запускаются в чистом `alpine:3.19` — минимальный размер образов).
*   **Оркестрация через Docker Compose** (автоматическое создание сети, проброс портов, `depends_on` с `condition: service_healthy`).
*   **API Gateway** (маршрутизация HTTP -> gRPC).
*   **Saga Pattern** (распределенная транзакция между Order и Catalog).
*   **Fanout Pub/Sub** (события RabbitMQ идут во все привязанные очереди).
*   **Graceful Shutdown** (корректное завершение по SIGINT/SIGTERM).
*   **Structured Logging** (через `slog`).
*   **Конфигурация через переменные окружения** (все сервисы читают `DB_URL`, `MONGO_URI`, `RABBITMQ_URL` из `env`).
