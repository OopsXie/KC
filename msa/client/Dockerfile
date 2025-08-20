FROM golang:1.24-alpine AS builder

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
  && apk add --no-cache git

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY ./client ./client
RUN cd client && CGO_ENABLED=0 GOOS=linux go build -o /app/client main.go

FROM alpine:latest

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
  && apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/client .

EXPOSE 28300
CMD ["./client", "--config=config.yaml"]
