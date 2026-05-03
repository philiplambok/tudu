# Development Dockerfile for Tudu
FROM golang:1.26-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        git \
        postgresql-client \
        tzdata \
    && rm -rf /var/lib/apt/lists/*

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOPATH=/go \
    PATH="/go/bin:${PATH}"

RUN go install github.com/air-verse/air@latest \
    && go install github.com/pressly/goose/v3/cmd/goose@latest

WORKDIR /app

EXPOSE 8080

CMD ["sleep", "infinity"]
