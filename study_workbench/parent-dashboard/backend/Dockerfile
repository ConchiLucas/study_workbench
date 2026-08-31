FROM node:22-alpine AS parent-web

WORKDIR /build/parent-dashboard/frontend

COPY parent-dashboard/frontend/package.json parent-dashboard/frontend/package-lock.json* ./
RUN npm ci

COPY parent-dashboard/frontend/ ./

ARG VITE_KID_APP_URL=http://localhost:19082
ENV VITE_KID_APP_URL=$VITE_KID_APP_URL

RUN npm run build

FROM golang:1.26-alpine AS backend-build

WORKDIR /src

COPY parent-dashboard/backend/go.mod parent-dashboard/backend/go.sum ./
RUN go mod download

COPY parent-dashboard/backend/ ./
COPY --from=parent-web /build/parent-dashboard/backend/internal/http/dist ./internal/http/dist

RUN CGO_ENABLED=0 go build -o /out/workbench ./cmd/server && \
    CGO_ENABLED=0 go build -o /out/seed ./cmd/seed

FROM alpine:3.21 AS backend

RUN apk add --no-cache wget

WORKDIR /app

COPY --from=backend-build /out/workbench ./workbench

ENV APP_ADDR=:19081

EXPOSE 19081

HEALTHCHECK --interval=5s --timeout=3s --retries=10 \
  CMD wget -qO- http://127.0.0.1:19081/healthz >/dev/null || exit 1

CMD ["./workbench"]

FROM alpine:3.21 AS seed

WORKDIR /app

COPY --from=backend-build /out/seed ./seed
COPY parent-dashboard/backend/scripts/docker-seed.sh ./docker-seed.sh
RUN chmod +x ./docker-seed.sh

CMD ["./docker-seed.sh"]
