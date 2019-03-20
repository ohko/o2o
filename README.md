[![Build Status](https://drone.cdeyun.com/api/badges/cdeyun.com/o2o/status.svg)](https://drone.cdeyun.com/cdeyun.com/o2o)

# TCP 隧道

## Build
```
go build -mod=vendor -o server ./server
go build -mod=vendor -o client ./client
```

## Server
```
# 开启2399等待Client连接、传送指令、隧道服务
./server -s :2399 -key=mykey
```

## Client
```
# 连接服务器2399端口，传送指令
# 请求服务器开启2345端口，用于代理192.168.1.240的5000端口
./client -s x.x.x.x:2399 -p 2345:192.168.1.240:5000 -key=mykey
```

# Docker
```
# Server 开启2390-2399端口段
docker run --name=o2o -d --restart=always -p 2390-2399:2390-2399 ohko/o2o /server -s :2399 -key=mykey

# Client 请求2345端口代理192.168.1.240的5000端口
docker run --name=o2o -d --restart=always ohko/o2o /client -s x.x.x.x:2399 -p 2345:192.168.1.240:5000 -key=mykey

# 请求
curl http://x.x.x.x:2345
```
