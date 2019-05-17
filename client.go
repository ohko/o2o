package o2o

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ohko/omsg"
)

// Client 客户端
type Client struct {
	ocli                  *omsg.Client
	serverPort, proxyPort string
	conns                 sync.Map // map[浏览器IP:Port + 本地服务IP:Port]本地服务连接
}

// Start 启动客户端
func (o *Client) Start(key, serverPort, proxyPort string) error {
	o.serverPort, o.proxyPort = serverPort, proxyPort

	// setup AES
	if len(key) > 0 {
		aesEnable = true
		aesKey = sha256.Sum256([]byte(key))
		aesIV = md5.Sum([]byte(key))
		ll.Log4Trace("AES crypt enabled")
	} else {
		ll.Log4Trace("AES crypt disabled")
	}

	o.ocli = omsg.NewClient()
	o.ocli.OnData = o.onData
	o.ocli.OnClose = o.onClose
	return o.Reconnect()
}

func (o *Client) onClose() {
	ll.Log4Trace("connect closed:", o.serverPort, o.proxyPort)

	// 清理本地数据
	o.conns.Range(func(key, val interface{}) bool {
		o.conns.Delete(key)
		return true
	})

	// 断线后重连
	o.Reconnect()
}

func (o *Client) onData(cmd, ext uint16, data []byte) {
	var err error
	data = aesEncode(data)
	ll.Log0Debug(fmt.Sprintf("0x%x-0x%x:\n%s", cmd, ext, hex.Dump(data)))

	switch cmd {
	case CMDHEART:
		ll.Log4Trace(string(data))
	case CMDSUCCESS:
		ll.Log4Trace("tunnel success:", o.serverPort, string(data))
	case CMDFAILED:
		ll.Log2Error("tunnel failed:", o.serverPort, string(data))
		time.Sleep(time.Second * 5)
		o.ocli.Send(CMDTUNNEL, 0, aesEncode([]byte(o.proxyPort)))
	case CMDCLOSE:
		if conn, ok := o.conns.Load(string(data)); ok {
			ll.Log0Debug("关闭本地连接:", string(data))
			conn.(net.Conn).Close()
		}
	case CMDDATA:
		client, addr, browserData := deData(data)

		// 此浏览器的请求是否已有本地服务连接
		conn, ok := o.conns.Load(client + addr)
		if !ok {
			// 创建本地连接
			// addr[2345:192.168.1.240:5000]
			tmp := strings.Split(addr, ":")
			conn, err = net.Dial("tcp", strings.Join(tmp[1:], ":"))
			if err != nil {
				ll.Log2Error(err)
				// 通知本地服务连接失败
				if err := o.ocli.Send(CMDLOCALCLOSE, 0, aesEncode(enData(client, addr, []byte(err.Error())))); err != nil {
					ll.Log2Error(err)
				}
				return
			}
			o.conns.Store(client+addr, conn)

			// 监听本地服务数据
			go func(conn net.Conn) {
				buf := make([]byte, bufferSize)
				for {
					n, err := conn.Read(buf)
					if err != nil {
						return
					}

					// 数据转发到远端
					if err := o.ocli.Send(CMDDATA, 0, aesEncode(enData(client, addr, buf[:n]))); err != nil {
						ll.Log2Error(err)
						// 关闭本次通道
						conn.Close()
					}
				}
				ll.Log0Debug("local close:", conn.RemoteAddr())
			}(conn.(net.Conn))
		}

		// 远端数据转发到本地服务
		if _, err := conn.(net.Conn).Write(browserData); err != nil {
			ll.Log2Error(err)

			// 通知本地服务连接失败
			if err := o.ocli.Send(CMDLOCALCLOSE, 0, aesEncode(enData(client, addr, []byte(err.Error())))); err != nil {
				ll.Log2Error(err)
			}

			// 关闭本次通道
			conn.(net.Conn).Close()
		}
	}
}

// Reconnect 重新连接服务器
func (o *Client) Reconnect() error {
	for {
		if err := o.ocli.Connect(o.serverPort); err != nil {
			ll.Log2Error(err)
			time.Sleep(time.Second)
			continue
		}

		// 连接成功，发送Tunnel请求
		o.ocli.Send(CMDTUNNEL, 0, aesEncode([]byte(o.proxyPort)))
		break
	}
	ll.Log4Trace("connect success:", o.serverPort, o.proxyPort)
	return nil
}
