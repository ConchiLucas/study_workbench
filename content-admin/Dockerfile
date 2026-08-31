# syntax=docker/dockerfile:1

FROM node:22-alpine AS frontend
WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend-build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /build/backend/internal/http/dist ./internal/http/dist
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache wget
WORKDIR /app
COPY --from=backend-build /out/server ./server
COPY backend/assets/fonts ./assets/fonts
ENV APP_ADDR=:19091
ENV GLYPH_FONT_PATH=/app/assets/fonts/NotoSerifSC-Regular.otf
ENV APP_MINIO_PUBLIC_BASE=http://localhost:19100
EXPOSE 19091
HEALTHCHECK --interval=5s --timeout=3s --retries=10 \
  CMD wget -qO- http://127.0.0.1:19091/healthz >/dev/null || exit 1
CMD ["./server"]
