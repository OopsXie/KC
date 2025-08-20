FROM golang:1.24-alpine AS builder

# 设置 apk 和 Go mod 加速
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
  && apk add --no-cache git

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY ./registry ./registry
RUN cd registry && CGO_ENABLED=0 GOOS=linux go build -o /app/registry main.go

FROM alpine:latest

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
  && apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/registry .
COPY registry/config/registry-docker-1.yaml ./config/
COPY registry/config/registry-docker-2.yaml ./config/

EXPOSE 28180
CMD ["./registry"]
