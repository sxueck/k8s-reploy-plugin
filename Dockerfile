FROM golang:1.20.0-alpine3.17 as builder

# Go 1.19 支持泛型

RUN apk add git

COPY / /app
WORKDIR /app
RUN GO111MODULE=on GOPROXY="https://goproxy.cn" CGO_ENABLED=0 GOOS=linux go build -a -o /go/bin/redeploy .

FROM alpine:3.17

EXPOSE 80
RUN echo "https://mirrors.aliyun.com/alpine/v3.17/main" > /etc/apk/repositories
RUN echo "https://mirrors.aliyun.com/alpine/v3.17/community" >> /etc/apk/repositories
RUN apk add -U tzdata \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && apk add ca-certificates \
    && rm -rf /var/cache/apk/*

COPY --from=builder /go/bin/redeploy .

ENV API_SERVER="" \
    WEBHOOK_TOKEN="" \
    KUBECONFIG="/root/kubeconfig"

CMD ["/redeploy"]
