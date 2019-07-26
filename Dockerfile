FROM golang:1.12-stretch AS builder
ENV GO111MODULE on
ENV CGO_ENABLED 0
ENV GOFLAGS -mod=vendor
COPY . /go/src
WORKDIR /go/src
RUN go build -v -o server -ldflags "-s -w" ./server
RUN go build -v -o client -ldflags "-s -w" ./client

# ===========================

FROM scratch
LABEL maintainer="ohko <ohko@qq.com>"
COPY --from=builder /go/src/server /
COPY --from=builder /go/src/client /
WORKDIR /
