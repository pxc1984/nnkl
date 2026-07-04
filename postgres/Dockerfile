FROM postgres:16-bookworm AS build

ARG AGE_VERSION=master
ARG PGVECTOR_VERSION=v0.8.1

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    git \
    flex \
    bison \
    libreadline-dev \
    zlib1g-dev \
    postgresql-server-dev-16 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /tmp/build

RUN git clone --depth 1 --branch "${AGE_VERSION}" https://github.com/apache/age.git age \
    && make -C age PG_CONFIG=/usr/lib/postgresql/16/bin/pg_config install

RUN git clone --depth 1 --branch "${PGVECTOR_VERSION}" https://github.com/pgvector/pgvector.git pgvector \
    && make -C pgvector PG_CONFIG=/usr/lib/postgresql/16/bin/pg_config install

FROM postgres:16-bookworm

COPY --from=build /usr/lib/postgresql/16/lib/age.so /usr/lib/postgresql/16/lib/
COPY --from=build /usr/lib/postgresql/16/lib/vector.so /usr/lib/postgresql/16/lib/
COPY --from=build /usr/share/postgresql/16/extension/age* /usr/share/postgresql/16/extension/
COPY --from=build /usr/share/postgresql/16/extension/vector* /usr/share/postgresql/16/extension/
COPY initdb/ /docker-entrypoint-initdb.d/
