# LightRAG + OCR

RAG для вопросов по PDF-документам. OCR преобразует PDF в LaTeX, LightRAG индексирует текст в PostgreSQL, а LLM и embeddings вызываются через внешние API.

## Архитектура

```text
PDF -> OCR API :8000 -> .tex -> LightRAG :9621 -> PostgreSQL
                                  |              |
                                  +-> Cerebras --+-> ответы
                                  +-> Jina AI ---+-> embeddings
```

GPU не требуется. В стандартной конфигурации используются Cerebras `gpt-oss-120b` и Jina AI `jina-embeddings-v3`.

## Запуск в общем Docker Compose

Из корня проекта:

```powershell
Copy-Item .env.example .env   # если .env ещё нет
# Заполните LIGHTRAG_LLM_BINDING_API_KEY и LIGHTRAG_EMBEDDING_BINDING_API_KEY
docker compose up -d --build
```

Web UI: <http://127.0.0.1:9621>

## Отдельный запуск LightRAG

```powershell
cd lightrag
Copy-Item .env.docker.example .env   # если .env ещё нет
# Заполните LLM_BINDING_API_KEY и EMBEDDING_BINDING_API_KEY
docker compose up -d --build
```

## Индексация через OCR

Полный пайплайн PDF -> LaTeX -> LightRAG:

```powershell
cd lightrag
powershell -ExecutionPolicy Bypass -File .\ingest-via-ocr.ps1
```

Импорт готовых результатов OCR:

```powershell
powershell -ExecutionPolicy Bypass -File .\import-ocr-results.ps1 -TriggerScan
```

## Запрос

```powershell
powershell -ExecutionPolicy Bypass -File .\query.ps1 -Query "Что такое фракционирование изотопов углерода?"
```

## Локальный запуск без Docker

```powershell
cd lightrag
powershell -ExecutionPolicy Bypass -File .\setup.ps1
# Заполните ключи в .env
powershell -ExecutionPolicy Bypass -File .\start.ps1
```

## Переменные моделей

| Переменная | Значение по умолчанию |
|---|---|
| `LLM_BINDING` | `openai` |
| `LLM_BINDING_HOST` | `https://api.cerebras.ai/v1` |
| `LLM_MODEL` | `gpt-oss-120b` |
| `EMBEDDING_BINDING` | `jina` |
| `EMBEDDING_BINDING_HOST` | `https://api.jina.ai/v1/embeddings` |
| `EMBEDDING_MODEL` | `jina-embeddings-v3` |
| `EMBEDDING_DIM` | `1024` |

## Важные детали

- LightRAG 1.5.4 принимает `.tex` напрямую.
- `import-ocr-results.ps1` удаляет дубликаты по SHA256.
- Индекс зависит от embedding-модели и её размерности.
- После смены embedding-провайдера документы необходимо переиндексировать.
- Первый OCR-запуск может загружать модели распознавания несколько минут; это отдельная CPU-нагрузка и не связано с LLM.