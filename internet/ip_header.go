package internet

import (
	"encoding/binary"
	"fmt"
)

const (
	IPv4              = 4
	IHL               = 5
	TOS               = 0
	TTL               = 64
	LENGTH            = IHL * 4
	TCP_PROTCOL       = 6
	IP_HEADER_MIN_LEN = 20
)

// Header
// RFC791
type Header struct {
	Version            uint8
	IHL                uint8
	TypeOfService      uint8
	TotalLength        uint16
	Identification     uint16
	Flags              uint8
	FragmentOffset     uint16
	TimeToLive         uint8
	Protocol           uint8
	HeaderChecksum     uint16
	SourceAddress      [4]byte
	DestinationAddress [4]byte
}

func unmarshal(pkt []byte) (*Header, error) {
	if len(pkt) < IP_HEADER_MIN_LEN {
		return nil, fmt.Errorf("invalid IP header length")
	}

	return &Header{
		Version:            pkt[0] >> 4,
		IHL:                pkt[0] & 0x0F,
		TypeOfService:      pkt[1],
		TotalLength:        binary.BigEndian.Uint16(pkt[2:4]),
		Identification:     binary.BigEndian.Uint16(pkt[4:6]),
		Flags:              pkt[6] >> 5,
		FragmentOffset:     binary.BigEndian.Uint16(pkt[6:8]) & 0x1FFF,
		TimeToLive:         pkt[8],
		Protocol:           pkt[9],
		HeaderChecksum:     binary.BigEndian.Uint16(pkt[10:12]),
		SourceAddress:      [4]byte(pkt[12:16]),
		DestinationAddress: [4]byte(pkt[16:20]),
	}, nil
}

func (h *Header) Marshal() []byte {
	pkt := make([]byte, 20)
	pkt[0] = (h.Version << 4) | h.IHL
	pkt[1] = h.TypeOfService
	binary.BigEndian.PutUint16(pkt[2:4], h.TotalLength)
	binary.BigEndian.PutUint16(pkt[4:6], h.Identification)
	binary.BigEndian.PutUint16(pkt[6:8], (uint16(h.Flags<<13))|(h.FragmentOffset&0x1FFF))
	pkt[8] = h.TimeToLive
	pkt[9] = h.Protocol
	binary.BigEndian.PutUint16(pkt[10:12], 0) // チェックサムフィールドを0にセット
	copy(pkt[12:16], h.SourceAddress[:])
	copy(pkt[16:20], h.DestinationAddress[:])

	h.setChecksum(pkt)
	binary.BigEndian.PutUint16(pkt[10:12], h.HeaderChecksum)

	return pkt
}

func (h *Header) setChecksum(pkt []byte) {
	length := len(pkt)
	var checksum uint32

	for i := 0; i < length-1; i += 2 {
		checksum += binary.BigEndian.Uint32(pkt[i : i+2])
	}
	if length%2 != 0 {
		checksum += uint32(pkt[length-1])
	}

	for checksum > 0xFFFF {
		checksum = (checksum & 0xFFFF) + (checksum >> 16) // 下位16bit + 上位16bit
	}

	h.HeaderChecksum = ^uint16(checksum)
}
