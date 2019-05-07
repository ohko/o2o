#!/bin/sh

go test .
if [ $? -ne 0 ];then echo "err!";exit 1;fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./server_linux -ldflags "-s -w" ./server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./client_linux -ldflags "-s -w" ./client

docker build --rm --no-cache -t registry.cn-shenzhen.aliyuncs.com/cdeyun/o2o .
docker push registry.cn-shenzhen.aliyuncs.com/cdeyun/o2o
docker images |grep "<none>"|awk '{print $3}'|xargs docker image rm

rm -rf server_linux client_linux