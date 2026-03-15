# ТЕХНИЧЕСКОЕ ЗАДАНИЕ
## Система управления микросервисными приложениями (MAMS)

---

## 1. Цель

Разработать систему управления микросервисными приложениями и визуализации процессов их эксплуатации.

---

## 2. Архитектура

![img.png](img.png)

### Развёртывание

- **MAMS Core**: виртуалка (не в Kubernetes)
- **Kubernetes**: один кластер, организации разделяются по namespace

---

## 3. Компоненты системы

| Компонент   | Технология                 | Назначение                        |
|-------------|----------------------------|-----------------------------------|
| API Gateway | Traefik                    | Единая точка входа, маршрутизация |
| Backend     | Go 1.24                    | REST API, интеграция с K8s        |
| Frontend    | React                      | Веб-интерфейс                     |
| База данных | PostgreSQL 15+             | Пользователи, организации, сервисы, релизы |
| Логи        | MongoDB 8.1                | Логи приложений                   |
| Мониторинг  | Prometheus 3.8.1 + Grafana | Метрики и дашборды                |
| CI/CD       | GitHub Actions             | Деплой приложений                 |

---

## 4. Функциональные модули

### 4.1 Main Module (Управление сервисами)

- Список сервисов организации
- Информация о сервисе: название, описание, версия, тип
- Конфигурация из `app.yaml` (загружается из Git-репозитория)
- Поля сервиса:
  - `name` — название
  - `description` — описание
  - `type` — `business` | `composition`
  - `version` — текущая версия
  - `test_coverage` — тестовое покрытие (%)
  - `pii_sensitive` — работа с ПДН
  - `responsible_team_url` — ссылка на чат команды
  - `importance` — важность (1-10)

### 4.2 Logs Module

- Просмотр логов в реальном времени
- Фильтрация по уровню, времени, ключевым словам
- Агрегация логов
- Хранение в MongoDB
- **Доступ**: Developer, Service Owner

### 4.3 Metrics Module

- Запрос метрик из Prometheus
- Фильтрация по периоду, перцентилям
- Встраивание Grafana-дашбордов
- **Доступ**: Developer, Service Owner (только чтение)

### 4.4 Contract Module

- Визуализация API-контрактов
- Парсинг `project.proto` из Git-репозитория
- Интерактивное отображение эндпоинтов
- **Доступ**: все авторизованные пользователи

### 4.5 Release Module (Деплой)

- **Стратегии деплоя**:
  - `Rolling` — постепенная замена подов
  - `Recreate` — полное пересоздание
  - `Canary` — частичное развёртывание с процентом трафика

- **Параметры**:
  - Branch — выбор Git-ветки
  - Environment — `dev` | `staging` | `prod`

- **Rollback** — откат к предыдущей версии

- **Доступ**: Developer, Service Owner

---

## 5. Модель данных

> **Примечание:** PostgreSQL используется для всех сущностей, кроме логов. Логи хранятся в MongoDB.

### 5.1 Organization
```json
{
  "id": "string",
  "name": "string",
  "created_at": "timestamp"
}
```

### 5.2 User
```json
{
  "id": "string",
  "email": "string",
  "password_hash": "string",
  "role": "developer | service_owner | observer",
  "organization_id": "string",
  "created_at": "timestamp"
}
```

