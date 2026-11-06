# BookFlow — Микросервисная платформа для бронирования книг (Local Dev)

**BookFlow** — это учебный пет-проект, демонстрирующий микросервисную архитектуру на Go. 
Проект включает в себя 4 независимых сервиса, API Gateway, асинхронную обработку событий и распределенные транзакции (Saga).

> ⚠️ **Примечание:** В данной ветке проект запускается локально. Все зависимости (Postgres, MongoDB, RabbitMQ) должны быть запущены на вашей машине при помощи docker-compose.yml. Для ознакомления с конечной версией проекта переключитесь на ветку production.

## 🏗 Архитектура

Проект состоит из следующих компонентов:

1. **API Gateway (`gateway/`)** — точка входа для клиентов (HTTP/REST). Проксирует запросы в gRPC-сервисы, обрабатывает JWT-аутентификацию.
2. **Auth Service (`auth/`)** — управление пользователями (регистрация, логин). Использует PostgreSQL и выдает JWT.
3. **Catalog Service (`catalog/`)** — управление книгами. Использует MongoDB (гибкая схема, вложенные документы).
4. **Order Service (`order/`)** — оформление заказов (аренда книг). Использует PostgreSQL. Реализует **Saga-паттерн** для распределенных транзакций между Catalog и Order.
5. **Notification Worker (`notification/`)** — фоновый consumer, слушает RabbitMQ и обрабатывает события `order.created` (симуляция email/analytics).

### Схема взаимодействия:

    [Client (curl/Postman)]
           │ HTTP
           ▼
    [ API Gateway :8080 ]
           │ gRPC
           ├─────────────► [ Auth Service :50051 ] ─────► [ PostgreSQL ]
           │
           ├─────────────► [ Catalog Service :50052 ] ──► [ MongoDB ]
           │
           └─────────────► [ Order Service :50053 ] ────► [ PostgreSQL ]
                                  │
                                  │ (Publish Event)
                                  ▼
                            [ RabbitMQ ] (Fanout Exchange: events)
                                  │
                                  ▼
                        [ Notification Worker ]
                        (Email / Analytics)

## 🛠 Стек технологий

*   **Язык:** Go (1.27+)
*   **API:** REST (HTTP), gRPC (Protobuf)
*   **Базы данных:** PostgreSQL (транзакции, ACID), MongoDB (документы, агрегации)
*   **Брокер сообщений:** RabbitMQ (Fanout, DLQ)
*   **Аутентификация:** JWT (JSON Web Tokens)
*   **Логирование:** `log/slog` (структурированные логи)

## 🚀 Запуск проекта (Локально)

Для запуска вам понадобятся запущенные экземпляры PostgreSQL, MongoDB и RabbitMQ.

### 1. Требования
*   Установленный Go (версия 1.22+)
*   Установленный `protoc` (компилятор Protobuf)
*   Запустить docker-compose.yml

### 2. Подготовка Баз Данных
Перед первым запуском примените миграции (или создайте таблицы вручную) в PostgreSQL:

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
    );

### 3. Генерация gRPC кода
Если вы вносили изменения в `.proto` файлы, перегенерируйте код:

    protoc --go_out=. --go_opt=paths=source_relative \
           --go-grpc_out=. --go-grpc_opt=paths=source_relative \
           proto/*.proto

### 4. Запуск сервисов
Откройте 5 терминалов и запустите каждый сервис отдельно:

    # Терминал 1: Auth
    go run ./auth

    # Терминал 2: Catalog
    go run ./catalog

    # Терминал 3: Order
    go run ./order

    # Терминал 4: Notification
    go run ./notification

    # Терминал 5: Gateway (HTTP)
    go run ./gateway

*Примечание: В данной ветке адреса (localhost) и порты захардкожены в коде для удобства локальной отладки.*

## 🧪 Проверка работоспособности (через API Gateway)

После запуска всех сервисов, API доступен на `http://localhost:8080`.

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
    ├── docker-compose.yml
    ├── go.mod
    └── go.sum

## ✨ Реализованные паттерны
*   **API Gateway** (маршрутизация HTTP -> gRPC).
*   **Saga Pattern** (распределенная транзакция между Order и Catalog).
*   **Fanout Pub/Sub** (события RabbitMQ идут во все привязанные очереди).
*   **Graceful Shutdown** (корректное завершение по SIGINT/SIGTERM).
*   **Structured Logging** (через `slog`).
*   **Circuit Breaking (basic)** (компенсация при ошибках в Order Service).
