# API Documentation

## 🔓 Аутентификация

### Регистрация

```http
POST /api/auth/register
Content-Type: application/json

{
  "login": "john",
  "password": "123456",
  "role": "user"
}
```

---

### Логин

```http
POST /api/auth/login
Content-Type: application/json

{
  "login": "john",
  "password": "123456"
}
```

**Ответ:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

---

## 🔐 Защищённые эндпоинты

> Во **всех** запросах ниже обязательно передавать заголовок:
>
> ```
> Authorization: Bearer <token>
> ```

---

### Получить всех пользователей

```http
GET /api/users
Authorization: Bearer <token>
```

---

### Получить пользователя по ID

```http
GET /api/users/1
Authorization: Bearer <token>
```

---

### Обновить пользователя

```http
PUT /api/users/1
Authorization: Bearer <token>
Content-Type: application/json

{
  "login": "new_login",
  "password": "new_password",
  "role": "admin"
}
```

---

### Удалить пользователя

```http
DELETE /api/users/1
Authorization: Bearer <token>
```
