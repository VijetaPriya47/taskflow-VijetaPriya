FROM golang:1.22-alpine AS build

RUN apk add --no-cache ca-certificates git
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go build -o /out/taskflow ./cmd/taskflow

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.0


FROM alpine:3.20

RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app

COPY --from=build /out/taskflow /app/taskflow
COPY --from=build /go/bin/migrate /usr/local/bin/migrate
COPY internal/storage/postgres/migrations /app/migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh

RUN chmod +x /app/docker-entrypoint.sh && chown -R app:app /app
USER app

EXPOSE 4000
ENTRYPOINT ["/app/docker-entrypoint.sh"]

