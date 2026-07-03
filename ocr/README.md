# PDF OCR Service

FastAPI parsing microservice for synchronous processing of `pdf`, `docx`, and `pptx` files already uploaded by a gateway into shared Postgres storage.

## Flow

1. Gateway stores document bytes in shared Postgres table `input_blobs`.
2. Gateway sends `POST /api/v1/parse` with `document_id`, `input_blob_id`, `output_format`, `language`.
3. OCR service reads the blob from Postgres, parses `pdf` through Docling and extracts `docx` / `pptx` text natively, stores output in Postgres tables `parse_jobs` and `parse_results`, and returns `201 Created`.
4. Gateway can read status and result through OCR API or directly from the shared database.

## API

### Parse document

```bash
curl -X POST http://localhost:8000/api/v1/parse \
  -H "Content-Type: application/json" \
  -d '{
    "document_id": "doc-123",
    "input_blob_id": "blob-123",
    "output_format": "latex",
    "language": "auto"
  }'
```

Successful response:

```json
{
  "document_id": "doc-123",
  "job_id": "job-uuid",
  "result_id": "result-uuid",
  "status": "completed"
}
```

### Status

```bash
curl http://localhost:8000/api/v1/status/doc-123
```

### Result

```bash
curl http://localhost:8000/api/v1/result/doc-123
```

### Health

```bash
curl http://localhost:8000/api/v1/health
```

## Shared database tables

- `input_blobs`: source PDF bytes written by gateway
- `parse_jobs`: OCR execution status keyed by `document_id`
- `parse_results`: parsed text and optional zipped assets

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

## Test

```bash
cd ocr
uv sync --group dev
pytest tests -v
```
