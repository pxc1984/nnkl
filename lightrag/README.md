# LightRAG + OCR

Локальный RAG для вопросов по PDF-документам. PDF сначала конвертируются в LaTeX через OCR-сервис (Docling), затем индексируются в LightRAG.

## Архитектура

```
PDF (ocr/data/pdfs)  →  OCR API :8000  →  .tex (lightrag/inputs)
                                              ↓
                                    LightRAG :9621  →  граф + векторы
                                              ↓
                              Ollama (qwen3 + bge-m3)  →  ответы на вопросы
```

## Быстрый старт (Docker)

### 1. Запустить OCR

```powershell
cd ocr
docker compose up -d --build
```

Положите PDF в `ocr/data/pdfs/`.

### 2. Запустить LightRAG + Ollama

```powershell
cd lightrag
Copy-Item .env.docker.example .env   # если .env ещё нет
docker compose up -d --build
```

Первый запуск скачает модели Ollama (~4 ГБ). Web UI: <http://127.0.0.1:9621>

### 3. Конвертировать PDF и проиндексировать

**Вариант A** — полный пайплайн (PDF → LaTeX → LightRAG):

```powershell
cd lightrag
# Положите PDF в documents/ или укажите -SourceDir
powershell -ExecutionPolicy Bypass -File .\ingest-via-ocr.ps1
```

**Вариант B** — импорт уже готовых .tex из OCR:

```powershell
cd lightrag
powershell -ExecutionPolicy Bypass -File .\import-ocr-results.ps1 -TriggerScan
```

Скрипт дедуплицирует одинаковые результаты и кладёт уникальные `.tex` в `inputs/`.

### 4. Задать вопрос

```powershell
powershell -ExecutionPolicy Bypass -File .\query.ps1 -Query "Что такое фракционирование изотопов углерода?"
```

Или через Web UI: <http://127.0.0.1:9621>

## Локальный запуск (без Docker для LightRAG)

```powershell
cd lightrag
powershell -ExecutionPolicy Bypass -File .\setup.ps1   # Ollama + venv
powershell -ExecutionPolicy Bypass -File .\start.ps1     # сервер
# в другом окне:
powershell -ExecutionPolicy Bypass -File .\import-ocr-results.ps1 -TriggerScan
```

## Модели

- `qwen3:4b-instruct` — извлечение сущностей, связей и ответы
- `bge-m3` — мультиязычные embeddings (1024 dim)

## Скрипты

| Скрипт | Назначение |
|--------|------------|
| `ingest-via-ocr.ps1` | PDF → OCR API → `inputs/*.tex` → `/documents/scan` |
| `import-ocr-results.ps1` | Готовые `.tex` из `ocr/data/results` → `inputs/` |
| `query.ps1` | Вопрос к проиндексированной базе |
| `prepare-input.ps1` | Прямое копирование PDF (без OCR) |
| `index.ps1` | Только `/documents/scan` |

## Важные детали

- LightRAG 1.5.4 поддерживает `.tex` напрямую — отдельная конвертация в Markdown не нужна.
- `import-ocr-results.ps1` убирает дубликаты по SHA256 (один PDF → один `.tex`).
- `ocr_mapping.json` хранится в корне `lightrag/`, не в `inputs/`.
- Индекс привязан к embedding-модели. После смены модели удалите `rag_storage/` и переиндексируйте.
- При таймаутах LLM увеличьте `LLM_TIMEOUT` в `.env` (по умолчанию 600 с).
- Первый OCR-запуск воркера загружает модели Docling (~2–5 мин).

## Переменные окружения (.env)

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `LLM_BINDING_HOST` | `http://ollama:11434` | Ollama в Docker |
| `LLM_MODEL` | `qwen3:4b-instruct` | LLM |
| `EMBEDDING_MODEL` | `bge-m3` | Embeddings |
| `LLM_TIMEOUT` | `600` | Таймаут LLM (сек) |
| `OLLAMA_LLM_NUM_CTX` | `4096` | Контекст LLM |
| `SUMMARY_LANGUAGE` | `Russian` | Язык ответов |

## Сброс индекса

```powershell
cd lightrag
Remove-Item -Recurse -Force rag_storage\*
docker compose restart lightrag
powershell -ExecutionPolicy Bypass -File .\import-ocr-results.ps1 -TriggerScan
```
