package main

import (
	"flag"
	"io"
	"log"
	"net"
	"o2o"
	"strings"
	"sync"
)

var (
	clients    map[net.Conn]net.Listener
	webClients map[string]net.Conn
	lock       sync.Mutex

	serverPort = flag.String("s", ":2399", "外网服务器服务端口，用于接收内网服务器连接")
)

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	clients = make(map[net.Conn]net.Listener)
	webClients = make(map[string]net.Conn)

	startServer()
	o2o.CatchCtrlC()
}

func startServer() {
	server, err := net.Listen("tcp", *serverPort)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("server:", *serverPort)

	for {
		conn, err := server.Accept()
		if err != nil {
			break
		}

		go listen(conn)
	}
}

func listen(conn net.Conn) {
	defer func() {
		// 断开
		lock.Lock()
		defer lock.Unlock()
		if v, ok := clients[conn]; ok {
			v.Close()
			delete(clients, conn)
		}
	}()

	for {
		data, err := o2o.Recv(conn)
		if err != nil {
			return
		}

		cmd, ext := string(data[0]), string(data[1:])
		switch string(cmd) {
		case o2o.CMDCLIENT:
			log.Println("req:", o2o.Conn2IP(conn), ext)
			go createListen(ext, conn)
		case o2o.CMDTENNEL:
			go func() {
				lock.Lock()
				defer lock.Unlock()
				go io.Copy(webClients[ext], conn)
				go io.Copy(conn, webClients[ext])
			}()
			return // 必须有，结束后返回，不要再recv数据了
		}
	}
}

// 8080:192.168.1.238:50000 请求服务器开启8080端口代理192.168.1.238的5000端口
func createListen(addr string, client net.Conn) {
	tmp := strings.Split(addr, ":")
	if len(tmp) != 3 {
		if err := o2o.Send(client, o2o.CMDMSG, `format error:`+addr); err != nil {
			log.Println(err)
		}
		return
	}
	port := tmp[0]

	web, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Println(err)
		if err := o2o.Send(client, o2o.CMDMSG, err.Error()); err != nil {
			log.Println(err)
		}
		return
	}
	log.Println("listen:", port)

	lock.Lock()
	clients[client] = web
	lock.Unlock()

	for {
		conn, err := web.Accept()
		if err != nil {
			break
		}
		go func(conn net.Conn) {
			lock.Lock()
			defer lock.Unlock()

			webClients[conn.RemoteAddr().String()] = conn
			if err := o2o.Send(client, o2o.CMDREQUEST, conn.RemoteAddr().String()); err != nil {
				log.Println(err)
			}
		}(conn)
	}
	log.Println("close:", port)
}
