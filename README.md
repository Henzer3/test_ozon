# Сервисы постов и авторизации

- `post-service` — GraphQL-сервис постов и комментариев;
- `sso` — SSO-сервис;
- `pkg` — общий Go-модуль с protobuf-контрактами.
- `tests` - интеграционные тесты для sso и post-service

для удобства добавлен makefile с командами:

Генерация .proto файлов и graphql:

```bash
make generate
```

## Запуск

```bash
make up
```
## почистить контейнеры
```bash
make clean
```

## Запуск интеграционных тестов, которые проверяют такие сценарии:
- создание постов разными клиентами
- комментарии от разных клиентов
- запрос постов и комментариев
- пагинация
- регистрация
- получение токена

```bash
make test
```

Также откроется GraphQL Playground  по адресу `http://localhost:8080/`, endpoint API —
`http://localhost:8080/query`, в котором можно самому отправлять запросы.

Примеры запросов:

Регистрация:

curl -i \
  -X POST http://localhost:28080/api/register \
  -H "Content-Type: application/json" \
  -d '{"name":"test@example.com","password":"password123"}'

Получение токена:

curl -i \
  -X POST http://localhost:28080/api/login \
  -H "Content-Type: application/json" \
  -d '{"name":"test@example.com","password":"password123"}'

копируем токен и уже на GraphQL Playground вставляем в Headers:
{
  "Authorization": "Token ваш_токен"
}

Токен действует 1 час, можно не волноваться, что истечет.


создание поста: 
mutation {
  createPost(
    input: {
      title: "Первый пост"
      content: "Содержимое первого поста"
      commentsEnabled: true
    }
  ) {
    id
    authorID
    title
    content
    commentsEnabled
  }
}


создание комментария:
mutation {
  createComment(
    input: {
      postID: "1"
      content: "первый комментарий"
    }
  ) {
    id
    postID
    parentID
    content
  }
}

создание ответа:
mutation {
  createComment(
    input: {
      postID: "1"
      parentID: "1"
      content: "Ответ на комментарий"
    }
  ) {
    id
    parentID
    content
  }
}

просмотр поста и комментариев:

first - сколько корневых комментариев выдать, по умолчанию 20

query {
  post(id: "1") {
    id
    title
    comments(first: 20) {
      nodes {
        id
        content
        replies {
          id
          content
        }
      }
      pageInfo {
        endCursor
        hasNextPage
      }
    }
  }
}

следущая пачка комментариев:

after - метка для пагинации, берется из ответа на прошлый запрос

query {
  post(id: "1") {
    id
    title
    comments(first: 20, after: "MjA=") {
      nodes {
        id
        content
        replies {
          id
          content
        }
      }
      pageInfo {
        endCursor
        hasNextPage
      }
    }
  }
}



