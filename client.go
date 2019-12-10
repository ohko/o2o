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
func (o *Client) OmsgError(err error) {
	if err != io.EOF {
		ll.Log2Error(err)
	}
}

// OmsgClose ...
func (o *Client) OmsgClose() {
	ll.Log4Trace("connect closed:", o.serverPort, o.proxyPort)

	// 清理本地数据
	o.serves.Range(func(key, val interface{}) bool {
		key.(net.Conn).Close()
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
	case cmdTunnelSuccess: // 通道创建成功
		ll.Log4Trace("tunnel success:", o.serverPort, string(data))
	case cmdTunnelFailed: // 通道创建失败，断开连接
		ll.Log2Error("tunnel failed:", o.serverPort, string(data))
		o.msg.Close()
	case cmdBrowserClose: // 远端浏览器关闭
		if serve, ok := o.serves.Load(string(data)); ok {
			ll.Log0Debug("close local connect:", string(data))
			serve.(net.Conn).Close()
		}
	case cmdData: // 远端浏览器数据流
		o.browserDataStream(deData(data))
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
		o.Send(cmdTunnel, 0, []byte(o.proxyPort))
		break
	}
	ll.Log4Trace("connect success:", o.serverPort, o.proxyPort)

	return nil
}

// 处理浏览器数据流
func (o *Client) browserDataStream(browserAddr, serveAddr string, browserData []byte) {
	var (
		err   error
		serve net.Conn
	)

	defer func() {
		if err != nil {
			// 通知本地服务错误
			ll.Log0Debug("local serve error:", err)
			if err := o.Send(cmdLocaSrveClose, 0, enData(browserAddr, serveAddr, []byte(err.Error()))); err != nil {
				ll.Log2Error(err)
			}
		}
	}()

	// 此浏览器的请求是否已有本地服务连接
	if _conn, ok := o.serves.Load(browserAddr + serveAddr); ok {
		serve = _conn.(net.Conn)
	}

	// 本地还没有与服务建立连接，创建一个新的服务连接
	if serve == nil {
		// addr[0.0.0.0:2345:192.168.1.240:5000]
		tmp := strings.Split(serveAddr, ":")
		serve, err = net.DialTimeout("tcp", strings.Join(tmp[2:], ":"), time.Second*3)
		if err != nil {
			return
		}
		ll.Log0Debug("create local connect:", serve.LocalAddr())
		o.serves.Store(browserAddr+serveAddr, serve)

		go func() {
			defer serve.Close()
			// 转发服务的数据
			io.Copy(&pipeClient{
				serve:       serve,
				cli:         o,
				browserAddr: browserAddr,
				serveAddr:   serveAddr,
			}, serve)
			ll.Log0Debug("local end:", serve.LocalAddr())
			o.serves.Delete(browserAddr + serveAddr)
		}()
	}

	// 数据转发到本地服务
	if _, err := serve.Write(browserData); err != nil {
		// 发送失败，关闭本次通道
		serve.Close()
		return
	}
}

// 数据通道
type pipeClient struct {
	cli                    *Client
	serve                  net.Conn
	browserAddr, serveAddr string
}

func (o *pipeClient) Write(p []byte) (n int, err error) {
	// 数据转发到远端
	if err := o.cli.Send(cmdData, 0, enData(o.browserAddr, o.serveAddr, p)); err != nil {
		// 发送失败，关闭本次通道
		o.serve.Close()
		return 0, err
	}

	return len(p), nil
}
