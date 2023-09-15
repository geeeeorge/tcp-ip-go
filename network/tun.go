package network

import (
	"context"
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
)

type interfaceRequest struct {
	Name  [16]byte
	Flags int16
}

const (
	TUNSETIFF   = 0x400454ca
	IFF_TUN     = 0x0001
	IFF_NO_PI   = 0x1000
	PACKET_SIZE = 2048
	QUEUE_SIZE  = 10
)

type Packet struct {
	Buf []byte
	N   uintptr
}

// TUN
// ネットワークデバイスを仮想的にシミュレートできる
// TUNはL3すなわちIP層で動作する仮想デバイス
type TUN struct {
	file          *os.File
	incomingQueue chan Packet
	outgoingQueue chan Packet
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewTun() (*TUN, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("os.OpenFile: %s", err.Error())
	}

	ir := interfaceRequest{}
	copy(ir.Name[:], []byte("tun0"))
	ir.Flags = IFF_TUN | IFF_NO_PI

	_, _, sysErr := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ir)))
	if sysErr != 0 {
		return nil, fmt.Errorf("syscall.SYS_IOCTL: %s", sysErr.Error())
	}

	return &TUN{
		file:          file,
		incomingQueue: make(chan Packet, QUEUE_SIZE),
		outgoingQueue: make(chan Packet, QUEUE_SIZE),
	}, nil
}

func (t *TUN) Close() error {
	if err := t.file.Close(); err != nil {
		return fmt.Errorf("file.Close: %s", err.Error())
	}
	t.cancel()

	return nil
}

func (t *TUN) read(buf []byte) (uintptr, error) {
	n, _, sysErr := syscall.Syscall(syscall.SYS_READ, t.file.Fd(), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if sysErr != 0 {
		return 0, fmt.Errorf("sysccall.SYS_READ: %s", sysErr.Error())
	}
	return n, nil
}

func (t *TUN) write(buf []byte) (uintptr, error) {
	n, _, sysErr := syscall.Syscall(syscall.SYS_WRITE, t.file.Fd(), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if sysErr != 0 {
		return 0, fmt.Errorf("sysccall.SYS_WRITE: %s", sysErr.Error())
	}
	return n, nil
}

// Bind
// パケットの送受信を行うゴルーチンを起動
func (t *TUN) Bind() {
	t.ctx, t.cancel = context.WithCancel(context.Background())

	// 受信用ゴルーチン
	go func() {
		for {
			select {
			case <-t.ctx.Done():
				return
			default:
				buf := make([]byte, PACKET_SIZE)
				n, err := t.read(buf)
				if err != nil {
					log.Printf("TUN.read: %s", err.Error())
				}
				packet := Packet{
					Buf: buf[:n],
					N:   n,
				}
				t.incomingQueue <- packet
			}
		}
	}()

	// 送信用ゴルーチン
	go func() {
		for {
			select {
			case <-t.ctx.Done():
				return
			case pkt := <-t.outgoingQueue:
				_, err := t.write(pkt.Buf[:pkt.N])
				if err != nil {
					log.Printf("TUN.write: %s", err.Error())
				}
			}
		}
	}()
}

func (t *TUN) Read() (Packet, error) {
	pkt, ok := <-t.incomingQueue
	if !ok {
		return Packet{}, fmt.Errorf("incoming queue is closed")
	}
	return pkt, nil
}

func (t *TUN) Write(pkt Packet) error {
	select {
	case t.outgoingQueue <- pkt:
		return nil
	case <-t.ctx.Done():
		return fmt.Errorf("device closed")
	}
}
