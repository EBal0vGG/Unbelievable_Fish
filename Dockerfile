# syntax=docker/dockerfile:1.7

FROM golang:1.22-alpine AS build
ARG SERVICE
WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /bin/service ./cmd/${SERVICE}

FROM alpine:3.19
COPY --from=build /bin/service /service
COPY migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/service"]
