FROM golang:alpine AS builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o api ./cmd/api/
RUN go build -o cleaner ./cmd/cleaner/
RUN go install github.com/pressly/goose/v3/cmd/goose@latest


FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/api ./api
COPY --from=builder /build/cleaner ./cleaner
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY internal/database/migrations ./migrations
COPY entrypoint.sh ./entrypoint.sh

RUN chmod +x entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
CMD ["./api"]
