FROM python:3.12-slim-bookworm

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --upgrade pip && pip install -r requirements.txt

RUN mkdir -p /app/rag_storage /app/inputs

EXPOSE 9621

CMD ["lightrag-server", \
     "--host", "0.0.0.0", \
     "--port", "9621", \
     "--working-dir", "/app/rag_storage", \
     "--input-dir", "/app/inputs", \
     "--workers", "1"]
