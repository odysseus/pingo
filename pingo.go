package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

const (
	ICMP_ECHO_REQUEST = 8 // Echo request code
	ICMP_ECHO_REPLY   = 0 // Echo reply code
)

func main() {
	flag.Parse()
	straddr := flag.Arg(0)

	raddr, err := net.ResolveIPAddr("ip4:icmp", straddr)
	checkErr(err)

	conn, err := net.DialIP("ip4:icmp", nil, raddr)
	checkErr(err)

	id := os.Getpid() & 0xffff
	seq := 0
	pktlen := 32

	for {
		seq++
		p := makePacket(id, seq, pktlen, []byte("Hello, world!"))
		checkErr(err)

		n, err := conn.Write(p)
		if err != nil || n != len(p) {
			log.Fatal("Packet send failed")
		}

		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

		resp := make([]byte, 1024)
		for {
			rlen, _, err := conn.ReadFrom(resp)
			checkErr(err)

			fmt.Println(resp[:rlen])
			break
		}
		time.Sleep(1e9)
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func makePacket(id, seq, pktlen int, filler []byte) []byte {
	p := make([]byte, pktlen)
	copy(p[8:], bytes.Repeat(filler, (pktlen-8)/len(filler)+1))

	// Set up the packet
	p[0] = ICMP_ECHO_REQUEST // Type
	p[1] = 0                 // Code
	p[2] = 0                 // Header checksum - set later
	p[3] = 0                 // Header checksum - set later
	p[4] = uint8(id >> 8)    // Identifier
	p[5] = uint8(id & 0xff)  // Identifier
	p[6] = uint8(seq >> 8)   // Sequence number
	p[7] = uint8(seq & 0xff) // Sequence number

	// calculate icmp checksum
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

	// place checksum back in header; using ^= avoids the
	// assumption the checksum bytes are zero
	p[2] ^= uint8(^s & 0xff)
	p[3] ^= uint8(^s >> 8)

	return p
}
