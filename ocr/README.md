# OCR Service

OCR microservice for document processing using Yandex Vision API.

## Features

- PDF text extraction and OCR processing
- Yandex Vision API integration for high-quality OCR
- Native text extraction for documents that already contain text
- PDF quality analysis to determine OCR requirements
- Markdown output with proper formatting
- Table post-processing for improved readability

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
# Install dependencies
pip install -e .

# Run the service
uvicorn app.main:app --reload
```

## Docker

Build and run with Docker:

```bash
docker build -t ocr-service .
docker run -p 8000:8000 ocr-service
```