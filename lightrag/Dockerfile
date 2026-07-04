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

COPY patch_postgres_table_names.py .
RUN python patch_postgres_table_names.py && rm patch_postgres_table_names.py

RUN mkdir -p /app/data/rag_storage /app/data/inputs /app/data/prompts

EXPOSE 9621

CMD ["lightrag-server", \
     "--host", "0.0.0.0", \
     "--port", "9621", \
     "--working-dir", "/app/data/rag_storage", \
     "--input-dir", "/app/data/inputs", \
     "--workers", "1"]
