package main

import (
	"flag"
	"log"
	"o2o"
)

var (
	serverPort = flag.String("s", ":2399", "外网服务器服务端口，用于接收内网服务器连接")
)

func main() {
	flag.Parse()

	o := &o2o.Server{}
	if err := o.Start(*serverPort); err != nil {
		log.Fatal(err)
	}

	o2o.WaitCtrlC()
}
