# TCP 隧道

## Server
```
# 开启2399等待Client连接，传送指令
# 开启2398用于创建TCP隧道
go run ./server -s :2399 -t :2398
```

## Client
```
# 连接服务器2399端口，传送指令
# 设定服务器用于创建TCP隧道的端口2398
# 请求服务器开启2345端口，用于代理192.168.1.238的5000端口
go run ./client -s x.x.x.x:2399 -t x.x.x.x:2398 -p 2345:192.168.1.238:5000
```

# Docker
```
# Server 开启2300-2399端口段
docker run --name=o2o -d --restart=always -p 2300-2399:2300-2399 ohko/o2o /server -s :2399 -t :2398

# Client 请求2345端口代理192.168.1.238的5000端口
docker run --name=o2o -d --restart=always ohko/o2o /client -s x.x.x.x:2399 -t x.x.x.x:2398 -p 2345:192.168.1.238:5000

# 请求
curl http://x.x.x.x:2345
```