## быстрый старт

```bash
cp .env.example .env
```

```bash
docker compose up --build
```

```text
http://localhost:9689
```

LightRAG API и Web UI:

```text
http://localhost:9621
```

При первом запуске Ollama скачает локальные модели. После запуска сервисов
индексация документов выполняется отдельно:

```bash
docker compose --profile tools run --rm lightrag-indexer
```

Перед полной индексацией проверьте GPU командой
`docker compose exec ollama ollama ps` во время пробного запроса.

Повторный запуск команды безопасно инициирует сканирование каталога документов.
Индекс и модели сохраняются в Docker volumes `lightrag_data` и `ollama_data`.

## гайд как разрабатывать

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

## тесты qa

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

## окружение

```env
POSTGRES_USER=admin
POSTGRES_PASSWORD=admin
POSTGRES_DB=db
POSTGRES_HOST=db
POSTGRES_PORT=5432
POSTGRES_SSLMODE=disable
HOST=0.0.0.0
PORT=8080
LOG_LEVEL=INFO
GIN_MODE=release
STORE_BACKEND=memory
LIGHTRAG_URL=http://lightrag:9621
AUTH_SECRET=change-me-in-production
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
```

виды бекенд сторов:

- `STORE_BACKEND=postgres`
- `STORE_BACKEND=memory`

опционально для фронтенда `frontend/.env`:

```env
API_URL=
```
