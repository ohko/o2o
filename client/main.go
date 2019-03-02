package main

import (
	"flag"
	"io"
	"log"
	"net"
	"o2o"
	"strings"
	"time"
)

var (
	proxyPort  = flag.String("p", "2345:192.168.1.238:5000", "请求创建隧道：internetPort:intranet:port")
	serverPort = flag.String("s", ":2399", "外网服务器")
)

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	startClient()
	o2o.CatchCtrlC()
}

func startClient() {
	for {
		client, err := net.DialTimeout("tcp", *serverPort, time.Second*3)
		if err != nil {
			time.Sleep(time.Second * 5)
			continue
		}
		log.Println("connect success:", *serverPort)

		listen(client)
		log.Println("connect closed")
	}
}

func listen(conn net.Conn) {
	if err := o2o.Send(conn, o2o.CMDCLIENT, *proxyPort); err != nil {
		log.Println(err)
		conn.Close()
		return
	}

	for {
		data, err := o2o.Recv(conn)
		if err != nil {
			return
		}

		cmd, ext := string(data[0]), string(data[1:])
		switch string(cmd) {
		case o2o.CMDMSG:
			log.Println("msg:", ext)
		case o2o.CMDREQUEST:
			tmp := strings.Split(*proxyPort, ":")
			loc, err := net.Dial("tcp", strings.Join(tmp[1:], ":"))
			if err != nil {
				log.Println(err)
				return
			}

			cs, err := net.Dial("tcp", *serverPort)
			if err != nil {
				log.Println(err)
				return
			}

			o2o.Send(cs, o2o.CMDTENNEL, ext)
			go io.Copy(loc, cs)
			go io.Copy(cs, loc)
		}
	}
}
