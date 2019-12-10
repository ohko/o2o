package o2o

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ohko/omsg"
)

// Server 服务端
type Server struct {
	msg        *omsg.Server
	serverPort string
	webs       sync.Map // web服务监听
	brows      sync.Map // 浏览器连接
}

type tunnelInfo struct {
	srv  *Server
	addr string   // tunnel请求端口
	conn net.Conn // client -> server
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

	o.msg = omsg.NewServer(o)
	go func() {
		ll.Log4Trace("server:", o.serverPort)
		ll.Log4Trace(o.msg.StartServer(o.serverPort))
	}()

	return nil
}

// OmsgError ...
func (o *Server) OmsgError(conn net.Conn, err error) { ll.Log2Error(err) }

// OmsgNewClient ...
func (o *Server) OmsgNewClient(conn net.Conn) {
	ll.Log4Trace("client connect:", conn.RemoteAddr())
}

// OmsgClientClose ...
func (o *Server) OmsgClientClose(conn net.Conn) {
	// 释放tunnel监听的端口
	if web, ok := o.webs.Load(conn); ok {
		ll.Log4Trace("close port:", web.(net.Listener).Addr().String())
		web.(net.Listener).Close()
		o.webs.Delete(conn)
	}
	ll.Log4Trace("client close:", conn.RemoteAddr())
}

// OmsgData ...
func (o *Server) OmsgData(conn net.Conn, cmd, ext uint16, data []byte) {
	data = aesCrypt(data)
	// ll.Log0Debug(fmt.Sprintf("0x%x-0x%x:\n%s", cmd, ext, hex.Dump(data)))

	switch cmd {
	case CMDTUNNEL:
		t := &tunnelInfo{srv: o, addr: string(data), conn: conn}
		if err := o.createListen(t); err != nil {
			if err := o.Send(conn, CMDFAILED, 0, []byte(err.Error())); err != nil {
				ll.Log2Error(err)
			}
		} else {
			if err := o.Send(conn, CMDSUCCESS, 0, data); err != nil {
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
				if err := o.Send(conn, CMDCLOSE, 0, []byte(conn.RemoteAddr().String()+browser.(*browserInfo).tunnel.addr)); err != nil {
					ll.Log2Error(err)
				}
			}
		}
	case CMDLOCALCLOSE:
		client, _, data := deData(data)
		ll.Log4Trace("client server error:", string(data))
		if browser, ok := o.brows.Load(client); ok {
			browser.(*browserInfo).tunnel.conn.Close()
		}
	}
}

// Send 原始数据加密后发送
func (o *Server) Send(conn net.Conn, cmd, ext uint16, originData []byte) error {
	return omsg.Send(conn, cmd, ext, aesCrypt(originData))
}

// 0.0.0.0:8080:192.168.1.238:50000 请求服务器开启8080端口代理192.168.1.238的5000端口
func (o *Server) createListen(tunnel *tunnelInfo) error {
	// 检查建立通道的参数
	tmp := strings.Split(tunnel.addr, ":")
	if len(tmp) != 4 {
		return fmt.Errorf(`format error:` + tunnel.addr)
	}
	port := strings.Join(tmp[:2], ":")

	// 关闭之前的此端口
	o.webs.Range(func(key, val interface{}) bool {
		ss := strings.Split(val.(net.Listener).Addr().String(), ":")
		if port == ss[len(ss)-1] {
			ll.Log1Warn("close before listener:", port)
			val.(net.Listener).Close()
			time.Sleep(time.Second)
			return false
		}
		return true
	})

	// 监听服务端口
	web, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	o.webs.Store(tunnel.conn, web)
	ll.Log4Trace("listen:", port, tunnel.conn.RemoteAddr(), tunnel.addr)

	go func() {
		defer func() {
			ll.Log4Trace("closed:", port, tunnel.conn.RemoteAddr(), tunnel.addr)
			web.Close()
			tunnel.conn.Close()
			o.webs.Delete(tunnel.conn)
		}()

		// 接受browser连接
		for {
			conn, err := web.Accept()
			if err != nil {
				break
			}
			ll.Log0Debug("new brows:", conn.RemoteAddr())

			brow := &browserInfo{conn: conn, tunnel: tunnel}
			// 互相COPY数据
			go func(conn net.Conn) {
				defer func() {
					conn.Close()
					// 通知浏览器关闭
					ll.Log0Debug("browser close:", conn.RemoteAddr().String())
					if err := o.Send(tunnel.conn, CMDCLOSE, 0, []byte(conn.RemoteAddr().String()+tunnel.addr)); err != nil {
						ll.Log2Error(err)
					}
				}()

				io.Copy(brow, conn)
			}(conn)
			o.brows.Store(conn.RemoteAddr().String(), brow)
		}
	}()

	return nil
}

func (o *browserInfo) Write(p []byte) (n int, err error) {
	// 浏览器数据转发到client
	if err := o.tunnel.srv.Send(o.tunnel.conn, CMDDATA, 0, enData(o.conn.RemoteAddr().String(), o.tunnel.addr, p)); err != nil {
		ll.Log2Error(err)
		o.conn.Close()
		// 发送错误，关闭连接
		return 0, err
	}
	return len(p), nil
}
