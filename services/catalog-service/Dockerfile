# syntax=docker/dockerfile:1

FROM golang:1.26.6-alpine AS build

ENV GOTOOLCHAIN=local

WORKDIR /src

COPY go.work go.work.sum ./
COPY gen/go/go.mod gen/go/go.mod
COPY services/catalog-service/go.mod services/catalog-service/go.sum services/catalog-service/
COPY services/pricing-service/go.mod services/pricing-service/go.sum services/pricing-service/

WORKDIR /src/services/catalog-service
RUN go mod download

WORKDIR /src
COPY gen/go gen/go
COPY services/catalog-service services/catalog-service

WORKDIR /src/services/catalog-service
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/catalog-service ./cmd/catalog-service
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/http-healthcheck ./cmd/http-healthcheck

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=build /out/catalog-service /app/catalog-service
COPY --from=build /out/http-healthcheck /app/http-healthcheck

USER app

EXPOSE 8080

ENTRYPOINT ["/app/catalog-service"]
