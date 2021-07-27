FROM golang:1.16.6-alpine3.14 as builder

COPY / /app
WORKDIR /app
RUN GO111MODULE=on GOPROXY="https://goproxy.cn" CGO_ENABLED=0 GOOS=linux go build -a -o /go/bin/redeploy .

FROM alpine:3.14

EXPOSE 80
RUN echo "http://mirrors.aliyun.com/alpine/v3.14/main" > /etc/apk/repositories
RUN echo "http://mirrors.aliyun.com/alpine/v3.14/community" >> /etc/apk/repositories
RUN apk add -U tzdata \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && apk add ca-certificates \
    && rm -rf /var/cache/apk/*

COPY --from=builder /go/bin/redeploy .

ENV API_SERVER="" \
    WEBHOOK_TOKEN="" \
    KEBECONFIG="/root/kubeconfig"

CMD ["/redeploy"]
