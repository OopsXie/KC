FROM golang:1.24.1-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy && go build -o provider .

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/provider .
EXPOSE 8080
CMD ["./provider"]