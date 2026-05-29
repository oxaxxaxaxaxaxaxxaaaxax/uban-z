# Auth Service API

Источник контракта: `docx/auth/auth-api.yaml`.

Внутренний auth-service слушает префикс `/api`, поэтому его endpoints выглядят как `/api/auth/...`. Через API Gateway по `docx/apigateway/gateway-api.yaml` эти же auth endpoints доступны без внутреннего префикса: `/auth/register`, `/auth/login`, `/auth/me`.

Ответ пользователя всегда безопасный, без пароля:

```json
{
  "id": 1,
  "login": "john",
  "role": "student_b",
  "full_name": "John Smith"
}
```

Доступные роли для регистрации:

```text
student_b
student_m
student_a
teacher
admin
```

Ошибки возвращаются в формате:

```json
{
  "error": "error message"
}
```

## Public

### POST /api/auth/register

Создаёт пользователя. Поле `full_name` обязательно и сохраняется как ФИО пользователя.

```http
POST /api/auth/register
Content-Type: application/json

{
  "login": "john",
  "password": "123456",
  "role": "student_b",
  "full_name": "John Smith"
}
```

Ответ `200 OK`:

```json
{
  "id": 1,
  "login": "john",
  "role": "student_b",
  "full_name": "John Smith"
}
```

Возможные ошибки:

| Код | Причина |
| --- | --- |
| `400` | Некорректный JSON, пустые обязательные поля или неизвестная роль |
| `409` | Пользователь с таким `login` уже существует |
| `500` | Внутренняя ошибка сервера |

### POST /api/auth/login

Возвращает JWT. Токен содержит claims `sub`, `user_id`, `login`, `role`, `exp`; `sub` нужен booking-service для защищённых endpoints.

```http
POST /api/auth/login
Content-Type: application/json

{
  "login": "john",
  "password": "123456"
}
```

Ответ `200 OK`:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

Возможные ошибки:

| Код | Причина |
| --- | --- |
| `400` | Некорректный JSON или пустые обязательные поля |
| `401` | Неверный `login` или пароль |
| `500` | Внутренняя ошибка сервера |

## Protected

Для запроса нужен заголовок:

```http
Authorization: Bearer <token>
```

Если заголовок отсутствует, токен имеет неверный формат или подпись недействительна, сервис вернёт `401`.

### GET /api/auth/me

Возвращает текущего пользователя из JWT.

```http
GET /api/auth/me
Authorization: Bearer <token>
```

Ответ `200 OK`:

```json
{
  "id": 1,
  "login": "john",
  "role": "student_b",
  "full_name": "John Smith"
}
```

Возможные ошибки:

| Код | Причина |
| --- | --- |
| `400` | Пользователь из токена не найден |
| `401` | JWT не передан или недействителен |
| `500` | Внутренняя ошибка сервера |

## Removed

Auth API больше не поддерживает user-management endpoints: `/api/users`, `/api/users/me`, `/api/users/{id}` и операции редактирования или удаления пользователя. Для фронтенда актуальны только регистрация, вход и получение текущего пользователя.
