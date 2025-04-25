# ---- Build Stage ----
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go get ./...

COPY . .

RUN go build -o lambdaban ./cmd/web/main.go


FROM alpine:latest
RUN apk add --no-cache curl
WORKDIR /app
COPY --from=builder /app/lambdaban .
COPY --from=builder /app/assets ./assets
CMD ["./lambdaban"]
