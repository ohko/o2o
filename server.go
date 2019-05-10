package o2o

import (
	"crypto/md5"
	"crypto/sha256"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ohko/omsg"
)

// Server 服务端
type Server struct {
	oserv      *omsg.Server
	serverPort string
	srvs       sync.Map
	brows      sync.Map
}

// Start 启动服务
func (o *Server) Start(key, serverPort string) error {
	o.serverPort = serverPort

	// setup AES
	if len(key) > 0 {
		aesEnable = true
		aesKey = sha256.Sum256([]byte(key))
		aesIV = md5.Sum([]byte(key))
		log.Println("AES crypt enabled")
	} else {
		log.Println("AES crypt disabled")
	}

	o.oserv = omsg.NewServer()
	o.oserv.OnData = o.onData
	o.oserv.OnNewClient = o.onNewClient
	o.oserv.OnClientClose = o.onClientClose
	go func() {
		log.Println("server:", o.serverPort)
		log.Println(o.oserv.StartServer(o.serverPort))
	}()

	return nil
}

func (o *Server) onNewClient(conn net.Conn) {
	log.Println("client connect:", conn.RemoteAddr())
}
func (o *Server) onClientClose(conn net.Conn) {
	// 释放tunnel监听的端口
	if web, ok := o.srvs.Load(conn); ok {
		log.Println("close port:", web.(net.Listener).Addr().String())
		web.(net.Listener).Close()
	}
	log.Println("client close:", conn.RemoteAddr())
}
func (o *Server) onData(conn net.Conn, cmd, ext uint16, data []byte) {
	data = aesEncode(data)
	// log.Printf("0x%x-0x%x:\n%s", cmd, ext, hex.Dump(data))

	switch cmd {
	case CMDMSG:
		log.Println(string(data))
	case CMDTUNNEL:
		t := &tunnelInfo{addr: string(data), conn: conn}
		go o.createListen(t)
	case CMDDATA:
		client, _, data := deData(aesEncode(data))
		// log.Println("向浏览器发送：\n" + hex.Dump(data))
		if browser, ok := o.brows.Load(client); ok {
			browser.(net.Conn).Write(data)
		}
	case CMDLOCALCLOSE:
		client, _, data := deData(aesEncode(data))
		log.Println("client server error:", string(data))
		if browser, ok := o.brows.Load(client); ok {
			browser.(net.Conn).Close()
		}
	}
}

// 8080:192.168.1.238:50000 请求服务器开启8080端口代理192.168.1.238的5000端口
func (o *Server) createListen(tunnel *tunnelInfo) {
	tmp := strings.Split(tunnel.addr, ":")
	if len(tmp) != 3 {
		if err := o.oserv.Send(tunnel.conn, CMDMSG, 0, aesEncode([]byte(`format error:`+tunnel.addr))); err != nil {
			log.Println(err)
		}
		return
	}
	port := tmp[0]

	// 监听服务端口
	web, err := net.Listen("tcp", ":"+port)
	if err != nil {
		defer tunnel.conn.Close()
		log.Println(err)
		if err := o.oserv.Send(tunnel.conn, CMDMSG, 0, aesEncode([]byte(err.Error()))); err != nil {
			log.Println(err)
		}
		return
	}
	tunnel.srv = web
	log.Println("listen:", tunnel.conn.RemoteAddr(), tunnel.addr)

	// 通知服务监听成功
	if err := o.oserv.Send(tunnel.conn, CMDSUCCESS, 0, aesEncode([]byte(tunnel.addr))); err != nil {
		log.Println(err)
		// 通知失败，退出监听
		web.Close()
		return
	}

	o.srvs.Store(tunnel.conn, web)
	defer o.srvs.Delete(tunnel.conn)

	// 接受browser连接
	for {
		conn, err := web.Accept()
		if err != nil {
			break
		}

		go o.handleBrowser(tunnel, conn)
	}

	log.Println("closed:", tunnel.conn.RemoteAddr(), tunnel.addr)
}

// 处理浏览器数据
func (o *Server) handleBrowser(tunnel *tunnelInfo, conn net.Conn) {
	// 保存/删除broswer连接
	o.brows.Store(conn.RemoteAddr().String(), conn)
	defer func() {
		o.brows.Delete(conn.RemoteAddr().String())

		// 通知浏览器关闭
		if err := o.oserv.Send(tunnel.conn, CMDCLOSE, 0, aesEncode([]byte(conn.RemoteAddr().String()+tunnel.addr))); err != nil {
			log.Println(err)
		}
	}()

	recvedData := make(chan int)
	go func(conn net.Conn) {
		for { // 本地超过30秒没有数据就关闭连接
			select {
			case <-time.After(time.Second * 30):
				// log.Println("\033[32m强制关闭:" + conn.RemoteAddr().String() + "\033[m")
				conn.Close()
				return
			case <-recvedData:
				// log.Printf("\033[32m收到数据:["+conn.RemoteAddr().String()+"] %d\033[m\n", n)
			}
		}
	}(conn)

	buf := make([]byte, bufferSize)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		recvedData <- n

		// 浏览器数据转发到client
		if err := o.oserv.Send(tunnel.conn, CMDDATA, 0, aesEncode(enData(conn.RemoteAddr().String(), tunnel.addr, buf[:n]))); err != nil {
			log.Println(err)
			// 发送错误，关闭连接
			conn.Close()
			break
		}
	}
}
