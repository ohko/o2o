FROM golang:1.12-stretch AS builder
ENV GO111MODULE on
ENV CGO_ENABLED 0
ENV GOFLAGS -mod=vendor
COPY . /go/src
WORKDIR /go/src
RUN go build -v -o server_linux -ldflags "-s -w" ./server
RUN go build -v -o client_linux -ldflags "-s -w" ./client

# ===========================

FROM scratch
LABEL maintainer="ohko <ohko@qq.com>"
COPY --from=builder /go/src/server_linux /
COPY --from=builder /go/src/client_linux /
WORKDIR /
