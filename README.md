[![github.com/ohko/o2o](https://goreportcard.com/badge/github.com/ohko/o2o)](https://goreportcard.com/report/github.com/ohko/o2o)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/5379c3c2746a42338a2bfaabe40a53d2)](https://www.codacy.com/app/ohko/o2o?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=ohko/o2o&amp;utm_campaign=Badge_Grade)
[![](https://images.microbadger.com/badges/image/ohko/o2o.svg)](https://microbadger.com/images/ohko/o2o "Get your own image badge on microbadger.com")

# TCP 隧道
将局域网任意IP的端口映射到公网服务器指定端口，隧道数据支持AES加密。
> 例如：将局域网`192.168.1.240:5000`映射到公网`x.x.x.x:2399`

## Download
下载预编译版本：[Release](https://github.com/ohko/o2o/releases)

## Build
```shell
go build -mod=vendor -o server ./server
go build -mod=vendor -o client ./client
```

## Server
```shell
# 开启2399等待Client连接、传送指令、隧道服务
./server -s :2399 -key=mykey
```

## Client
```shell
# 连接服务器2399端口，传送指令
# 请求服务器开启2345端口，用于代理192.168.1.240的5000端口
./client -s x.x.x.x:2399 -key=mykey -p 2345:192.168.1.240:5000
```

## Docker
```shell
# Server 开启2390-2399端口段
docker run --name=o2o -d --restart=always -p 2390-2399:2390-2399 ohko/o2o /server -s :2399 -key=mykey

# Client 请求2345端口代理192.168.1.240的5000端口
docker run --name=o2o -d --restart=always ohko/o2o /client -s x.x.x.x:2399 -p 2345:192.168.1.240:5000 -key=mykey

# 测试访问
curl http://x.x.x.x:2345
```
