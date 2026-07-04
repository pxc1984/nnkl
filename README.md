## Быстрый старт

Создайте `.env` из шаблона и заполните ключи внешних API:

```bash
cp .env.example .env
```

Обязательные ключи:

- `LIGHTRAG_LLM_BINDING_API_KEY` — Cerebras API;
- `LIGHTRAG_EMBEDDING_BINDING_API_KEY` — Jina AI API.

```bash
docker compose up --build
```

Основное приложение: <http://localhost:9689>

LightRAG API и Web UI: <http://localhost:9621>

LLM и embeddings выполняются внешними API. Стек не требует GPU и подходит для CPU-сервера.

## Разработка

Backend:

```bash
cd backend
go run .
```

Frontend:

```bash
cd frontend
pnpm install
pnpm run dev
```

## Тесты

Backend:

```bash
cd backend
go test ./...
```

Frontend:

```bash
cd frontend
pnpm run check
pnpm run lint
pnpm run build
```

## Хранилище

Доступные значения `STORE_BACKEND`:

- `postgres`
- `memory`

Для фронтенда можно создать `frontend/.env`:

```env
API_URL=
```