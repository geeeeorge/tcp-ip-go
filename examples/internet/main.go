package main

import (
	"fmt"
	"github.com/geeeeorge/tcp-ip-go/internet"
	"github.com/geeeeorge/tcp-ip-go/network"
)

func main() {
	nw, _ := network.NewTun()
	nw.Bind()
	ip := internet.NewIpPacketQueue()
	ip.ManageQueues(nw)

	for {
		pkt, _ := ip.Read()
		fmt.Printf("IP Header: %+v\n", pkt.IpHeader)
	}
}
