# L0_DemoServise

**Демонстрационный сервис** на Go с интеграцией PostgreSQL, Kafka и реализацией LRU.

---

## Содержание

* [Описание](#описание)
* [Ключевые возможности](#ключевые-возможности)
* [Быстрый старт (Docker Compose)](#быстрый-старт-docker-compose)
* [Переменные окружения](#переменные-окружения)
* [Архитектура проекта](#архитектура-проекта)
* [Запуск тестов](#запуск-тестов)

---

## Описание

`L0_DemoServise` — Это Демонстрационный сервис, который представляет собой HTTP-сервер на Go. Этот сервис принимает сообщения из Kafka и сохраняет их в Сache и БД (PostgreSQL), также реализован UI, который позволяет получить данные о заказе по его UID в JSON формате, путём отправление HTTP запроса на сервер

Цель этого README — дать готовые инструкции для запуска, разработки и тестирования сервиса.

---

## Ключевые возможности

* Подключение к PostgreSQL и использование этой СУБД
* Интеграция с Kafka (producer/consumer)
* Настраиваемый кеш (Cache capacity задаётся через конфигурационный файл)
Cache реализован опираясь на алгоритм LRU (Last Reasent Use)
* Скрипты для инициализации БД в `scripts/db`

## Быстрый старт (Docker Compose)

1. Клонируйте репозиторий:

```bash
git clone https://github.com/AlekseyZapadovnikov/L0_DemoServise.git
cd L0_DemoServise
```

2. При необходимости отредактируйте `.env` (порты, пароли и т.д.).

3. Поднимите окружение:

```bash
docker compose up -d
```

---

## Переменные окружения

В репозитории присутствует файл `.evn.example` — используйте его как шаблон. Ниже — типичный набор переменных (приведено для примера):

```env
DB_USER=order_user
DB_PASSWORD=demo
DB_NAME=orders_db
DB_PORT=5432

```

Подставьте реальные значения перед запуском приложения.


## Архитектура проекта

Структура папок (основные каталоги):

```
/
|-- cmd/
|   |-- helpCMD/
|   |   |-- model.json
|   |   `-- secondMain.go
|   `-- main.go
|-- config/
|   |-- config.go
|   `-- config.json
|-- internal/
|   |-- broker/
|   |   `-- consumer.go
|   |-- entity/
|   |   `-- models.go
|   |-- server/
|   |   |-- templates/
|   |   |   `-- homePage.html
|   |   `-- httpServer.go
|   |-- service/
|   |   |-- tests/
|   |   |   `-- testData.json
|   |   |-- interfaces.go
|   |   |-- priorityQueue.go
|   |   |-- savePriorityQueue.go
|   |   |-- service_test.go
|   |   `-- service.go
|   `-- storage/
|       |-- storage_test.go
|       `-- storage.go
|-- scripts/
|   `-- db/
|       |-- init_role.sql
|       `-- init_tables.sql
|-- .env
|-- .env.example
|-- .gitignore
|-- docker-compose.yml
|-- go.mod
|-- go.sum
`-- README.md
```

Модули внутри `internal/`:

* `internal/server` — HTTP-server
* `internal/service` — бизнес-логика (Cache реализован чарез map с sync.Mutex{} и LRU)
* `internal/storage` — логика работы с БД
* `internal/broker` — Kafka consumer (принимает сообщения из Kafka и сохраняет в Cache и БД)

---

## Запуск тестов

```bash
go test ./... -v
```
* покрытие тестами - coverage: 42.7% of statements
---

## Контакты

* telegram - https://t.me/w_st3r
* gmail - Zapadovnikov145@gmail.com

---
