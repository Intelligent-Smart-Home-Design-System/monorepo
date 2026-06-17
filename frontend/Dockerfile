FROM node:20-alpine AS deps
WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
COPY frontend/apps/ ./apps/
COPY frontend/packages/ ./packages/
COPY services/layout/internal/configs/tracks.json /services/layout/internal/configs/tracks.json

RUN corepack enable && npm ci

FROM deps AS builder
ARG APP

RUN APP_NAME=$(node -e "console.log(require('./apps/$APP/package.json').name)") && \
    npm run build --workspace="$APP_NAME"

FROM node:20-alpine AS next
ARG APP
ENV APP=$APP

WORKDIR /app
COPY --from=builder /app/apps/$APP/.next/standalone ./
COPY --from=builder /app/apps/$APP/.next/static ./apps/$APP/.next/static
COPY --from=builder /app/apps/$APP/public ./apps/$APP/public

EXPOSE 3000
ENV PORT=3000
CMD sh -c "node apps/$APP/server.js"

FROM nginx:alpine AS vite
ARG APP

COPY --from=builder /app/apps/$APP/dist /usr/share/nginx/html

EXPOSE 80
