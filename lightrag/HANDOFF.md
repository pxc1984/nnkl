# Контекст LightRAG

## Текущая архитектура

- LightRAG `1.5.4`.
- PostgreSQL: KV, статусы документов, граф и векторы.
- LLM: Yandex AI Studio API, `YandexGPT 5.1`.
- Embeddings: Yandex AI Studio API, `text-embeddings`, размерность `1536`, batch size `1`.
- OCR: отдельный сервис, результат OCR backend-воркер отправляет в LightRAG автоматически.

Стек не запускает локальные LLM/embedding-модели и не резервирует GPU. Для работы нужны API-ключи в `.env`.

## Запуск

```powershell
Copy-Item .env.example .env
# Заполнить оба API-ключа
docker compose up -d --build
```

Приложение: <http://127.0.0.1:9689>

LightRAG: <http://127.0.0.1:19621>

## Индексация

Автоматическая: backend-воркер вызывает `POST /documents/text` после OCR или прямого извлечения текста.

Для ручной загрузки текста:

```powershell
cd lightrag
powershell -ExecutionPolicy Bypass -File .\ingest-via-ocr.ps1
```

## Проверка

1. `docker compose config` завершается без ошибок.
2. `docker compose ps` показывает healthy для `db` и `lightrag`.
3. Документы переходят в статус `processed`.
4. Ответы используют сведения из проиндексированных документов.

При смене embedding-модели или размерности требуется новый workspace либо очистка векторных данных и полная переиндексация.
