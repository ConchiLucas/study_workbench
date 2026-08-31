FROM golang:1.26-alpine AS builder

WORKDIR /build
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn
RUN apk add --no-cache git

COPY word_select_dashboard/server/go.mod word_select_dashboard/server/go.sum ./
RUN go mod download

COPY word_select_dashboard/server/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:latest

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=builder /build/server .
COPY word_select_dashboard/server/config.docker.yaml config.yaml
RUN awk 'BEGIN { in_word_agent=0; in_system=0 } /^word-agent:/ { in_word_agent=1; in_system=0 } /^system:/ { in_system=1; in_word_agent=0 } /^[^[:space:]].*:/ && $0 !~ /^word-agent:/ && $0 !~ /^system:/ { in_word_agent=0; in_system=0 } in_word_agent && /^[[:space:]]+base-url:/ { print "    base-url: http://host.docker.internal:6017"; next } in_system && /^[[:space:]]+addr:/ { print "    addr: 6015"; next } { print }' config.yaml > config.yaml.tmp \
    && mv config.yaml.tmp config.yaml \
    && sed -i 's/username: change-me-user/username: conchi/' config.yaml \
    && sed -i 's/password: change-me-password/password: conchi123456/' config.yaml

EXPOSE 6015

CMD ["./server"]
