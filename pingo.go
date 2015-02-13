package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

const (
	ICMP_TYPE = 8 // 8 = Echo request
	ICMP_CODE = 0 // 0 = Echo code
)

func main() {
	flag.Parse()
	straddr := flag.Arg(0)

	raddr, err := net.ResolveIPAddr("ip4:icmp", straddr)
	checkErr(err)

	conn, err := net.DialIP("ip4:icmp", nil, raddr)
	checkErr(err)

	pid := uint16(os.Getpid())
	var seq uint16 = 1

	p, err := makePacket(pid, seq, []byte("Hello, world!"))
	checkErr(err)

	fmt.Println(conn)
	fmt.Println(p)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func makePacket(id, seq uint16, payload []byte) ([]byte, error) {
	// Find the payload length and ensure it does not exceed the max
	paylen := len(payload)
	if paylen > 256 {
		return nil, errors.New("Payload cannot exceed 256 bytes")
	}

	// Set up the packet
	p := make([]byte, 8+paylen)
	p[0] = ICMP_TYPE         // Type
	p[1] = ICMP_CODE         // Code
	p[2] = 0                 // Header checksum - set later
	p[3] = 0                 // Header checksum - set later
	p[4] = uint8(id >> 8)    // Identifier
	p[5] = uint8(id & 0xff)  // Identifier
	p[6] = uint8(seq >> 8)   // Sequence number
	p[7] = uint8(seq & 0xff) // Sequence number

	// Calculate and add the checksum
	c := checksum(p)
	p[2] = uint8(c >> 8)
	p[3] = uint8(c & 0xff)

	// Add the payload
	for i := 0; i < paylen; i++ {
		p[8+i] = payload[i]
	}

	return p, nil
}

func checksum(p []byte) uint16 {
	// http://www.faqs.org/rfcs/rfc1071.html
	cklen := len(p)
	s := uint32(0)

	for i := 0; i < (cklen - 1); i += 2 {
		s += uint32(p[i+1])<<8 | uint32(p[i])
	}

	if cklen&1 == 1 {
		s += uint32(p[cklen-1])
	}

	s = (s >> 16) + (s & 0xffff)
	s = s + (s >> 16)

	return uint16(s)
}
