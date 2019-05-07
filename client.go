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

// Client 客户端
type Client struct {
	ocli                  *omsg.Client
	serverPort, proxyPort string
	reconnect             bool
	onSuccess             func(msg string)
	conns                 sync.Map
}

// Start 启动客户端
func (o *Client) Start(key, serverPort, proxyPort string, reconnect bool, onSuccess func(msg string)) error {
	o.serverPort, o.proxyPort = serverPort, proxyPort
	o.reconnect = reconnect
	o.onSuccess = onSuccess

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

	o.ocli = omsg.NewClient()
	o.ocli.OnData = o.onData
	o.ocli.OnClose = o.onClose
	return o.Reconnect()
}
func (o *Client) onClose() {
	log.Println("connect closed")
	if o.reconnect {
		o.Reconnect()
	}
}
func (o *Client) onData(cmd, ext uint16, data []byte) {
	var err error
	data = aesEncode(data)

	switch cmd {
	case CMDMSG:
		log.Println(string(data))
	case CMDSUCCESS:
		log.Println("Success:", string(data))
		if o.onSuccess != nil {
			o.onSuccess(string(data))
		}
	case CMDCLOSE:
		if conn, ok := o.conns.Load(string(data)); ok {
			// log.Println("关闭本地连接:", string(data))
			conn.(net.Conn).Close()
		}
	case CMDDATA:
		client, addr, data := deData(data)
		conn, ok := o.conns.Load(client + addr)
		if !ok {
			tmp := strings.Split(addr, ":")
			conn, err = net.Dial("tcp", strings.Join(tmp[1:], ":"))
			if err != nil {
				log.Println(err)
				return
			}
			o.conns.Store(client+addr, conn)
			go func(conn net.Conn) {
				buf := make([]byte, bufferSize)
				for {
					n, err := conn.Read(buf)
					if err != nil {
						return
					}
					// log.Println("本地转发：\n" + hex.Dump(buf[:n]))
					if err := o.ocli.Send(CMDDATA, 0, enData(client, addr, buf[:n])); err != nil {
						log.Println(err)
					}
				}
			}(conn.(net.Conn))
		}
		// log.Println("向本地发送：\n" + hex.Dump(data))
		conn.(net.Conn).Write(data)
	}
}

// Reconnect 重新连接服务器
func (o *Client) Reconnect() error {
	log.Println("connect:", o.serverPort)
	for {
		if err := o.ocli.Connect(o.serverPort); err != nil {
			if o.reconnect {
				time.Sleep(time.Second * 5)
				continue
			}
			return err
		}
		o.ocli.Send(CMDTUNNEL, 0, aesEncode([]byte(o.proxyPort)))
		break
	}
	log.Println("connect success:", o.serverPort)
	return nil
}
