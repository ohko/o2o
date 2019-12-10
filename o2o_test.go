package o2o

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
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

	// server
	if err := s.Start(key, serverPort); err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)

	// client
	if err := c.Start(key, serverPort, proxyPort); err != nil {
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
