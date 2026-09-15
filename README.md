# backend

Go-монорепозиторий бэкенда: микросервисы и общие библиотеки, собранные в один `go.work`.

## Структура

```
pkg/observability/   общая библиотека: логи, трейсы, метрики, admin-сервер
services/auth/       сервис аутентификации
deploy/helm/service/ общий Helm-чарт для любого сервиса
deploy/helm/dev-infra/ Postgres и Redis для локального кластера
Taskfile.yml         команды для локального запуска в minikube
```

## Что реализовано

### Сервис `auth`

- **Регистрация** `POST /v1/auth/register`:
  - принимает `email`, `password` (не короче 8 символов) и `role` (`athlete` или `trainer`);
  - хеширует пароль bcrypt и сохраняет неподтверждённого пользователя в Postgres. Повторная регистрация на тот же email перезаписывает запись, пока она не подтверждена; для подтверждённого email возвращается `409`;
  - генерирует 6-значный код подтверждения и кладёт его в Redis с TTL (по умолчанию 15 минут);
  - публикует событие `user.registered`. Пока это заглушка, которая только пишет в лог; Kafka не подключена;
  - отвечает `201 {"user_id": "...", "expires_in": 900}`, ошибки в формате `{"error": {"code", "message"}}`.
- **Подтверждения email и входа пока нет.**
- **Архитектура:** `domain` → `application` (use case и порты) → `infrastructure` (Postgres, Redis, bcrypt, публикация событий) и `delivery/http`.
- **Эксплуатация в Kubernetes:**
  - admin-сервер на отдельном порту поднимается первым: `/livez` отвечает сразу, `/readyz` отвечает `503`, пока не подключены Postgres и Redis, затем пингует их;
  - подключение к зависимостям ограничено общим `STARTUP_TIMEOUT`, при превышении процесс завершается с ошибкой;
  - graceful shutdown по `SIGTERM`: readiness выключается, пауза `SHUTDOWN_DRAIN_DELAY`, затем остановка серверов с таймаутом и принудительным закрытием соединений.
- **Миграции** — golang-migrate, файлы в `services/auth/migrations`.
- **Конфигурация** — только переменные окружения, разбитые по секциям в `internal/config`.
- **Unit-тесты** на бизнес-логику регистрации (`internal/application`): валидация и нормализация входных данных, порядок шагов, остановка при ошибках зависимостей, генерация кода подтверждения.

### Библиотека `pkg/observability`

- **`logger`** — `log/slog` в stdout (JSON или text), к записям автоматически добавляются `trace_id` и `span_id` из контекста.
- **`tracing`** — OpenTelemetry: OTLP/gRPC-экспортёр, ratio-сэмплирование с учётом решения вызывающего сервиса. Провайдер и W3C-propagator устанавливаются глобально.
- **`metrics`** — Prometheus: собственный реестр, Go- и process-метрики, HTTP RED-метрики по маршрутам с `trace_id` в exemplar'ах.
- **`admin`** — HTTP-сервер для `/livez`, `/readyz` и `/metrics` с управлением состоянием готовности.

### Деплой

- **Dockerfile** на каждый сервис, собирается из корня репозитория. Два образа: `app` (distroless, non-root) и `migrations`.
- **Общий Helm-чарт** `deploy/helm/service`: Deployment с пробами и security context, ConfigMap и Secret, Service, миграции Job'ом как Helm hook, опционально PDB, HPA, Ingress, ServiceMonitor. Настройки сервиса лежат в `services/<svc>/deploy/values*.yaml`.

## Локальный запуск

### Что нужно

- Docker
- [minikube](https://minikube.sigs.k8s.io/docs/start/)
- [Helm](https://helm.sh/docs/intro/install/)
- kubectl
- [Task](https://taskfile.dev/installation/)

На Windows все утилиты, кроме Docker, ставятся через winget: `Kubernetes.minikube`, `Helm.Helm`, `Kubernetes.kubectl`, `Task.Task` (например, `winget install Helm.Helm`).

### Запуск

```sh
task minikube:start                   # локальный кластер
task up                               # Postgres, Redis, сборка образов, миграции и деплой всех сервисов
task service:forward SERVICE=auth     # API на localhost:8080, admin на localhost:9090
```

`task service:forward` занимает терминал, пока работает проброс портов.

### Проверка

```sh
curl localhost:9090/readyz
curl -X POST localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"Str0ngPassw0rd!","role":"athlete"}'
curl localhost:9090/metrics
```

### Работа с кластером

```sh
task service:deploy SERVICE=auth                          # пересобрать и выкатить сервис после изменений
kubectl -n backend get pods                               # состояние pod'ов
kubectl -n backend logs deploy/auth -f                    # логи сервиса
kubectl -n backend logs job/auth-migrations               # логи упавшей миграции
kubectl -n backend exec -it dev-infra-postgres-0 -- psql -U postgres -d auth
task down                                                 # удалить всё из кластера
minikube delete                                           # удалить сам кластер
```

Трейсинг в minikube выключен: коллектора в кластере нет.

## Разработка

```sh
cd services/auth && task test   # то же для pkg/observability
cd services/auth && task lint
cd services/auth && task fmt
```
