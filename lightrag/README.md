# Local LightRAG

Локальная конфигурация LightRAG для индексации университетских PDF без
платных API. Целевая машина: Windows, 16+ ГБ RAM, NVIDIA RTX 4060.

## Быстрый запуск

Откройте PowerShell в этой папке:

```powershell
powershell -ExecutionPolicy Bypass -File .\setup.ps1
powershell -ExecutionPolicy Bypass -File .\start.ps1
```

Не закрывайте окно сервера. В другом окне выполните:

```powershell
powershell -ExecutionPolicy Bypass -File .\index.ps1
```

Web UI: <http://127.0.0.1:9621>

## Модели

- `qwen3:4b-instruct` — извлечение сущностей, связей и ответы.
- `bge-m3` — мультиязычные embeddings размерности 1024.

Обе модели работают локально через Ollama. API-ключи не требуются.

## Важные детали

- PDF лежат в `documents/`.
- `prepare-input.ps1` создаёт копии с ASCII-именами в `inputs/`. Это обходит
  ошибку разрешения Unicode-путей в LightRAG 1.5.4 на Windows.
- `INPUT_DIR` задаётся отдельно, поскольку legacy-парсер читает именно эту
  переменную окружения.
- Индекс привязан к embedding-модели и её размерности. После смены модели
  удалите локальную папку `rag_storage` и проиндексируйте документы заново.
- `.env`, модели Ollama, виртуальное окружение и готовый индекс не коммитятся.
