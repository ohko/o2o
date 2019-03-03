package o2o

import (
	"bytes"
	"encoding/hex"
	"log"
	"net"
	"testing"
	"time"
)

func serivces() {
	serverPort := ":2399"
	proxyPort := "2345:127.0.0.1:5000"

	if err := (&Server{}).Start(serverPort); err != nil {
		log.Fatal(err)
	}

	chSuccess := make(chan bool)
	success := func(msg string) {
		log.Println(msg)
		if msg == proxyPort {
			chSuccess <- true
		}
	}

	if err := (&Client{}).Start(serverPort, proxyPort, false, success); err != nil {
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
	case <-time.After(time.Second):
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

	conn, err := net.Dial("tcp", ":2345")
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("123456789")
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
}
