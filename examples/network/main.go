package main

import (
	"encoding/hex"
	"fmt"
	"github.com/geeeeorge/tcp-ip-go/network"
)

func main() {
	nw, _ := network.NewTun()
	nw.Bind()

	for {
		pkt, _ := nw.Read()
		fmt.Print(hex.Dump(pkt.Buf[:pkt.N]))
		_ = nw.Write(pkt)
	}
}
