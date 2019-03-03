package o2o

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func TestAesEncode(t *testing.T) {
	texts := [][]byte{
		[]byte(strings.Repeat(".", 3)),
		[]byte(strings.Repeat(".", 0x10)),
	}

	for _, v := range texts {
		en, _ := aesEncode(v)
		fmt.Println(hex.Dump(en))
		de, _ := aesEncode(en)
		fmt.Println(hex.Dump(de))

		if bytes.Compare(v, de) != 0 {
			t.Fail()
		}
	}
}
