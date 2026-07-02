FROM node:22-alpine AS builder

WORKDIR /app

RUN corepack enable

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .
RUN pnpm run build

FROM node:22-alpine

WORKDIR /app

RUN corepack enable

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --prod=false

COPY --from=builder /app/dist ./dist

EXPOSE 5173

CMD ["pnpm", "exec", "vite", "preview", "--host", "0.0.0.0", "--port", "5173"]
