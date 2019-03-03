package o2o

import (
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

// Server 服务端
type Server struct {
	serverPort string
	clients    map[net.Conn]net.Listener
	webClients map[string]net.Conn
	lock       sync.Mutex
}

// Start 启动服务
func (o *Server) Start(serverPort string) error {
	o.serverPort = serverPort
	o.clients = make(map[net.Conn]net.Listener)
	o.webClients = make(map[string]net.Conn)

	server, err := net.Listen("tcp", o.serverPort)
	if err != nil {
		return err
	}
	log.Println("server:", o.serverPort)

	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				break
			}

			go o.listen(conn)
		}
	}()

	return nil
}

func (o *Server) listen(conn net.Conn) {
	defer func() {
		// 断开
		o.lock.Lock()
		defer o.lock.Unlock()
		if v, ok := o.clients[conn]; ok {
			v.Close()
			delete(o.clients, conn)
		}
	}()

	for {
		data, err := recv(conn)
		if err != nil {
			return
		}

		cmd, ext := string(data[0]), string(data[1:])
		switch string(cmd) {
		case CMDCLIENT:
			log.Println("req:", conn2IP(conn), ext)
			go o.createListen(ext, conn)
		case CMDTENNEL:
			go func() {
				o.lock.Lock()
				defer o.lock.Unlock()
				go io.Copy(o.webClients[ext], conn)
				go io.Copy(conn, o.webClients[ext])
			}()
			return // 必须有，结束后返回，不要再recv数据了
		}
	}
}

// 8080:192.168.1.238:50000 请求服务器开启8080端口代理192.168.1.238的5000端口
func (o *Server) createListen(addr string, client net.Conn) {
	tmp := strings.Split(addr, ":")
	if len(tmp) != 3 {
		if err := send(client, CMDMSG, `format error:`+addr); err != nil {
			log.Println(err)
		}
		return
	}
	port := tmp[0]

	web, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Println(err)
		if err := send(client, CMDMSG, err.Error()); err != nil {
			log.Println(err)
		}
		return
	}
	log.Println("listen:", port)

	if err := send(client, CMDSUCCESS, addr); err != nil {
		log.Println(err)
	}

	o.lock.Lock()
	o.clients[client] = web
	o.lock.Unlock()

	for {
		conn, err := web.Accept()
		if err != nil {
			break
		}
		go func(conn net.Conn) {
			o.lock.Lock()
			defer o.lock.Unlock()

			o.webClients[conn.RemoteAddr().String()] = conn
			if err := send(client, CMDREQUEST, conn.RemoteAddr().String()); err != nil {
				log.Println(err)
			}
		}(conn)
	}
	log.Println("close:", port)
}
