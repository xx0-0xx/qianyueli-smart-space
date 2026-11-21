FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/server ./server
COPY config/config.example.yaml ./config/config.yaml
COPY migrations/ ./migrations/
RUN mkdir -p audio uploads
EXPOSE 8080
CMD ["./server"]
