# syntax=docker/dockerfile:1

FROM golang:1.26.9-alpine AS build

ENV GOTOOLCHAIN=local

WORKDIR /src

COPY go.work go.work.sum ./
COPY gen/go/go.mod gen/go/go.mod
COPY services/catalog-service/go.mod services/catalog-service/go.sum services/catalog-service/
COPY services/pricing-service/go.mod services/pricing-service/go.sum services/pricing-service/

WORKDIR /src/services/pricing-service
RUN go mod download

WORKDIR /src
COPY gen/go gen/go
COPY services/pricing-service services/pricing-service

WORKDIR /src/services/pricing-service
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pricing-service ./cmd/pricing-service
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/grpc-healthcheck ./cmd/grpc-healthcheck

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=build /out/pricing-service /app/pricing-service
COPY --from=build /out/grpc-healthcheck /app/grpc-healthcheck

USER app

EXPOSE 9090

ENTRYPOINT ["/app/pricing-service"]
