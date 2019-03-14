#!/bin/sh

go test .
if [ $? -ne 0 ];then echo "err!";exit 1;fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./docker/server -ldflags "-s -w" ./server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./docker/client -ldflags "-s -w" ./client

cd ./docker
tar czf tmp.tar.gz server client
rm -rf server client

docker build --rm --no-cache -t ohko/o2o .
docker push ohko/o2o
docker images |grep "<none>"|awk '{print $3}'|xargs docker image rm

rm -rf tmp.tar.gz