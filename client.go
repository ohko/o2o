package o2o

import (
	"crypto/md5"
	"crypto/sha256"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ohko/omsg"
)

// Client 客户端
type Client struct {
	msg                   *omsg.Client
	serverPort, proxyPort string
	serves                sync.Map // map[浏览器IP:Port + 本地服务IP:Port]本地服务连接
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

	o.msg = omsg.NewClient(o)
	return o.Reconnect()
}

// OmsgError ...
func (o *Client) OmsgError(err error) { ll.Log2Error(err) }

// OmsgClose ...
func (o *Client) OmsgClose() {
	ll.Log4Trace("connect closed:", o.serverPort, o.proxyPort)

	// 清理本地数据
	o.serves.Range(func(key, val interface{}) bool {
		o.serves.Delete(key)
		return true
	})

	// 断线后重连
	o.Reconnect()
}

// OmsgData ...
func (o *Client) OmsgData(cmd, ext uint16, data []byte) {
	data = aesCrypt(data)
	// ll.Log0Debug(fmt.Sprintf("0x%x-0x%x:\n%s", cmd, ext, hex.Dump(data)))

	switch cmd {
	case CMDSUCCESS:
		ll.Log4Trace("tunnel success:", o.serverPort, string(data))
	case CMDFAILED:
		ll.Log2Error("tunnel failed:", o.serverPort, string(data))
		time.Sleep(time.Second * 5)
		o.Send(CMDTUNNEL, 0, []byte(o.proxyPort))
	case CMDCLOSE:
		if conn, ok := o.serves.Load(string(data)); ok {
			ll.Log0Debug("关闭本地连接:", string(data))
			conn.(net.Conn).Close()
		}
	case CMDDATA:
		client, addr, browserData := deData(data)
		if err := o.dispose(client, addr, browserData); err != nil {
			// 通知本地服务错误
			if err = o.Send(CMDLOCALCLOSE, 0, enData(client, addr, []byte(err.Error()))); err != nil {
				ll.Log2Error(err)
			}
		}
	}
}

// Send 原始数据加密后发送
func (o *Client) Send(cmd, ext uint16, originData []byte) error {
	return omsg.Send(o.msg.Conn, cmd, ext, aesCrypt(originData))
}

// Reconnect 重新连接服务器
func (o *Client) Reconnect() error {
	for {
		if err := o.msg.Connect(o.serverPort); err != nil {
			ll.Log2Error(err)
			time.Sleep(time.Second)
			continue
		}

		// 连接成功，发送Tunnel请求
		o.Send(CMDTUNNEL, 0, []byte(o.proxyPort))
		break
	}
	ll.Log4Trace("connect success:", o.serverPort, o.proxyPort)

	return nil
}

// 处理数据
func (o *Client) dispose(client, addr string, data []byte) (err error) {
	var serve net.Conn

	// 此浏览器的请求是否已有本地服务连接
	if _conn, ok := o.serves.Load(client + addr); ok {
		serve = _conn.(net.Conn)
	}

	// 本地还没有与服务建立连接，创建一个新的服务连接
	if serve == nil {
		// addr[0.0.0.0:2345:192.168.1.240:5000]
		tmp := strings.Split(addr, ":")
		serve, err = net.Dial("tcp", strings.Join(tmp[2:], ":"))
		if err != nil {
			return err
		}
		ll.Log0Debug("create local connect:", serve.LocalAddr())
		o.serves.Store(client+addr, serve)

		go func() {
			// 转发服务的数据
			io.Copy(&pipeClient{serve: serve, cli: o, client: client, addr: addr}, serve)
			ll.Log0Debug("local close:", serve.LocalAddr())
			o.serves.Delete(client + addr)
		}()
	}

	// 数据转发到本地服务
	if _, err := serve.Write(data); err != nil {
		// 关闭本次通道
		serve.Close()
		return err
	}

	return nil
}

// 数据通道
type pipeClient struct {
	serve        net.Conn
	cli          *Client
	client, addr string
}

func (o *pipeClient) Write(p []byte) (n int, err error) {
	// 数据转发到远端
	if err := o.cli.Send(CMDDATA, 0, enData(o.client, o.addr, p)); err != nil {
		// 关闭本次通道
		o.serve.Close()
		return 0, err
	}

	return len(p), nil
}
