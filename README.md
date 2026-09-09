# Сервисы постов и авторизации

Репозиторий организован как monorepo с независимыми Go-модулями:

- `post-service` — GraphQL-сервис постов и комментариев;
- `sso` — SSO-сервис;
- `pkg` — общий Go-модуль с protobuf-контрактами.

Корневой `go.work` объединяет модули для локальной разработки.

Код protobuf и gRPC повторно генерируется командой:

```bash
make generate-proto
```

## Запуск

```bash
make run-post
make run-sso
```

GraphQL Playground откроется по адресу `http://localhost:8080/`, endpoint API —
`http://localhost:8080/query`. Заглушка SSO предоставляет health check по адресу
`http://localhost:8081/health`. Порты можно изменить переменной окружения `PORT`.

## Генерация GraphQL-кода

После изменения `post-service/internal/graphql/schema.graphqls` выполните:

```bash
make generate
```

Проверка компиляции и тестов:

```bash
make tests
```

## Docker

Оба сервиса собираются собственными Dockerfile:

```bash
docker compose up --build
```
