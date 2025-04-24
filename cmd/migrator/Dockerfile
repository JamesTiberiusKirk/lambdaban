FROM golang:alpine AS builder
RUN mkdir /build 
WORKDIR /build
ADD ./internal/db/sql ./sql
ADD ./cmd/migrator/script.sh .
RUN go install github.com/JamesTiberiusKirk/migrator/cmd/migrator@latest

FROM alpine
COPY --from=builder /build/sql /app/sql
COPY --from=builder /build/script.sh /app/
COPY --from=builder /go/bin/migrator /app/
WORKDIR /app

ENTRYPOINT ["/app/script.sh"]
