# Build stage
FROM golang:1.26-alpine AS builder

LABEL maintainer="Walter Chavarria <wchavarria03@gmail.com>"

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o life-base-api .

# Final stage
FROM alpine:3.21

LABEL maintainer="Walter Chavarria <wchavarria03@gmail.com>"

RUN apk add --no-cache ca-certificates tzdata && \
    update-ca-certificates

RUN addgroup -g 10001 appgroup && \
    adduser -D -u 10001 -G appgroup appuser

WORKDIR /app

COPY --from=builder /app/life-base-api .

RUN mkdir -p /app/data/input /app/data/output && \
    chown -R appuser:appgroup /app && \
    chmod 755 /app/life-base-api

USER appuser

ENV PATH="/app:${PATH}" \
    TZ="UTC"

ENTRYPOINT ["life-base-api"]
CMD ["serve"]
