# ---------- Build stage ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o phylax \
    ./cmd/phylax


# ---------- Runtime stage ----------
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/phylax ./phylax

# Runtime files your app may need
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/database ./database
COPY --from=builder /app/docs ./docs

ENV PHYLAX_ENV=production
ENV PHYLAX_ADDR=:8080

EXPOSE 8080

CMD ["./phylax"]