### 5.3 Service
```json
{
  "id": "string",
  "organization_id": "string",
  "namespace": "string",
  "name": "string",
  "description": "string",
  "type": "business | composition",
  "version": "string",
  "test_coverage": "int",
  "pii_sensitive": "bool",
  "responsible_team_url": "string",
  "importance": "int",
  "repository_url": "string",
  "app_config": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

### 5.4 Release
```json
{
  "id": "string",
  "service_id": "string",
  "version": "string",
  "branch": "string",
  "environment": "dev | staging | prod",
  "strategy": "rolling | recreate | canary",
  "status": "pending | in_progress | success | failed",
  "author": "string",
  "deployed_at": "timestamp"
}
```

### 5.5 Log (MongoDB)
```json
{
  "id": "ObjectId",
  "service_id": "string",
  "level": "debug | info | warn | error",
  "message": "string",
  "timestamp": "datetime",
  "metadata": "object"
}
```

---

## 6. Ролевая модель (RBAC)

| Функция | Developer | Service Owner | Observer |
|---------|-----------|---------------|----------|
| Просмотр сервисов | ✓ | ✓ | ✓ |
| Просмотр логов | ✓ | ✓ | ✗ |
| Просмотр метрик | ✓ | ✓ | ✗ |
| Просмотр контрактов | ✓ | ✓ | ✓ |
| Деплой (Release) | ✓ | ✓ | ✗ |
| Управление сервисом | ✗ | ✓ | ✗ |
| Управление пользователями | ✗ | ✓ | ✗ |

---

## 7. Аутентификация и авторизация

- **Аутентификация**: БД пользователей + JWT токены
- **Авторизация**: RBAC на уровне API и UI
- **Сессия**: JWT с expiration

---

## 8. Интеграции

### 8.1 Kubernetes
- Взаимодействие через `github.com/kubernetes/client-go`
- Каждая организация — отдельный namespace
- Операции: deploy, rollback, scale, status

### 8.2 GitHub
- Получение `app.yaml` и `project.proto` из репозитория
- GitHub API для получения содержимого файлов
- GitHub Actions workflow для деплоя

### 8.3 Prometheus
- Запрос метрик через HTTP API
- Встраивание Grafana-дашбордов по URL

---

## 9. API Endpoints

### Auth
- `POST /api/auth/register` — регистрация
- `POST /api/auth/login` — вход
- `POST /api/auth/refresh` — обновление токена

### Organizations
- `GET /api/organizations` — список организаций
- `POST /api/organizations` — создание организации
- `GET /api/organizations/:id` — детали организации

### Services
- `GET /api/services` — список сервисов организации
- `POST /api/services` — создание сервиса
- `GET /api/services/:id` — детали сервиса
- `PUT /api/services/:id` — обновление сервиса
- `DELETE /api/services/:id` — удаление сервиса

### Logs
- `GET /api/services/:id/logs` — получение логов
- `GET /api/services/:id/logs/stream` — логи в реальном времени (WS)

### Metrics
- `GET /api/services/:id/metrics` — метрики сервиса
- `GET /api/grafana/dashboard/:uid` — Grafana дашборд

### Contracts
- `GET /api/services/:id/contracts` — API контракты сервиса

### Releases
- `GET /api/services/:id/releases` — история релизов
- `POST /api/services/:id/deploy` — запуск деплоя
- `POST /api/services/:id/rollback` — откат

### Users
- `GET /api/users` — список пользователей организации
- `POST /api/users` — создание пользователя
- `PUT /api/users/:id` — обновление пользователя
- `DELETE /api/users/:id` — удаление пользователя

---

## 10. CI/CD (GitHub Actions)

### Workflow для деплоя сервиса

```yaml
name: Deploy
on:
  workflow_dispatch:
    inputs:
      environment:
        required: true
        type: choice
        options: [dev, staging, prod]
      strategy:
        type: choice
        options: [rolling, recreate, canary]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to ${{ github.event.inputs.environment }}
        run: |
          # kubectl apply -f ./k8s/
```

### Trigger из MAMS
- MAMS отправляет trigger в GitHub API
- GitHub Actions запускает workflow с переданными параметрами

---

## 11. Критерии приёмки

1. Система разворачивается на виртуалке
2. JWT-аутентификация работает
3. 3 роли работают согласно матрице доступа
4. CRUD сервисов работает
5. Логи отображаются в реальном времени
6. Метрики загружаются из Prometheus
7. Контракты парсятся из `project.proto`
8. Деплой в Kubernetes (rolling/recreate/canary)
9. Rollback работает
10. GitHub Actions интеграция для деплоя
11. Поддержка нескольких организаций (namespace-разделение)
12. Логи сохраняются в MongoDB


---

## 13. Версии

| Компонент | Версия |
|-----------|--------|
| Go | 1.24 |
| PostgreSQL | 15+ |
| MongoDB | 8.1 |
| Prometheus | 3.8.1 |
| Traefik | latest |
| React | 18.x |

---
