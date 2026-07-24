# syntax=docker/dockerfile:1.7

FROM node:20-alpine AS web-deps
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci

FROM web-deps AS web-build
COPY web/ ./
RUN npm run build

FROM web-deps AS frontend-test
COPY web/ ./
RUN npm test && touch /tmp/frontend-tests-ok

FROM golang:1.25-alpine AS go-deps
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod ./
RUN go mod download

FROM go-deps AS backend-test
COPY . .
RUN go mod tidy && go test ./...

FROM backend-test AS test
COPY --from=frontend-test /tmp/frontend-tests-ok /tmp/frontend-tests-ok

FROM go-deps AS go-build
COPY . .
RUN go mod tidy \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/wishtrack ./cmd/app

FROM alpine:3.21 AS runtime
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 wishtrack \
    && adduser -S -D -H -u 10001 -G wishtrack wishtrack \
    && mkdir -p /app/data /app/media \
    && chown -R wishtrack:wishtrack /app
WORKDIR /app
COPY --from=go-build /out/wishtrack /app/wishtrack
COPY --from=web-build /src/web/dist /app/web/dist
COPY migrations /app/migrations
USER wishtrack
EXPOSE 8080
ENV HTTP_ADDR=:8080 \
    DATABASE_PATH=/app/data/wishtrack.db \
    MEDIA_DIR=/app/media \
    MIGRATIONS_DIR=/app/migrations \
    WEB_DIR=/app/web/dist
ENTRYPOINT ["/app/wishtrack"]
