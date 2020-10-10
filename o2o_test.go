package o2o

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testCount = 1

var (
	s = &Server{}
	c = &Client{}
)

func serivces() {
	serverPort := ":2399"
	proxyPort := "0.0.0.0:2345:127.0.0.1:5000"
	key := "12345678"
	crc := false

	// server
	if err := s.Start(key, serverPort, crc); err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)

	// client
	if err := c.Start(key, serverPort, proxyPort, crc); err != nil {
		log.Fatal(err)
	}

	// go func() {
	// 	// client
	// 	if err := (&Client{}).Start(key, serverPort, proxyPort); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }()

	// local server
	s, err := net.Listen("tcp", ":5000")
	if err != nil {
		log.Fatal(err)
	}
	go func(s net.Listener) {
		for {
			conn, err := s.Accept()
			if err != nil {
				log.Fatal(err)
				break
			}

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				log.Fatal(err)
			}

			if _, err := conn.Write(reverse(buf[:n])); err != nil {
				log.Fatal(err)
			}

			conn.Close()
		}
	}(s)

	time.Sleep(time.Second)
}

func reverse(data []byte) []byte {
	n := len(data)
	msg := make([]byte, len(data))
	for k, v := range data {
		msg[n-1-k] = v
	}
	return msg
}

// go test o2o -run TestServerClient -v -count=1
func TestServerClient(t *testing.T) {
	serivces()

	var wg sync.WaitGroup
	fn := func(msg []byte) {
		conn, err := net.Dial("tcp", ":2345")
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		if _, err := conn.Write(msg); err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		if bytes.Compare(msg, reverse(buf[:n])) != 0 {
			log.Println(hex.Dump(msg))
			log.Println(hex.Dump(reverse(buf[:n])))
			t.Fail()
		}
		wg.Done()
	}

	wg.Add(testCount)
	for i := 0; i < testCount; i++ {
		go fn([]byte(fmt.Sprintf("12345678%d", i)))
	}

	wg.Wait()

	time.Sleep(time.Second * 2)
	c.msg.Close()
	time.Sleep(time.Second * 3)
}

func TestAesCrypt(t *testing.T) {
	texts := [][]byte{
		[]byte(strings.Repeat(".", 3)),
		[]byte(strings.Repeat(".", 0x10)),
	}

	for _, v := range texts {
		en := aesCrypt(v)
		de := aesCrypt(en)

		if bytes.Compare(v, de) != 0 {
			t.Fail()
		}
	}
}

// go test o2o -run TestDisconnect -v -count=1
func TestDisconnect(t *testing.T) {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.Flags() | log.Lshortfile)

	// 服务器连接数
	sLink := int64(0)

	// 模拟本地服务器
	var localConn net.Conn
	local, err := net.Listen("tcp", ":5000")
	if err != nil {
		t.Fatal(err)
	}
	go func(local net.Listener) {
		for {
			conn, err := local.Accept()
			if err != nil {
				break
			}

			go func(conn net.Conn) {
				defer atomic.AddInt64(&sLink, -1)
				defer conn.Close()
				localConn = conn

				atomic.AddInt64(&sLink, 1)
				buf := make([]byte, 1024)
				for {
					n, err := conn.Read(buf)
					if err != nil {
						break
					}

					n, err = conn.Write(reverse(buf[:n]))
					if err != nil {
						break
					}
				}
			}(conn)
		}
	}(local)

	serverPort := ":2399"
	proxyPort := "0.0.0.0:2345:127.0.0.1:5000"
	key := "12345678"
	crc := false

	// server
	if err := s.Start(key, serverPort, crc); err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)

	// client
	if err := c.Start(key, serverPort, proxyPort, crc); err != nil {
		log.Fatal(err)
	}

	if sLink != 0 && c.localServersCount != 0 {
		t.Fatal("sLink:", sLink, "cLink:", c.localServersCount)
	}

	// 测试
	msg := make([]byte, 32)
	nmsg, err := rand.Read(msg)
	if err != nil {
		t.Fatal(err)
	}
	if nmsg != 32 {
		t.Fatal("nmsg!=32", nmsg)
	}
	conn, err := net.Dial("tcp", ":2345")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := conn.Write(msg); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(msg, reverse(buf[:n])) != 0 {
		log.Println(hex.Dump(msg))
		log.Println(hex.Dump(reverse(buf[:n])))
		t.Fail()
	}

	if sLink != 1 && c.localServersCount != 1 {
		t.Fatal("sLink:", sLink, "cLink:", c.localServersCount)
	}

	time.Sleep(time.Second)

	// User断开情况
	// log.Println(conn.Close())

	// LocalServer断开情况
	localConn.Close()
	local.Close()

	// Server断开情况
	// s.msg.Close()

	// Client断开情况
	// c.msg.Close()

	time.Sleep(time.Second)
	if sLink != 0 && c.localServersCount != 0 {
		t.Fatal("sLink:", sLink, "cLink:", c.localServersCount)
	}
}
