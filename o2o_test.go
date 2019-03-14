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

func serivces() {
	serverPort := ":2399"
	proxyPort := "2345:127.0.0.1:5000"
	key := "12345678"

	if err := (&Server{}).Start(key, serverPort); err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Microsecond * 100)

	chSuccess := make(chan bool)
	success := func(msg string) {
		log.Println("SUCCESS:", msg)
		if msg == proxyPort {
			chSuccess <- true
		}
	}

	if err := (&Client{}).Start(key, serverPort, proxyPort, false, success); err != nil {
		log.Fatal(err)
	}

	s, err := net.Listen("tcp", ":5000")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
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
	}()

	select {
	case <-chSuccess:
	case <-time.After(time.Second * 3):
		log.Fatal("timeout")
	}
}

func reverse(data []byte) []byte {
	n := len(data)
	msg := make([]byte, len(data))
	for k, v := range data {
		msg[n-1-k] = v
	}
	return msg
}

func TestServerClient(t *testing.T) {
	serivces()

	var wg sync.WaitGroup
	fn := func(msg []byte) {
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
		wg.Done()
	}

	count := 100
	wg.Add(count)
	for i := 0; i < count; i++ {
		go fn([]byte(fmt.Sprintf("12345678%d", i)))
	}

	wg.Wait()
}

func TestAesEncode(t *testing.T) {
	texts := [][]byte{
		[]byte(strings.Repeat(".", 3)),
		[]byte(strings.Repeat(".", 0x10)),
	}

	for _, v := range texts {
		en := aesEncode(v)
		fmt.Println(hex.Dump(en))
		de := aesEncode(en)
		fmt.Println(hex.Dump(de))

		if bytes.Compare(v, de) != 0 {
			t.Fail()
		}
	}
}
