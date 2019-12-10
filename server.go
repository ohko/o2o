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
	srv        *Server
	tunnelAddr string   // tunnel请求端口
	clientConn net.Conn // client -> server
}

// 浏览器信息
type browserInfo struct {
	browserConn net.Conn // 浏览器连接
	tunnel      *tunnelInfo
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
func (o *Server) OmsgError(conn net.Conn, err error) {
	if err != io.EOF {
		ll.Log2Error(err)
	}
}

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
	case cmdTunnel:
		t := &tunnelInfo{srv: o, tunnelAddr: string(data), clientConn: conn}
		if err := o.createListen(t); err != nil {
			if err := o.Send(conn, cmdTunnelFailed, 0, []byte(err.Error())); err != nil {
				ll.Log2Error(err)
			}
		} else {
			if err := o.Send(conn, cmdTunnelSuccess, 0, data); err != nil {
				ll.Log2Error(err)
			}
		}
	case cmdData:
		client, _, data := deData(data)
		if browser, ok := o.brows.Load(client); ok {
			// ll.Log0Debug("向浏览器发送：\n" + hex.Dump(data))
			if _, err := browser.(*browserInfo).browserConn.Write(data); err != nil {
				ll.Log0Debug("browser write err:", err)
				// 通知浏览器关闭
				if err := o.Send(conn, cmdBrowserClose, 0, []byte(conn.RemoteAddr().String()+browser.(*browserInfo).tunnel.tunnelAddr)); err != nil {
					ll.Log2Error(err)
				}
			}
		}
	case cmdLocaSrveClose:
		client, _, data := deData(data)
		ll.Log4Trace("client server error:", string(data))
		if browser, ok := o.brows.Load(client); ok {
			browser.(*browserInfo).tunnel.clientConn.Close()
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
	tmp := strings.Split(tunnel.tunnelAddr, ":")
	if len(tmp) != 4 {
		return fmt.Errorf(`format error:` + tunnel.tunnelAddr)
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
	o.webs.Store(tunnel.clientConn, web)
	ll.Log4Trace("tunnel:", port, tunnel.clientConn.RemoteAddr(), tunnel.tunnelAddr)

	go func() {
		defer func() {
			ll.Log4Trace("closed:", port, tunnel.clientConn.RemoteAddr(), tunnel.tunnelAddr)
			web.Close()
			tunnel.clientConn.Close()
			o.webs.Delete(tunnel.clientConn)
		}()

		// 接受browser连接
		for {
			browserConn, err := web.Accept()
			if err != nil {
				break
			}
			ll.Log0Debug("new brows:", browserConn.RemoteAddr())

			brow := &browserInfo{browserConn: browserConn, tunnel: tunnel}
			// 互相COPY数据
			go func(browserConn net.Conn) {
				defer func() {
					browserConn.Close()
					// 通知浏览器关闭
					ll.Log0Debug("browser close:", browserConn.RemoteAddr().String())
					if err := o.Send(tunnel.clientConn, cmdBrowserClose, 0, []byte(browserConn.RemoteAddr().String()+tunnel.tunnelAddr)); err != nil {
						ll.Log2Error(err)
					}
				}()

				io.Copy(brow, browserConn)
				ll.Log0Debug("browser end:", browserConn.RemoteAddr().String())
				o.brows.Delete(browserConn.RemoteAddr().String())
			}(browserConn)
			o.brows.Store(browserConn.RemoteAddr().String(), brow)
		}
	}()

	return nil
}

func (o *browserInfo) Write(p []byte) (n int, err error) {
	// 浏览器数据转发到client
	if err := o.tunnel.srv.Send(o.tunnel.clientConn, cmdData, 0, enData(o.browserConn.RemoteAddr().String(), o.tunnel.tunnelAddr, p)); err != nil {
		ll.Log2Error(err)
		o.browserConn.Close()
		// 发送错误，关闭连接
		return 0, err
	}
	return len(p), nil
}
