# Auth Service API

Базовый префикс сервиса: `/api`.

Все ответы с пользователем возвращают безопасный объект без пароля:

```json
{
  "id": 1,
  "login": "john",
  "role": "student_b"
}
```

Доступные роли:

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

Создаёт пользователя.

```http
POST /api/auth/register
Content-Type: application/json

{
  "login": "john",
  "password": "123456",
  "role": "student_b"
}
```

Ответ `200 OK`:

```json
{
  "id": 1,
  "login": "john",
  "role": "student_b"
}
```

Возможные ошибки:

| Код | Причина |
| --- | --- |
| `400` | Некорректный JSON или неизвестная роль |
| `409` | Пользователь с таким `login` уже существует |
| `500` | Внутренняя ошибка сервера |

### POST /api/auth/login

Возвращает JWT. В токене есть claims `user_id`, `login`, `role`.

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
| `400` | Некорректный JSON |
| `401` | Неверный `login` или пароль |
| `500` | Внутренняя ошибка сервера |

## Protected

Для всех запросов ниже нужен заголовок:

```http
Authorization: Bearer <token>
```

Если заголовок отсутствует, токен имеет неверный формат или подпись недействительна, сервис вернёт `401`.

### GET /api/users/me

Возвращает текущего пользователя из JWT.

```http
GET /api/users/me
Authorization: Bearer <token>
```

Ответ `200 OK`:

```json
{
  "id": 1,
  "login": "john",
  "role": "student_b"
}
```

Возможные ошибки:

| Код | Причина |
| --- | --- |
| `401` | JWT не передан или недействителен |
| `404` | Пользователь из токена не найден |
| `500` | Внутренняя ошибка сервера |

### PUT /api/users/me

Обновляет текущего пользователя. Пользователь может менять только `login` и `password`; роль через этот endpoint менять нельзя.

```http
PUT /api/users/me
Authorization: Bearer <token>
Content-Type: application/json

{
  "login": "john_new",
  "password": "new_password"
}
```

Все поля опциональны. Если поле не передано, оно не изменяется.

Ответ `200 OK`:

```json
{
  "id": 1,
  "login": "john_new",
  "role": "student_b"
}
```

Возможные ошибки:

| Код | Причина |
| --- | --- |
| `400` | Некорректный JSON |
| `401` | JWT не передан или недействителен |
| `403` | Передано поле `role` |
| `404` | Пользователь из токена не найден |
| `409` | Новый `login` уже занят |
| `500` | Внутренняя ошибка сервера |

### DELETE /api/users/me

Удаляет текущего пользователя.

```http
DELETE /api/users/me
Authorization: Bearer <token>
```

Ответ `204 No Content`, тело ответа пустое.

Возможные ошибки:

| Код | Причина |
| --- | --- |
| `401` | JWT не передан или недействителен |
| `404` | Пользователь из токена не найден |
| `500` | Внутренняя ошибка сервера |

## Admin Only

Для запросов ниже нужен JWT пользователя с ролью `admin`.

### GET /api/users

Возвращает список пользователей.

```http
GET /api/users
Authorization: Bearer <admin-token>
```

Ответ `200 OK`:

```json
[
  {
    "id": 1,
    "login": "john",
    "role": "student_b"
  },
  {
    "id": 2,
    "login": "admin",
    "role": "admin"
  }
]
```

Возможные ошибки:

| Код | Причина |
| --- | --- |
| `401` | JWT не передан или недействителен |
| `403` | Роль пользователя не `admin` |
| `500` | Внутренняя ошибка сервера |

### GET /api/users/{id}

Возвращает пользователя по ID.

```http
GET /api/users/1
Authorization: Bearer <admin-token>
```

Ответ `200 OK`:

```json
{
  "id": 1,
  "login": "john",
  "role": "student_b"
}
```

Возможные ошибки:

| Код | Причина |
| --- | --- |
| `400` | Некорректный `id` |
| `401` | JWT не передан или недействителен |
| `403` | Роль пользователя не `admin` |
| `404` | Пользователь не найден |
| `500` | Внутренняя ошибка сервера |

### PUT /api/users/{id}

Обновляет пользователя по ID. Доступно только администратору. Через этот endpoint можно менять `login`, `password` и `role`.

```http
PUT /api/users/1
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "login": "john_admin_edit",
  "password": "new_password",
  "role": "teacher"
}
```

Все поля опциональны. Если поле не передано, оно не изменяется.

Ответ `200 OK`:

```json
{
  "id": 1,
  "login": "john_admin_edit",
  "role": "teacher"
}
```

Возможные ошибки:

| Код | Причина |
| --- | --- |
| `400` | Некорректный JSON, `id` или роль |
| `401` | JWT не передан или недействителен |
| `403` | Роль пользователя не `admin` |
| `404` | Пользователь не найден |
| `409` | Новый `login` уже занят |
| `500` | Внутренняя ошибка сервера |

### DELETE /api/users/{id}

Удаляет пользователя по ID. Доступно только администратору.

```http
DELETE /api/users/1
Authorization: Bearer <admin-token>
```

Ответ `204 No Content`, тело ответа пустое.

Возможные ошибки:

| Код | Причина |
| --- | --- |
| `400` | Некорректный `id` |
| `401` | JWT не передан или недействителен |
| `403` | Роль пользователя не `admin` |
| `404` | Пользователь не найден |
| `500` | Внутренняя ошибка сервера |
