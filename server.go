package o2o

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ohko/omsg"
)

// Server 服务端
type Server struct {
	msg        *omsg.Server
	serverPort string
	clients    sync.Map // client服务监听
	users      sync.Map // 浏览器连接
}

type tunnelInfo struct {
	srv        *Server
	tunnelAddr string   // tunnel请求端口
	clientConn net.Conn // client -> server
}

// user信息
type userInfo struct {
	userConn   net.Conn // 浏览器连接
	tunnel     *tunnelInfo
	data       chan []byte
	dataClosed int64 // 0=未关闭/1=已关闭
}

// Start 启动服务
func (o *Server) Start(key, serverPort string, crc bool) error {
	lServer.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	lServer.SetPrefix("S")
	lServer.SetColor(true)
	o.serverPort = serverPort

	// setup AES
	if len(key) > 0 {
		aesEnable = true
		aesKey = sha256.Sum256([]byte(key))
		aesIV = md5.Sum([]byte(key))
		lServer.Log4Trace("AES crypt enabled")
	} else {
		lServer.Log4Trace("AES crypt disabled")
	}

	o.msg = omsg.NewServer(o, crc)
	go func() {
		lServer.Log4Trace("server:", o.serverPort)
		lServer.Log4Trace(o.msg.StartServer(o.serverPort))
	}()

	return nil
}

// OmsgError ...
func (o *Server) OmsgError(conn net.Conn, err error) {
	if err != io.EOF {
		lServer.Log2Error(err)
	}
}

// OmsgNewClient ...
func (o *Server) OmsgNewClient(conn net.Conn) {
	lServer.Log0Debug("client connect:", conn.RemoteAddr())
}

// OmsgClientClose ...
func (o *Server) OmsgClientClose(conn net.Conn) {
	// 释放tunnel监听的端口
	if client, ok := o.clients.Load(conn); ok {
		lServer.Log0Debug("close port:", client.(net.Listener).Addr().String())
		client.(net.Listener).Close()
		o.clients.Delete(conn)
	}
	lServer.Log0Debug("client close:", conn.RemoteAddr())
}

// OmsgData ...
func (o *Server) OmsgData(conn net.Conn, cmd, ext uint16, data []byte) {
	data = aesCrypt(data)
	// lServer.Log0Debug(fmt.Sprintf("0x%x-0x%x:\n%s", cmd, ext, hex.Dump(data)))

	switch cmd {
	case cmdTunnel:
		t := &tunnelInfo{srv: o, tunnelAddr: string(data), clientConn: conn}
		if err := o.createListen(t); err != nil {
			if err := o.Send(conn, cmdTunnelFailed, 0, []byte(err.Error())); err != nil {
				lServer.Log2Error(err)
			}
		} else {
			if err := o.Send(conn, cmdTunnelSuccess, 0, data); err != nil {
				lServer.Log2Error(err)
			}
		}
	case cmdData:
		client, _, data := deData(data)
		if user, ok := o.users.Load(client); ok {
			user.(*userInfo).data <- data
		}
	case cmdLocaSrveClose:
		client, _, data := deData(data)
		lServer.Log0Debug("client server error:", string(data))
		if user, ok := o.users.Load(client); ok {
			if atomic.CompareAndSwapInt64(&user.(*userInfo).dataClosed, 0, 1) {
				close(user.(*userInfo).data)
			}
			lServer.Log0Debug("close user:", client)
		}
	}
}

// Send 原始数据加密后发送
func (o *Server) Send(conn net.Conn, cmd, ext uint16, originData []byte) error {
	return o.msg.Send(conn, cmd, ext, aesCrypt(originData))
}

// 0.0.0.0:8080:192.168.1.238:50000 请求服务器开启8080端口代理192.168.1.238的5000端口
func (o *Server) createListen(tunnel *tunnelInfo) error {
	// 检查建立通道的参数
	tmp := strings.Split(tunnel.tunnelAddr, ":")
	if len(tmp) != 4 {
		return fmt.Errorf(`format error:` + tunnel.tunnelAddr)
	}
	address := strings.Join(tmp[:2], ":")
	port := tmp[1]

	// 关闭之前的此端口
	o.clients.Range(func(key, val interface{}) bool {
		ss := strings.Split(val.(net.Listener).Addr().String(), ":")
		if port == ss[len(ss)-1] {
			lServer.Log1Warn("close before listener:", address)
			val.(net.Listener).Close()
			time.Sleep(time.Second)
			return false
		}
		return true
	})

	// 监听服务端口
	client, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	o.clients.Store(tunnel.clientConn, client)
	lServer.Log0Debug("tunnel:", address, tunnel.clientConn.RemoteAddr(), tunnel.tunnelAddr)

	go func() {
		defer func() {
			lServer.Log0Debug("closed:", address, tunnel.clientConn.RemoteAddr(), tunnel.tunnelAddr)
			client.Close()
			tunnel.clientConn.Close()
			o.clients.Delete(tunnel.clientConn)
		}()

		// 接受user连接
		for {
			userConn, err := client.Accept()
			if err != nil {
				break
			}
			lServer.Log0Debug("new user:", userConn.RemoteAddr())

			go o.newUser(&userInfo{userConn: userConn, tunnel: tunnel, data: make(chan []byte)})
		}
	}()

	return nil
}

func (o *Server) newUser(user *userInfo) {
	o.users.Store(user.userConn.RemoteAddr().String(), user)

	// 互相COPY数据
	defer func() {
		if atomic.CompareAndSwapInt64(&user.dataClosed, 0, 1) {
			close(user.data)
		}
		user.userConn.Close()
		o.users.Delete(user.userConn.RemoteAddr().String())
		// 通知user关闭
		lServer.Log0Debug("user close:", user.userConn.RemoteAddr().String())
		if err := o.Send(user.tunnel.clientConn, cmdUserClose, 0, []byte(user.userConn.RemoteAddr().String()+user.tunnel.tunnelAddr)); err != nil {
			lServer.Log2Error(err)
		}
	}()

	go func() {
		io.Copy(user.userConn, user)
		user.userConn.Close()
	}()
	io.Copy(user, user.userConn)
	lServer.Log0Debug("user end:", user.userConn.RemoteAddr().String())
}

func (o *userInfo) Write(p []byte) (n int, err error) {
	// user数据转发到client
	if err := o.tunnel.srv.Send(o.tunnel.clientConn, cmdData, 0, enData(o.userConn.RemoteAddr().String(), o.tunnel.tunnelAddr, p)); err != nil {
		lServer.Log2Error(err)
		o.userConn.Close()
		// 发送错误，关闭连接
		return 0, err
	}
	return len(p), nil
}

func (o *userInfo) Read(p []byte) (n int, err error) {
	// client数据转发到user
	data, ok := <-o.data
	if !ok {
		return 0, io.EOF
	}
	if len(p) >= len(data) {
		copy(p, data)
	} else {
		p = data
	}
	return len(data), nil
}
