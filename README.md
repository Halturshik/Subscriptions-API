# REST API-сервис для управления онлайн-подписками пользователей

Хранение информации о подписках пользователей на различные сервисы.

## Возможности API
1. **Создание подписки** (POST `/subscriptions`)
2. **Получение списка всех подписок пользователя (активных или архивных)** (GET `/users/{user_id}/subscriptions`)
3. **Получение списка подписок пользователя на конкретный сервис (активной или архивных)** (GET `/users/{user_id}/subscriptions/{service_name}`)
4. **Обновление стоимости и/или даты окончания подписки на конкретный сервис пользователя** (PUT `/users/{user_id}/subscriptions/{service_name}`)
5. **Удаление конкретной подписки пользователя на конкретный сервис** (DELETE `/users/{user_id}/subscriptions/{service_name}`)
6. **Подсчет суммарной стоимости подписок пользователя на конкретный сервис за заданный период** (Post `/users/{user_id}/subscriptions/{service_name}/total`)

## Стек
1) Go 1.23+
2) PostgreSQL 16
3) Chi — маршрутизация HTTP-запросов
4) Swaggo/swag — автогенерация Swagger-документации
5) Godotenv — загрузка переменных окружения
6) Docker Compose — контейнеризация приложения и базы данных
7) Goose — миграции базы данных

## Запуск проекта
1. Адаптировать `.env.example` в `.env`
2. Выполнить команду:
```bash
docker-compose up --build
```

## После запуска
1. API доступно по адресу: http://localhost:8080
2. Swagger документация доступна по адресу: http://localhost:8080/swagger/index.html

Примеры запросов:
1) Post (Create):
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440001",
  "service_name": "HBO",
  "price": 50,
  "start_date": "03-2019",
  "end_date": "04-2026"
}
```

2) Put:
```json
{
  "new_price": 100,
  "new_end_date": "11-2026"
}
```

3) Delete:
```json
{
    "start_date": "03-2019"
}
```

4) Post (Total):
```json
{
    "total_from": "05-2019",
    "total_to": "10-2025"
}
```