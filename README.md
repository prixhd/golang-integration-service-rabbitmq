# Go Integration Service

HTTP-сервис для асинхронного выполнения пользовательского кода с использованием Go, RabbitMQ, PostgreSQL, Redis и Docker.

Проект реализован в рамках обучения Go-разработке и демонстрирует взаимодействие нескольких сервисов, работу с очередями сообщений, базами данных, контейнеризацию и мониторинг.

## Возможности

- Регистрация и авторизация пользователей.
- Хранение пользователей и задач в PostgreSQL.
- Хранение пользовательских сессий в Redis.
- Асинхронная обработка задач через RabbitMQ.
- Выполнение пользовательского кода в отдельных Docker-контейнерах.
- Получение статуса задачи и результата выполнения.
- Автоматическое применение SQL-миграций.
- Сбор метрик через Prometheus.
- Визуализация метрик в Grafana.
- Документация HTTP API через Swagger.

## Архитектура

```text
                    HTTP Client
                         |
                         v
                    HTTP Server
                    /    |     \
                   /     |      \
                  v      v       v
             PostgreSQL Redis  RabbitMQ
             users/tasks       |
                               v
                          CodeProcessor
                               |
                               v
                          Docker Runner
                               |
                               v
                         POST /commit
                               |
                               v
                           HTTP Server


HTTP Server ----\
                 >---- Prometheus ---- Grafana
CodeProcessor --/
```

### HTTP Server

Обрабатывает HTTP-запросы, регистрирует пользователей, создаёт задачи и предоставляет результаты выполнения.

Пользователи и задачи сохраняются в PostgreSQL, а сессии — в Redis.

### RabbitMQ

Используется для асинхронной передачи задач от HTTP-сервера к CodeProcessor.

### CodeProcessor

Получает задачи из RabbitMQ и запускает пользовательский код в отдельных Docker-контейнерах.

После выполнения отправляет результат в HTTP-сервер через защищённый эндпоинт `/commit`.

Поддерживаемые трансляторы:

| Транслятор | Язык |
|------------|------|
| `python3` | Python 3 |
| `gcc` | C |
| `clang` | C |
| `g++` | C++ |
| `clang++` | C++ |

### Transactional Outbox

Чтобы не терять задачи при недоступности RabbitMQ, публикация сообщений выполняется через таблицу outbox:

1. При создании задачи сервер в одной транзакции PostgreSQL сохраняет задачу и событие в таблицу outbox.
2. Фоновый воркер (`internal/outbox`) периодически выбирает неопубликованные события.
3. Воркер публикует их в RabbitMQ и помечает событие как отправленное.

Гарантия доставки: at-least-once. Если воркер упал после публикации, но до пометки, событие будет отправлено повторно, поэтому потребитель должен быть идемпотентным.

### PostgreSQL

Используется для постоянного хранения пользователей, задач, статусов и результатов выполнения.

Для управления схемой базы данных применяется `sql-migrate`.

### Redis

Используется для хранения пользовательских сессий.

### Prometheus и Grafana

Prometheus собирает метрики HTTP-сервера и CodeProcessor.

Grafana позволяет визуализировать количество запросов, время их обработки и использование различных трансляторов.

## Технологии

- Go
- PostgreSQL
- Redis
- RabbitMQ
- Docker и Docker Compose
- Prometheus
- Grafana
- sql-migrate
- Swagger
- GitHub Actions

## HTTP API

| Метод | Маршрут | Описание |
|-------|---------|----------|
| POST | `/register` | Регистрация пользователя |
| POST | `/login` | Авторизация |
| POST | `/task` | Создание задачи |
| GET | `/status/{id}` | Получение статуса задачи |
| GET | `/result/{id}` | Получение результата |
| POST | `/commit` | Сохранение результата CodeProcessor |
| GET | `/metrics` | Метрики HTTP-сервера |

Маршруты `/task`, `/status/{id}` и `/result/{id}` требуют авторизации:

```http
Authorization: Bearer <token>
```

Эндпоинт `/commit` предназначен для внутреннего взаимодействия CodeProcessor с HTTP-сервером и защищён отдельным секретом.

