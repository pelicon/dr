FROM golang:alpine

# set mirror repository for the package management
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/' /etc/apk/repositories
RUN apk add make protobuf docker-cli git build-base bash
COPY ./tools/operator-sdk-v0.18.2-x86_64-linux-gnu /usr/bin/operator-sdk
COPY ./tools/protoc-gen-go-v1.2.0 /go/bin/protoc-gen-go
RUN chmod +x /usr/bin/operator-sdk /go/bin/protoc-gen-go

ENV GO111MODULE=on GOPROXY=https://goproxy.io
RUN go get github.com/golang/mock/mockgen@v1.3.1
RUN git clone -b release-1.18 --single-branch https://gitee.com/newlooc/code-generator.git && mv code-generator /code-generator
RUN echo "replace github.com/DaoCloud/uds => /go/src/github.com/DaoCloud/uds" >> /code-generator/go.mod
COPY boilerplate.go.txt /code-generator/boilerplate.go.txt

