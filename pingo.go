package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

const (
	ICMP_ECHO_REQUEST = 8 // Echo request code
	ICMP_ECHO_REPLY   = 0 // Echo reply code
)

func main() {
	// Parse the address
	flag.Parse()
	straddr := flag.Arg(0)

	// Make the remote connection
	raddr, err := net.ResolveIPAddr("ip4:icmp", straddr)
	checkErr(err)

	conn, err := net.DialIP("ip4:icmp", nil, raddr)
	checkErr(err)

	// Data to use in the packet header
	id := os.Getpid() & 0xffff
	seq := 0
	pktlen := 64

	// Connection has been made, we can announce
	fmt.Printf("PING request to %s (%s) - %v data bytes\n",
		straddr, raddr, pktlen-8)

	// Track success, failure, and a list of the times
	total := 0
	success := 0
	fail := 0
	times := make([]float32, 0)

	// Capture the exit signal and print statistics for the session
	// when we receive SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			loss := float32(fail) / float32(total)
			fmt.Printf("\n--- %s ping statistics ---\n", straddr)
			fmt.Printf("%v packets sent, %v packets received - %0.2f%% packet loss\n",
				total, success, loss)
			os.Exit(0)
		}
	}()

	for {
		seq++
		p := makePacket(id, seq, pktlen, []byte("Hello, world!"))
		checkErr(err)

		n, err := conn.Write(p)
		sent := time.Now()
		if err != nil || n != len(p) {
			fmt.Printf("Packet send failure seq= %v\n", seq)
			continue
		}
		total++

		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

		resp := make([]byte, 1024)
		for {
			rlen, _, err := conn.ReadFrom(resp)
			if err != nil {
				printFailure(seq)
				fail++
				break
			}

			if resp[0] == ICMP_ECHO_REPLY {
				since := float32(time.Since(sent).Nanoseconds()) / 1000000
				success++
				times = append(times, since)
				printSuccess(rlen, seq, conn, since)
				break
			}
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

	// Add the payload
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

	// Calculate and add the checksum
	c := checksum(p)
	p[2] ^= uint8(c & 0xff)
	p[3] ^= uint8(c >> 8)

	return p
}

func checksum(p []byte) uint16 {
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

	return uint16(^s)
}

func printSuccess(rlen, seq int, conn *net.IPConn, since float32) {
	fmt.Printf("%v bytes from %s: seq=%v time=%0.3f ms\n",
		rlen, conn.RemoteAddr(), seq, since)
}

func printFailure(seq int) {
	fmt.Printf("icmp timeout seq=%v\n", seq)
}
