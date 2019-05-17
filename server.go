package o2o

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/ohko/omsg"
)

// Server 服务端
type Server struct {
	oserv      *omsg.Server
	serverPort string
	srvs       sync.Map
	brows      sync.Map
}

type tunnelInfo struct {
	oserv *omsg.Server
	addr  string   // tunnel请求端口
	conn  net.Conn // client -> server
}

// 浏览器信息
type browserInfo struct {
	conn   net.Conn // 浏览器连接
	tunnel *tunnelInfo
}

// Start 启动服务
func (o *Server) Start(key, serverPort string) error {
	o.serverPort = serverPort

	// setup AES
	if len(key) > 0 {
		aesEnable = true
		aesKey = sha256.Sum256([]byte(key))
		aesIV = md5.Sum([]byte(key))
		ll.Log4Trace("AES crypt enabled")
	} else {
		ll.Log4Trace("AES crypt disabled")
	}

	o.oserv = omsg.NewServer()
	o.oserv.OnData = o.onData
	o.oserv.OnNewClient = o.onNewClient
	o.oserv.OnClientClose = o.onClientClose
	go func() {
		ll.Log4Trace("server:", o.serverPort)
		ll.Log4Trace(o.oserv.StartServer(o.serverPort))
	}()

	return nil
}

func (o *Server) onNewClient(conn net.Conn) {
	ll.Log4Trace("client connect:", conn.RemoteAddr())
}
func (o *Server) onClientClose(conn net.Conn) {
	// 释放tunnel监听的端口
	if web, ok := o.srvs.Load(conn); ok {
		ll.Log4Trace("close port:", web.(net.Listener).Addr().String())
		web.(net.Listener).Close()
		o.srvs.Delete(conn)
	}
	ll.Log4Trace("client close:", conn.RemoteAddr())
}
func (o *Server) onData(conn net.Conn, cmd, ext uint16, data []byte) {
	data = aesEncode(data)
	ll.Log0Debug(fmt.Sprintf("0x%x-0x%x:\n%s", cmd, ext, hex.Dump(data)))

	switch cmd {
	case CMDHEART:
		ll.Log4Trace(string(data))
	case CMDTUNNEL:
		t := &tunnelInfo{oserv: o.oserv, addr: string(data), conn: conn}
		if err := o.createListen(t); err != nil {
			if err := o.oserv.Send(conn, CMDFAILED, 0, aesEncode([]byte(err.Error()))); err != nil {
				ll.Log2Error(err)
			}
		} else {
			if err := o.oserv.Send(conn, CMDSUCCESS, 0, aesEncode(data)); err != nil {
				ll.Log2Error(err)
			}
		}
	case CMDDATA:
		client, _, data := deData(data)
		if browser, ok := o.brows.Load(client); ok {
			// ll.Log0Debug("向浏览器发送：\n" + hex.Dump(data))
			if _, err := browser.(*browserInfo).conn.Write(data); err != nil {
				ll.Log0Debug("browser write err:", err)
				// 通知浏览器关闭
				if err := o.oserv.Send(conn, CMDCLOSE, 0, aesEncode([]byte(conn.RemoteAddr().String()+browser.(*browserInfo).tunnel.addr))); err != nil {
					ll.Log2Error(err)
				}
			}
		}
	case CMDLOCALCLOSE:
		client, _, data := deData(data)
		ll.Log4Trace("client server error:", string(data))
		if browser, ok := o.brows.Load(client); ok {
			browser.(*browserInfo).conn.Close()
			browser.(*browserInfo).tunnel.conn.Close()
		}
	}
}

// 8080:192.168.1.238:50000 请求服务器开启8080端口代理192.168.1.238的5000端口
func (o *Server) createListen(tunnel *tunnelInfo) error {
	// 检查建立通道的参数
	tmp := strings.Split(tunnel.addr, ":")
	if len(tmp) != 3 {
		return fmt.Errorf(`format error:` + tunnel.addr)
	}
	port := tmp[0]

	// 监听服务端口
	web, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	o.srvs.Store(tunnel.conn, web)
	ll.Log4Trace("listen:", ":"+port, tunnel.conn.RemoteAddr(), tunnel.addr)

	go func() {
		defer o.srvs.Delete(tunnel.conn)
		defer tunnel.conn.Close()
		defer web.Close()
		defer ll.Log4Trace("closed:", ":"+port, tunnel.conn.RemoteAddr(), tunnel.addr)

		// 接受browser连接
		for {
			conn, err := web.Accept()
			if err != nil {
				break
			}
			ll.Log0Debug("new brows:", conn.RemoteAddr())

			brow := &browserInfo{conn: conn, tunnel: tunnel}
			// 互相COPY数据
			go func() {
				io.Copy(brow, conn)
				// 通知浏览器关闭
				ll.Log0Debug("通知浏览器关闭")
				if err := o.oserv.Send(tunnel.conn, CMDCLOSE, 0, aesEncode([]byte(conn.RemoteAddr().String()+tunnel.addr))); err != nil {
					ll.Log2Error(err)
				}
			}()
			o.brows.Store(conn.RemoteAddr().String(), brow)
		}
	}()

	return nil
}

func (o *browserInfo) Write(p []byte) (n int, err error) {
	// 浏览器数据转发到client
	if err := o.tunnel.oserv.Send(o.tunnel.conn, CMDDATA, 0, aesEncode(enData(o.conn.RemoteAddr().String(), o.tunnel.addr, p))); err != nil {
		ll.Log2Error(err)
		o.conn.Close()
		// 发送错误，关闭连接
		return 0, err
	}
	return len(p), nil
}
