# Контекст для продолжения работы

## Цель

Подготовить бесплатный локальный LightRAG для хакатон-проекта. База знаний —
43 университетских PDF по математическому анализу, линейной алгебре и
геометрии. После индексации система должна отвечать на вопросы по материалам
и использовать граф сущностей и связей.

## Целевая машина

- Windows.
- Intel Core i7.
- NVIDIA RTX 4060 Laptop.
- Желательно не менее 16 ГБ RAM.

Первоначальная настройка выполнялась на Ryzen 5 7535HS без дискретной GPU.
Локальная индексация на нём была слишком медленной, поэтому работа переносится
на ноутбук с RTX 4060.

## Выбранный стек

- LightRAG `1.5.4`.
- Ollama.
- LLM: `qwen3:4b-instruct`.
- Embeddings: `bge-m3`.
- Размерность embeddings: `1024`.
- Контекст: `8192`.
- Хранилища LightRAG по умолчанию: JSON KV, NanoVectorDB, NetworkX.

Модели запускаются полностью локально. Платные API и API-ключи не нужны.

## Почему не OpenAI

Изначально использовались:

- `gpt-4o-mini`;
- `text-embedding-3-large`;
- embedding dimension `3072`.

Индексация падала с OpenAI `429 insufficient_quota`, поскольку у аккаунта нет
API-кредитов. Конфигурация была заменена на Ollama.

## Обнаруженные особенности LightRAG 1.5.4 на Windows

### Unicode-имена PDF

Прямое сканирование PDF с русскими именами завершалось ошибкой:

```text
legacy source file not found
```

Решение: исходные файлы остаются в `documents/`, а `prepare-input.ps1`
создаёт копии `doc_001.pdf`, `doc_002.pdf` и так далее в `inputs/`.

### INPUT_DIR

Параметр CLI `--input-dir` используется сканером, но legacy-парсер версии
1.5.4 разрешает путь через переменную окружения `INPUT_DIR`. Поэтому
`start.ps1` задаёт и параметр CLI, и переменную окружения.

### Несовместимость индексов

Индекс, созданный с `text-embedding-3-large` размерности 3072, нельзя
использовать с `bge-m3` размерности 1024. На новой машине индекс нужно строить
заново. Локальный индекс намеренно не хранится в Git.

## Структура

```text
lightrag/
├── documents/          # 43 исходных PDF
├── .env.example        # конфигурация RTX 4060
├── requirements.txt
├── setup.ps1           # Python, зависимости, Ollama-модели
├── prepare-input.ps1   # ASCII-копии PDF
├── start.ps1           # запуск API и Web UI
└── index.ps1           # запуск сканирования
```

Каталоги `.venv`, `inputs`, `rag_storage`, логи и `.env` игнорируются Git.

## Продолжение на новом ноутбуке

После клонирования переключиться на ветку:

```powershell
git switch feature/lightrag-local
cd lightrag
```

Установить окружение и модели:

```powershell
powershell -ExecutionPolicy Bypass -File .\setup.ps1
```

Запустить сервер:

```powershell
powershell -ExecutionPolicy Bypass -File .\start.ps1
```

В другом PowerShell запустить индексирование:

```powershell
powershell -ExecutionPolicy Bypass -File .\index.ps1
```

Web UI: <http://127.0.0.1:9621>

## Что проверить первым

1. `nvidia-smi` видит RTX 4060.
2. `ollama ps` после начала обработки показывает использование GPU.
3. Первый документ получает статус `Processed`, а не `Failed`.
4. В логах отсутствуют `out of memory` и `legacy source file not found`.
5. Ответы на русском используют сведения из PDF.

Если не хватает VRAM, сначала уменьшить `OLLAMA_LLM_NUM_CTX` до `4096`.
Если качество извлечения графа недостаточно, попробовать `qwen3:8b` при
наличии достаточной VRAM/RAM; это увеличит время индексации.
