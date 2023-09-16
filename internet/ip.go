package internet

import (
	"context"
	"fmt"
	"github.com/geeeeorge/tcp-ip-go/network"
	"log"
)

const (
	QUEUE_SIZE = 10
)

type IpPacket struct {
	IpHeader *Header
	Packet   network.Packet
}

type IpPacketQueue struct {
	incomingQueue chan IpPacket
	outgoingQueue chan network.Packet
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewIpPacketQueue() *IpPacketQueue {
	return &IpPacketQueue{
		incomingQueue: make(chan IpPacket, QUEUE_SIZE),
		outgoingQueue: make(chan network.Packet, QUEUE_SIZE),
	}
}

func (q *IpPacketQueue) ManageQueues(network *network.TUN) {
	q.ctx, q.cancel = context.WithCancel(context.Background())

	// 受信用ゴルーチン
	go func() {
		for {
			select {
			case <-q.ctx.Done():
				return
			default:
				pkt, err := network.Read()
				if err != nil {
					log.Printf("network.Read: %s", err.Error())
				}
				ipHeader, err := unmarshal(pkt.Buf[:pkt.N])
				if err != nil {
					log.Printf("unmarshal: %s", err)
					continue
				}
				ipPacket := IpPacket{
					IpHeader: ipHeader,
					Packet:   pkt,
				}
				q.incomingQueue <- ipPacket
			}
		}
	}()

	go func() {
		for {
			select {
			case <-q.ctx.Done():
				return
			case pkt := <-q.outgoingQueue:
				err := network.Write(pkt)
				if err != nil {
					log.Printf("network.Write: %s", err.Error())
				}
			}
		}
	}()
}

func (q *IpPacketQueue) Close() {
	q.cancel()
}
func (q *IpPacketQueue) Read() (IpPacket, error) {
	pkt, ok := <-q.incomingQueue
	if !ok {
		return IpPacket{}, fmt.Errorf("incoming queue is closed")
	}
	return pkt, nil
}

func (q *IpPacketQueue) Write(pkt network.Packet) error {
	select {
	case q.outgoingQueue <- pkt:
		return nil
	case <-q.ctx.Done():
		return fmt.Errorf("network closed")
	}
}
