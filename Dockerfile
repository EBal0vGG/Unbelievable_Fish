FROM golang:1.22-alpine AS build
ARG SERVICE
WORKDIR /app
COPY . .
RUN go build -o /bin/service ./cmd/${SERVICE}

FROM alpine:3.19
COPY --from=build /bin/service /service
COPY migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/service"]
