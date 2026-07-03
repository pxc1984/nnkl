# PDF OCR Service

Микросервис для OCR материаловедческих документов (справочники, отчёты, ТУ, ГОСТы) с экспортом в LaTeX/Markdown.

## Стек

- **Python 3.11** (Docker), FastAPI 0.110+, Celery 5.4+, Redis
- **Docling** (IBM) — парсинг PDF, OCR, таблицы, формулы
- **PyMuPDF** — предобработка сканов
- **pikepdf** — валидация PDF
- Docker (GPU опционально)

## Быстрый старт

### 1. Подготовка

```bash
cd ocr
cp .env.example .env
mkdir -p data/pdfs data/results data/models
```

Windows (PowerShell):

```powershell
cd ocr
Copy-Item .env.example .env
New-Item -ItemType Directory -Force -Path data/pdfs, data/results, data/models
```

Положите PDF-файлы в `data/pdfs/` (эта директория монтируется в контейнеры как `/data/pdfs`).

### 2. Запуск

```bash
docker compose up --build
```

Сервисы:
- **API**: http://localhost:8000
- **Swagger**: http://localhost:8000/docs
- **Redis**: localhost:6379

### 3. GPU / CPU

По умолчанию `docker-compose.yml` запускает воркер в **CPU-режиме** (`DOCLING_USE_GPU=false`, `DOCLING_DO_FORMULA_ENRICHMENT=false`).

Для GPU установите в `.env`:

```env
DOCLING_USE_GPU=true
```

и добавьте в сервис `worker` секцию `deploy.resources` с доступом к NVIDIA GPU (требуется [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html)).

## API

### Health check

```bash
curl http://localhost:8000/api/v1/health
```

Ответ:

```json
{
  "status": "ok",
  "api": "ok",
  "redis": "ok",
  "celery": "ok"
}
```

### Конвертация PDF

```bash
curl -X POST http://localhost:8000/api/v1/convert \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: my-trace-001" \
  -d '{
    "file_path": "/data/pdfs/report.pdf",
    "output_format": "latex",
    "language": "auto"
  }'
```

`output_format`: `latex` (по умолчанию) или `markdown`.  
`language`: `auto` (по умолчанию), `ru`, `en`.

Ответ:

```json
{"task_id": "abc-123", "status": "pending"}
```

### Статус задачи

```bash
curl http://localhost:8000/api/v1/status/abc-123
```

Ответ:

```json
{
  "task_id": "abc-123",
  "status": "processing",
  "progress": 45,
  "stage": "docling",
  "result_url": null,
  "error": null
}
```

### Очередь задач

```bash
# Все задачи в очереди
curl http://localhost:8000/api/v1/queue

# Только ожидающие (pending + scheduled)
curl "http://localhost:8000/api/v1/queue?status=pending"

# Только выполняющиеся
curl "http://localhost:8000/api/v1/queue?status=processing"
```

Ответ:

```json
{
  "total": 3,
  "pending_count": 2,
  "processing_count": 1,
  "scheduled_count": 0,
  "pending": [
    {
      "task_id": "abc-123",
      "status": "pending",
      "name": "convert_pdf",
      "file_path": "/data/pdfs/report.pdf",
      "output_format": "latex",
      "language": "auto",
      "worker": null,
      "progress": 0,
      "eta": null
    }
  ],
  "processing": [],
  "scheduled": []
}
```

### Скачивание результата

```bash
curl -O http://localhost:8000/api/v1/download/abc-123
```

## Безопасность путей

Клиент передаёт **локальный путь** к PDF, а не файл. Защита:

| Проверка | Описание |
|----------|----------|
| `ALLOWED_BASE_PATH` | Путь должен быть внутри разрешённой директории |
| `resolve()` | Нормализация пути |
| `..` | Запрет path traversal |
| Symlink | Запрет ссылок за пределы base path |
| Расширение | Только `.pdf` |
| Размер | Лимит `MAX_FILE_SIZE_MB` |

Подозрительные попытки логируются в security audit log (`security.path_access_denied`).

## Volume mounts

```yaml
volumes:
  - ./data/pdfs:/data/pdfs:ro      # PDF-файлы (read-only)
  - ./data/results:/app/results     # Результаты .tex/.md
  - ./data/models:/app/models       # Кэш моделей Docling (только worker)
```

API и Worker **должны** иметь доступ к одним и тем же `pdfs` и `results`.

## Локальная разработка

```bash
cd ocr
python -m venv .venv
source .venv/bin/activate          # Windows: .venv\Scripts\activate
pip install -r requirements.txt

# Redis
docker run -d -p 6379:6379 redis:7-alpine

# API (из каталога ocr/)
uvicorn app.main:app --reload

# Worker (отдельный терминал)
celery -A app.workers.celery_app:celery_app worker --loglevel=info --concurrency=1
```

Для локального запуска задайте пути в `.env` (например, `ALLOWED_BASE_PATH=./data/pdfs`, `RESULTS_DIR=./data/results`, `REDIS_URL=redis://localhost:6379/0`).

### Тесты

```bash
cd ocr
pip install -r requirements.txt
pytest tests/ -v
```

Unit-тесты мокируют Docling и не требуют GPU. Полный OCR-пайплайн проверяйте через Docker (`docker compose up`).

## Архитектура

```
Клиент → POST /convert (file_path) → FastAPI валидация → Celery queue
                                                              ↓
Клиент ← GET /download ← results/ ← LaTeX export ← Docling ← Worker
         GET /status
         GET /queue
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `ALLOWED_BASE_PATH` | `/data/pdfs` | Корневая директория PDF |
| `MAX_FILE_SIZE_MB` | `200` | Макс. размер файла |
| `RESULTS_DIR` | `/app/results` | Директория результатов |
| `REDIS_URL` | `redis://redis:6379/0` | Redis |
| `CELERY_BROKER_URL` | `redis://redis:6379/0` | Брокер Celery |
| `CELERY_RESULT_BACKEND` | `redis://redis:6379/1` | Backend результатов Celery |
| `DOCLING_USE_GPU` | `true` | GPU для моделей Docling |
| `DOCLING_DO_FORMULA_ENRICHMENT` | `true` | Обогащение формул (VLM; на CPU очень медленно) |
| `DOCLING_PRELOAD_MODELS` | `false` | Предзагрузка моделей при старте воркера |
| `TASK_SOFT_TIME_LIMIT` | `1800` | Soft timeout (сек) |
| `TASK_TIME_LIMIT` | `2100` | Hard timeout (сек) |

## Известные ограничения

1. **Первый запуск воркера** — загрузка моделей Docling занимает несколько минут и ~2–4 ГБ RAM/VRAM.
2. **GPU** — без NVIDIA Container Toolkit воркер нужно запускать в CPU-режиме (`DOCLING_USE_GPU=false`).
3. **LaTeX tabularx** — сложные таблицы с вложенными объединениями ячеек могут требовать ручной правки.
4. **do_formula_enrichment** — в Docling 2.x используется `do_formula_enrichment` (не `do_formula_recognition`); в воркере по умолчанию отключено из-за производительности на CPU.
5. **OCR** — выполняется встроенным pipeline Docling (`do_ocr=True`); движок выбирается автоматически.
6. **Предобработка сканов** — базовая эвристика; для критичных документов рекомендуется ручная настройка DPI.
