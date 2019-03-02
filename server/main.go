package main

import (
	"flag"
	"io"
	"log"
	"net"
	"o2o"
	"strings"
	"sync"

	"github.com/ohko/omsg"
)

var (
	server     *omsg.Server
	proxy      map[net.Conn]net.Listener
	webClients map[string]net.Conn
	lock       sync.Mutex

	serverPort = flag.String("s", ":2399", "外网服务器服务端口，用于接收内网服务器连接")
	tunnelPort = flag.String("t", ":2398", "内网需服务连接到外网服务器的隧道端口")
)

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	proxy = make(map[net.Conn]net.Listener)
	webClients = make(map[string]net.Conn)

	go tunnelServer()

	server = omsg.NewServer()
	server.OnNewClient = onNewClient
	server.OnClientClose = onClientClose
	server.OnData = onData
	log.Println("server:", *serverPort)
	log.Println(server.StartServer(*serverPort))
}

func onNewClient(conn net.Conn) {
	// log.Println("new client:", conn.RemoteAddr())
}

func onClientClose(conn net.Conn) {
	// log.Println("client disconnect:", conn.RemoteAddr())
	lock.Lock()
	defer lock.Unlock()

	if v, ok := proxy[conn]; ok {
		v.Close()
		delete(proxy, conn)
	}
}

func onData(conn net.Conn, data []byte) {
	tmp := strings.Split(string(data), "|")
	cmd, ext := tmp[0], tmp[1]
	if cmd == "proxy" {
		// 8080:192.168.1.238:50000 请求服务器开启8080端口代理192.168.1.238的5000端口
		log.Println("req:", o2o.Conn2IP(conn), ext)

		go createWeb(ext, conn)
	}
}

func createWeb(addr string, client net.Conn) {
	tmp := strings.Split(addr, ":")
	if len(tmp) != 3 {
		if err := server.Send(client, []byte(`error|addr format error:`+addr)); err != nil {
			log.Println(err)
		}
		return
	}
	port := tmp[0]

	web, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Println(err)
		if err := server.Send(client, []byte(`error|`+err.Error())); err != nil {
			log.Println(err)
		}
		return
	}
	log.Println("listen:", port)
	lock.Lock()
	proxy[client] = web
	lock.Unlock()
	if err := server.Send(client, []byte(`listen|`+addr)); err != nil {
		log.Println(err)
	}
	for {
		conn, err := web.Accept()
		if err != nil {
			break
		}
		go func(conn net.Conn) {
			lock.Lock()
			defer lock.Unlock()

			webClients[conn.RemoteAddr().String()] = conn
			if err := server.Send(client, []byte(`tunnel|`+conn.RemoteAddr().String())); err != nil {
				log.Println(err)
			}
		}(conn)
	}
	log.Println("close:", port)
}

func tunnelServer() {
	tunnel, err := net.Listen("tcp", *tunnelPort)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("tunnel:", *tunnelPort)
	for {
		conn, err := tunnel.Accept()
		if err != nil {
			log.Println(err)
			return
		}

		tmp, err := o2o.Recv(conn)
		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			lock.Lock()
			defer lock.Unlock()
			go io.Copy(webClients[string(tmp)], conn)
			go io.Copy(conn, webClients[string(tmp)])
		}()
	}
}
