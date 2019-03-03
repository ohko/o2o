package o2o

import (
	"crypto/md5"
	"crypto/sha256"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

// Client 客户端
type Client struct {
	client                net.Conn
	serverPort, proxyPort string
	reconnect             bool
	onSuccess             func(msg string)
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

	return o.Reconnect()
}

// Reconnect 重新连接服务器
func (o *Client) Reconnect() error {
	for {
		conn, err := net.DialTimeout("tcp", o.serverPort, time.Second*5)
		if err != nil {
			if o.reconnect {
				time.Sleep(time.Second * 5)
				continue
			}
			return err
		}
		o.client = conn
		break
	}
	log.Println("connect success:", o.serverPort)

	go func() {
		o.listen()
		log.Println("connect closed")
		if o.reconnect {
			o.Reconnect()
		}
	}()
	return nil
}

func (o *Client) listen() {
	if err := send(o.client, CMDCLIENT, o.proxyPort); err != nil {
		log.Println(err)
		o.client.Close()
		return
	}

	for {
		data, err := recv(o.client)
		if err != nil {
			log.Println(err)
			return
		}

		go func(cmd, ext string) {
			switch cmd {
			case CMDMSG:
				log.Println("msg:", ext)
			case CMDSUCCESS:
				log.Println("success:", ext)
				if o.onSuccess != nil {
					o.onSuccess(ext)
				}
			case CMDREQUEST:
				tmp := strings.Split(o.proxyPort, ":")
				loc, err := net.Dial("tcp", strings.Join(tmp[1:], ":"))
				if err != nil {
					log.Println(err)
					return
				}

				cs, err := net.Dial("tcp", o.serverPort)
				if err != nil {
					loc.Close()
					log.Println(err)
					return
				}

				if err := send(cs, CMDTENNEL, ext); err != nil {
					loc.Close()
					cs.Close()
					return
				}
				if aesEnable {
					rw1 := &rw{conn: loc, origin: loc}
					rw2 := &rw{conn: cs, origin: loc}
					go io.Copy(rw1, rw2)
					go io.Copy(rw2, rw1)
				} else {
					go io.Copy(loc, cs)
					go io.Copy(cs, loc)
				}
			}
		}(string(data[0]), string(data[1:]))
	}
}
