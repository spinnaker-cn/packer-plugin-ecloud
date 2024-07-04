FROM cs-hub.imgo.tv/devops/golang:1.20.13-alpine

MAINTAINER weizhi@mgtv.com

#ENV CGO_ENABLED=0
ENV GOOS=linux
#ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,https://goproxy.cn,https://ecloud.10086.cn/api/query/developer/nexus/repository/go-sdk/,direct
#ENV GO111MODULE=off
ENV GOPATH="/go/release:/go/release/src/gopathlib/"
 
 
ARG TZ="Asia/Shanghai"
ENV TZ ${TZ}

WORKDIR /build

RUN set -eux \
    && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories \
    && apk add --no-cache git tzdata \
    && cp /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo "${TZ}" > /etc/timezone \
    && apk del tzdata \
    && rm -rf /tmp/* /var/tmp/*


CMD ["bash"]

