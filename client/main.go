package main

import (
	"flag"
	"io"
	"log"
	"net"
	"o2o"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ohko/omsg"
)

var (
	client     *omsg.Client
	bReconnect bool
	proxyPort  = flag.String("p", "2345:192.168.1.238:5000", "请求创建隧道：internetPort:intranet:port")
	serverPort = flag.String("s", ":2399", "外网服务器")
	tunnelPort = flag.String("t", ":2398", "隧道服务器")
)

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	client = omsg.NewClient(false)
	client.OnData = onData
	client.OnClose = onClose
	bReconnect = true
	reconnect()

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
}

func reconnect() {
	for {
		if !bReconnect {
			return
		}
		err := client.ConnectTimeout(*serverPort, time.Second*5)
		if err == nil {
			if err := client.SendAsync([]byte(`proxy|` + *proxyPort)); err != nil {
				log.Println(err)
			}
			break
		}
		log.Println(err)
		time.Sleep(time.Second * 5)
	}
	log.Println("connect server success")
}

func onData(data []byte) {
	tmp := strings.Split(string(data), "|")
	cmd, ext := tmp[0], tmp[1]

	if cmd == "heart" {
		log.Println("heart:", ext)
		if err := client.SendAsync([]byte(`close|close`)); err != nil {
			log.Println(err)
		}
	} else if cmd == "listen" {
		log.Println("tunnel create success:", ext)
	} else if cmd == "error" {
		log.Println("server error:", ext)
	} else if cmd == "tunnel" {
		if err := client.SendAsync([]byte(`close|close`)); err != nil {
			log.Println(err)
		}
		xx := strings.Split(*proxyPort, ":")
		loc, err := net.Dial("tcp", xx[1]+":"+xx[2])
		if err != nil {
			log.Println(err)
			return
		}

		cs, err := net.Dial("tcp", *tunnelPort)
		if err != nil {
			log.Println(err)
			return
		}
		go o2o.Send(cs, []byte(ext))

		go func() {
			go io.Copy(loc, cs)
			go io.Copy(cs, loc)
		}()
	}
}

func onClose() {
	log.Println("reconnect...")
	reconnect()
}
