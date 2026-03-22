# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/gadk ./cmd/gadk

# ---- Runtime stage ----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/gadk /usr/local/bin/gadk

ENTRYPOINT ["gadk"]