### Пример создания задачи

```http
POST /task
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{
  "translator": "python3",
  "code": "print('Hello, World!')"
}
```

Ответ:

```json
{
  "task_id": "f84d9924-7223-4670-befb-873363f1633a"
}
```

После выполнения задачи результат можно получить через:

```http
GET /result/{id}
```

## Быстрый запуск

### Требования

Для запуска необходимы:

- Docker
- Docker Compose
- Make

Go необязателен для запуска через Docker Compose.

### Запуск приложения

Клонировать репозиторий:

```bash
git clone https://github.com/prixhd/golang-integration-service-rabbitmq.git
cd golang-integration-service-rabbitmq
```

Для получения реализации четвёртой части перейти в ветку:

```bash
git switch feature/part4
```

Запустить приложение:

```bash
make up
```

Команда собирает Docker-образы и запускает необходимые сервисы.

SQL-миграции применяются автоматически после запуска PostgreSQL.

### Остановка

```bash
make down
```

Данные PostgreSQL, Redis, Prometheus и Grafana сохраняются в Docker volumes.

Для просмотра логов:

```bash
make logs
```

## Миграции PostgreSQL

Для миграций используется библиотека [sql-migrate](https://github.com/rubenv/sql-migrate).

SQL-файлы находятся в каталоге:

```text
migrations/
```

При запуске Docker Compose отдельный контейнер `migrate` выполняет:

```bash
sql-migrate up
```

HTTP-сервер запускается после успешного завершения миграций.

Для ручной работы с миграциями можно установить CLI:

```bash
go install github.com/rubenv/sql-migrate/...@latest
```

Проверка состояния:

```bash
sql-migrate status
```

Применение:

```bash
sql-migrate up
```

Откат последней миграции:

```bash
sql-migrate down
```

Команды ручного запуска используют окружение `development` из `dbconfig.yml` и предполагают доступность PostgreSQL на `localhost:5432`.

## Мониторинг

### Prometheus

Веб-интерфейс:

http://localhost:9090

Prometheus собирает метрики:

- HTTP Server: `server:8000/metrics`
- CodeProcessor: `processor:9091/metrics`

Основные метрики:

```text
http_requests_total
http_request_duration_seconds

codeprocessor_tasks_total
codeprocessor_task_duration_seconds
```

Метрики CodeProcessor содержат метку `translator`, позволяющую анализировать использование разных языков и компиляторов.

### Grafana

Веб-интерфейс:

http://localhost:3000

Учётные данные для локального запуска:

```text
Логин: admin
Пароль: admin
```

Для подключения Prometheus в качестве Data Source используется адрес:

```text
http://prometheus:9090
```

Пример PromQL-запроса для количества обработанных задач:

```promql
codeprocessor_tasks_total
```

Среднее время выполнения:

```promql
codeprocessor_task_duration_seconds_sum
/
codeprocessor_task_duration_seconds_count
```

## Дополнительные интерфейсы

| Сервис | Адрес |
|--------|-------|
| HTTP API | http://localhost:8000 |
| Swagger | http://localhost:8000/swagger/index.html |
| RabbitMQ Management | http://localhost:15672 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |

RabbitMQ Management использует локальные учётные данные `guest / guest`.

## Тестирование

Запуск Go-тестов:

```bash
make test
```

Или:

```bash
go test ./...
```

В проекте присутствуют тесты Docker Runner и HTTP middleware для метрик.

Интеграционные тесты Docker Runner требуют доступного Docker Engine и образа `code-runner:latest`, который собирается командой `make up`.

## Безопасность и ограничения

Проект предназначен для обучения и локального запуска.

Для ограничения выполнения пользовательского кода используются отдельные Docker-контейнеры с ограничениями памяти, CPU, количества процессов и сетевого доступа.

При этом CodeProcessor использует Docker socket:

```text
/var/run/docker.sock
```

Такой доступ предоставляет широкие привилегии управления Docker-хостом.

Поэтому текущая конфигурация не предназначена для публичного выполнения недоверенного кода в production.

Пароли и секреты в Docker Compose также являются демонстрационными и должны быть заменены при реальном развёртывании.
