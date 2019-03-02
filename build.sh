#!/bin/sh

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./docker/server ./server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./docker/client ./client

cd ./docker
tar czf tmp.tar.gz server client
rm -rf server client

docker build --rm --no-cache -t ohko/o2o .
docker push ohko/o2o
docker images |grep "<none>"|awk '{print $3}'|xargs docker image rm

rm -rf tmp.tar.gz