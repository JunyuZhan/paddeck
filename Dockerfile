# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS build
WORKDIR /src

ARG VERSION=dev
ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" -o /paddeck ./cmd/paddeck

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
COPY --from=build /paddeck /paddeck
EXPOSE 9080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:9080/api/health || exit 1
ENTRYPOINT ["/paddeck", "-config", "/config/config.yaml", "-db", "/data/paddeck.db"]
