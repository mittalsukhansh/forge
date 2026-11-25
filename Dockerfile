FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/forge ./cmd/forge

FROM alpine:3.23

RUN apk add --no-cache \
    ca-certificates \
    docker-cli \
    tzdata

RUN addgroup -g 10001 forge \
    && adduser -D -u 10001 -G forge forge \
    && mkdir -p /data \
    && chown forge:forge /data

WORKDIR /app

COPY --from=builder /out/forge /usr/local/bin/forge

ENV FORGE_ADDR=0.0.0.0:8080
ENV FORGE_DB=/data/forge.db

USER forge

VOLUME ["/data"]

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["forge"]

CMD ["serve"]
