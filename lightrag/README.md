# LightRAG

RAG-сервис для ответов по загруженным документам. LightRAG индексирует текст в PostgreSQL, LLM и embeddings вызываются через внешние API.

## Архитектура

```text
PDF -> backend -> OCR-сервис -> shared Postgres -> LightRAG -> PostgreSQL (KV/граф/векторы)
                                                       |
                                                       +-> Cerebras / Jina AI
```

GPU не требуется. В стандартной конфигурации используются Cerebras `gpt://<folder-id>/yandexgpt-5.1/latest` и Jina AI `emb://<folder-id>/text-embeddings/latest`.

## Запуск в общем Docker Compose

Из корня проекта:

```powershell
Copy-Item .env.example .env   # если .env ещё нет
# Заполните LIGHTRAG_LLM_BINDING_API_KEY и LIGHTRAG_EMBEDDING_BINDING_API_KEY
docker compose up -d --build
```

Web UI приложения: <http://127.0.0.1:9689>
LightRAG UI: <http://127.0.0.1:19621>

## Отдельный запуск LightRAG

```powershell
cd lightrag
Copy-Item .env.docker.example .env   # если .env ещё нет
# Заполните LLM_BINDING_API_KEY и EMBEDDING_BINDING_API_KEY
docker compose up -d --build
```

## Индексация

В общем Docker Compose индексация происходит автоматически: backend-воркер отправляет результат OCR в LightRAG.

Для ручной загрузки готового текста (.md / .txt):

```powershell
cd lightrag
powershell -ExecutionPolicy Bypass -File .\ingest-via-ocr.ps1
```

Для индексации PDF напрямую (без OCR):

```powershell
cd lightrag
copy ..\path\to\file.pdf documents\
powershell -ExecutionPolicy Bypass -File .\prepare-input.ps1
powershell -ExecutionPolicy Bypass -File .\index.ps1
```

## Запрос

```powershell
powershell -ExecutionPolicy Bypass -File .\query.ps1 -Query "Что такое фракционирование изотопов углерода?"
```

## Локальный запуск без Docker

Требуется локальный PostgreSQL с расширениями `pgvector` и `age`.

```powershell
cd lightrag
powershell -ExecutionPolicy Bypass -File .\setup.ps1
# Заполните ключи и параметры подключения к Postgres в .env
powershell -ExecutionPolicy Bypass -File .\start.ps1
```

## Переменные моделей

| Переменная | Значение по умолчанию |
|---|---|
| `LLM_BINDING` | `openai` |
| `LLM_BINDING_HOST` | `https://ai.api.cloud.yandex.net/v1` |
| `LLM_MODEL` | `gpt://<folder-id>/yandexgpt-5.1/latest` |
| `EMBEDDING_BINDING` | `openai` |
| `EMBEDDING_BINDING_HOST` | `https://ai.api.cloud.yandex.net/v1` |
| `EMBEDDING_MODEL` | `emb://<folder-id>/text-embeddings/latest` |
| `EMBEDDING_DIM` | `1536` |

## Важные детали

- LightRAG 1.5.4 хранит KV, статусы документов, граф и векторы в PostgreSQL.
- Переменные `LIGHTRAG_*_STORAGE` задаются в `docker-compose.yml` и не требуют ручного изменения.
- Индекс зависит от embedding-модели и её размерности.
- После смены embedding-провайдера документы необходимо переиндексировать.
- Старые скрипты `import-ocr-results.ps1` и `ingest-via-ocr.ps1` больше не используют устаревший OCR-конвейер; `ingest-via-ocr.ps1` теперь загружает готовый текст.
