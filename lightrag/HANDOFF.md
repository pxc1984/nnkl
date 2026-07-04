# Контекст LightRAG

## Текущая архитектура

- LightRAG `1.5.4`.
- PostgreSQL: KV, статусы документов, граф и векторы.
- LLM: Cerebras API, `gpt-oss-120b`.
- Embeddings: Jina AI API, `jina-embeddings-v3`, размерность `1024`.
- OCR: отдельный сервис с CPU-конфигурацией по умолчанию.

Стек не запускает локальные LLM/embedding-модели и не резервирует GPU. Для работы нужны `LIGHTRAG_LLM_BINDING_API_KEY` и `LIGHTRAG_EMBEDDING_BINDING_API_KEY` в корневом `.env`.

## Запуск

```powershell
Copy-Item .env.example .env
# Заполнить оба API-ключа
docker compose up -d --build
```

Приложение: <http://127.0.0.1:9689>

LightRAG: <http://127.0.0.1:19621>

## Индексация

```powershell
cd lightrag
powershell -ExecutionPolicy Bypass -File .\ingest-via-ocr.ps1
```

## Проверка

1. `docker compose config` завершается без ошибок.
2. `docker compose ps` показывает healthy для `db` и `lightrag`.
3. Документы переходят в статус `processed`.
4. Ответы используют сведения из проиндексированных PDF.

При смене embedding-модели или размерности требуется новый workspace либо очистка векторных данных и полная переиндексация.