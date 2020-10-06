FROM golang:1.15-buster AS builder
ENV CGO_ENABLED 0
COPY . /go/src
WORKDIR /go/src
RUN go build -v -o server_linux -ldflags "-s -w" ./server
RUN go build -v -o client_linux -ldflags "-s -w" ./client
RUN go build -v -o forward_linux -ldflags "-s -w" ./forward

# ===========================

FROM scratch
LABEL maintainer="ohko <ohko@qq.com>"
COPY --from=builder /go/src/server_linux /
COPY --from=builder /go/src/client_linux /
COPY --from=builder /go/src/forward_linux /
WORKDIR /
ENV TZ Asia/Shanghai
ENV LOG_LEVEL 1