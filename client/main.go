package main

import (
	"flag"
	"log"
	"o2o"
)

var (
	proxyPort  = flag.String("p", "2345:192.168.1.238:5000", "请求创建隧道：internetPort:intranet:port")
	serverPort = flag.String("s", ":2399", "外网服务器")
)

func main() {
	flag.Parse()

	o := &o2o.Client{}
	if err := o.Start(*serverPort, *proxyPort, true, nil); err != nil {
		log.Fatal(err)
	}

	o2o.WaitCtrlC()
}
