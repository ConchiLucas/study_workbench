FROM golang:1.26-alpine AS builder

WORKDIR /build
ENV GOPROXY=https://goproxy.cn,direct     GOSUMDB=sum.golang.google.cn
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:latest

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=builder /build/server .
COPY config.docker.yaml config.yaml

EXPOSE 6015

CMD ["./server"]
