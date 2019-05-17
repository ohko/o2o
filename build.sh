#!/bin/sh

go test .
if [ $? -ne 0 ];then echo "err!";exit 1;fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./server_linux -ldflags "-s -w" ./server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./client_linux -ldflags "-s -w" ./client

docker build --rm --no-cache -t registry.cn-shenzhen.aliyuncs.com/cdeyun/o2o .
docker push registry.cn-shenzhen.aliyuncs.com/cdeyun/o2o
docker images |grep "<none>"|awk '{print $3}'|xargs docker image rm

rm -rf server_linux client_linux

# docker pull registry.cn-shenzhen.aliyuncs.com/cdeyun/o2o && docker rm -fv o2o; docker run --name=o2o -d --restart=always -p 2399:2399 -p 127.0.0.1:5001:5001 -v /usr/share/zoneinfo:/usr/share/zoneinfo:ro -e TZ=Asia/Shanghai registry.cn-shenzhen.aliyuncs.com/cdeyun/o2o /server -s=:2399
# docker pull registry.cn-shenzhen.aliyuncs.com/cdeyun/o2o && docker rm -fv o2o; docker run --name=o2o -d --restart=always -v /usr/share/zoneinfo:/usr/share/zoneinfo:ro -e TZ=Asia/Shanghai registry.cn-shenzhen.aliyuncs.com/cdeyun/o2o /client -s=cdeyun.com:2399 -p 5001:192.168.1.241:5001