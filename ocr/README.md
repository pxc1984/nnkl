# OCR Service

OCR microservice for document processing using Yandex Vision API.

## Features

<<<<<<< HEAD
- PDF text extraction and OCR processing
- Yandex Vision API integration for high-quality OCR
- Native text extraction for documents that already contain text
- PDF quality analysis to determine OCR requirements
- Markdown output with proper formatting
- Table post-processing for improved readability
=======
1. Gateway stores document bytes in shared Postgres table `blobs`.
2. Gateway creates or updates an `uploads` row and sends `POST /api/v1/parse` with `upload_id`, `input_blob_id`, `language`.
3. OCR service reads the input blob from Postgres, parses `pdf` through MinerU and extracts `docx` / `pptx` text natively, stores parsed markdown in `blobs`, links it as `uploads.output_blob`, and returns `201 Created`.
4. Gateway can read status and result through OCR API or directly from the shared database.
>>>>>>> 9b0f2229349c8bf0fb7ca1d9be7d14f311c891f1

## Configuration

Environment variables (defined in the main `.env` file):

- `YANDEX_VISION_API_KEY`: Your Yandex Vision API key
- `YANDEX_FOLDER_ID`: Your Yandex Cloud folder ID
- `OCR_API_PREFIX`: API path prefix (default: `/api/v1`)
- `OCR_LOG_LEVEL`: Logging level (default: `INFO`)
- `OCR_MIN_CHARACTERS`: Minimum characters threshold for text usability (default: `40`)
- `OCR_MINIMUM_USABLE_PAGE_RATIO`: Minimum ratio of usable pages for native extraction (default: `0.95`)
- `OCR_DOCUMENT_TIMEOUT_SECONDS`: Timeout for document processing (default: `1800.0`)
- `OCR_PREPROCESS_SCANS`: Preprocess scanned documents (default: `true`)
- `OCR_SCAN_DPI`: DPI for scan preprocessing (default: `220`)
- `OCR_MAX_PAGE_MEGAPIXELS`: Maximum page size in megapixels (default: `12`)

## Endpoints

- `POST /api/v1/parse` - Parse document and return OCR results
- `GET /api/v1/status/{document_id}` - Get parsing status
- `GET /api/v1/result/{document_id}` - Get parsing result
- `GET /api/v1/health` - Health check

## Architecture

The service uses Yandex Vision API for OCR processing, eliminating the need for local models or heavy computational resources. Documents are analyzed to determine if they contain native text or require OCR processing.

## Local Development

```bash
<<<<<<< HEAD
# Install dependencies
pip install -e .

# Run the service
uvicorn app.main:app --reload
=======
curl -X POST http://localhost:8000/api/v1/parse \
  -H "Content-Type: application/json" \
  -d '{
    "upload_id": "00000000-0000-0000-0000-000000000123",
    "input_blob_id": "blob-123",
    "language": "auto"
  }'
>>>>>>> 9b0f2229349c8bf0fb7ca1d9be7d14f311c891f1
```

## Docker

<<<<<<< HEAD
Build and run with Docker:

```bash
docker build -t ocr-service .
docker run -p 8000:8000 ocr-service
```
=======
```json
{
  "upload_id": "00000000-0000-0000-0000-000000000123",
  "output_blob_id": "result-uuid",
  "status": "completed"
}
```

### Status

```bash
curl http://localhost:8000/api/v1/status/00000000-0000-0000-0000-000000000123
```

### Result

```bash
curl http://localhost:8000/api/v1/result/00000000-0000-0000-0000-000000000123
```

### Health

```bash
curl http://localhost:8000/api/v1/health
```

## Shared database tables

- `blobs`: shared binary content store for source files and parsed markdown
- `uploads`: upload status plus foreign keys to `input_blob` and `output_blob`

Tables are created automatically on startup.

## Local run

```bash
cp .env.example .env
docker compose up --build
```

Services:

- API: `http://localhost:8000`
- Postgres: `localhost:5432`

OCR is started from the root `docker-compose.yml` and uses the shared root `.env`.

## MinerU

PDF parsing uses [MinerU](https://github.com/OpenDataLab/MinerU) (`pipeline` backend by default).

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `OCR_MINERU_USE_GPU` | `false` | Use GPU backend when available |
| `OCR_MINERU_BACKEND` | `pipeline` | MinerU backend (`pipeline`, `hybrid-auto-engine`, ...) |
| `OCR_MINERU_MODELS_DIR` | — | Local models cache directory |
| `OCR_MINERU_DOCUMENT_TIMEOUT_SECONDS` | `1800` | MinerU subprocess timeout |
| `OCR_MINERU_PREPROCESS_SCANS` | `true` | Rasterize low-quality/scanned PDFs before OCR |
| `OCR_MINERU_SCAN_DPI` | `220` | Scan rasterization DPI |
| `OCR_MINERU_MAX_PAGE_MEGAPIXELS` | `12` | Per-page memory safety limit |
| `OCR_NATIVE_MIN_CHARACTERS` | `40` | Minimum useful native text per page |
| `OCR_NATIVE_MINIMUM_USABLE_PAGE_RATIO` | `0.95` | Native pages required to bypass OCR |
| `MINERU_MODEL_SOURCE` | `huggingface` | Set to `modelscope` if HuggingFace is blocked |

First container start downloads MinerU models (~2–4 GB).

## Test

```bash
cd ocr
uv sync --group dev
pytest tests -v
```
>>>>>>> 9b0f2229349c8bf0fb7ca1d9be7d14f311c891f1
