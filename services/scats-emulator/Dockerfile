# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod ./
COPY . .
RUN go mod tidy && go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o scats-emulator ./cmd

# Runtime stage
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/scats-emulator .

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8082

CMD ["./scats-emulator"]