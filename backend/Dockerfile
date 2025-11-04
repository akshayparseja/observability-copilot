FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download
RUN go mod tidy
RUN go build -o server ./cmd/server/main.go

FROM alpine:3.18

WORKDIR /app

# Install git in final image for cloning repos
RUN apk add --no-cache git

COPY --from=builder /app/server .

EXPOSE 8000

CMD ["./server"]
