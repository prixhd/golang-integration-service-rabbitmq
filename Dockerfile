FROM golang:1.26-alpine AS builder

WORKDIR /v1

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /v1/server ./cmd/server

FROM alpine:latest

WORKDIR /v1

COPY --from=builder /v1/server ./server

EXPOSE 8000

CMD ["./server"]