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

	if len(key) > 0 {
		aesEnable = true
		bsKey := sha256.Sum256([]byte(key))
		bsIV := md5.Sum([]byte(key))
		copy(aesKey[:], bsKey[:])
		copy(aesIV[:], bsIV[:])
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
	if web, ok := o.srvs.Load(conn); ok {
		web.(net.Listener).Close()
	}
	o.srvs.Delete(conn)
	log.Println("client close:", conn.RemoteAddr())
}
func (o *Server) onData(conn net.Conn, cmd, ext uint16, data []byte, err error) {
	if err != nil {
		outFakeMsg(conn)
		return
	}

	data = aesEncode(data)

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

	web, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Println(err)
		if err := o.oserv.Send(tunnel.conn, CMDMSG, 0, aesEncode([]byte(err.Error()))); err != nil {
			log.Println(err)
		}
		return
	}
	tunnel.srv = web
	log.Println("listen:", port)
	o.srvs.Store(tunnel.conn, web)

	if err := o.oserv.Send(tunnel.conn, CMDSUCCESS, 0, aesEncode([]byte(tunnel.addr))); err != nil {
		log.Println(err)
	}

	for {
		conn, err := web.Accept()
		if err != nil {
			break
		}
		o.brows.Store(conn.RemoteAddr().String(), conn)

		recvedData := make(chan int)
		go func() {
			for { // 本地超过5秒没有数据就关闭连接
				select {
				case <-time.After(time.Second * 15):
					// log.Println("\033[32m强制关闭:" + conn.RemoteAddr().String() + "\033[m")
					conn.Close()
					return
				case <-recvedData:
					// log.Printf("\033[32m收到数据:["+conn.RemoteAddr().String()+"] %d\033[m\n", n)
				}
			}
		}()

		go func(conn net.Conn) {
			buf := make([]byte, bufferSize)
			for {
				n, err := conn.Read(buf)
				if err != nil {
					break
				}
				recvedData <- n

				// log.Println("转发浏览器数据：\n" + hex.Dump(buf[:n]))
				if err := o.oserv.Send(tunnel.conn, CMDDATA, 0, aesEncode(enData(conn.RemoteAddr().String(), tunnel.addr, buf[:n]))); err != nil {
					log.Println(err)
				}
			}
			if err := o.oserv.Send(tunnel.conn, CMDCLOSE, 0, aesEncode([]byte(conn.RemoteAddr().String()+tunnel.addr))); err != nil {
				log.Println(err)
			}
			o.brows.Delete(conn.RemoteAddr().String())
		}(conn)
	}

	o.srvs.Delete(tunnel.conn)
	log.Println("close:", port)
}
