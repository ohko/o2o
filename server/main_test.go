package main

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func Test_web(t *testing.T) {
	x := make([]byte, 0x10)
	n := "123|" + string("\r\n|") + string(x) + "|456"
	fmt.Println(hex.Dump([]byte(n)))
	fmt.Println("===")
	xx := strings.Split(string(n), "|")
	fmt.Println(len(xx))
	fmt.Println(hex.Dump([]byte(xx[2])))
}
