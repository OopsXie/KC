FROM golang:1.24.1-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy && go build -o consumer .

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/consumer .
EXPOSE 8080
CMD ["./consumer"]