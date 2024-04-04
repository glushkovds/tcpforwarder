package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type Forwarder struct {
	src            Address
	dst            Address
	dialTimeout    int
	acceptIPFilter string
}

func NewForwarder(src Address, dst Address, dialTimeout int, acceptIPFilter string) *Forwarder {
	return &Forwarder{
		src:            src,
		dst:            dst,
		dialTimeout:    dialTimeout,
		acceptIPFilter: acceptIPFilter,
	}
}

func (f *Forwarder) Start() {
	listener, err := net.Listen("tcp", f.src.String())
	panicIfErr(err)
	_, acceptSubnet, _ := net.ParseCIDR(f.acceptIPFilter)
	for {
		srcConn, err := listener.Accept()
		if err != nil {
			println(err.Error())
			continue
		}
		srcIP, _ := net.ResolveTCPAddr("tcp", srcConn.RemoteAddr().String())
		if !acceptSubnet.Contains(srcIP.IP) {
			println(fmt.Sprintf(`%s -- %s restricted`, time.Now().Format(time.RFC3339), srcIP))
			_ = srcConn.Close()
			continue
		}
		go func(srcConn net.Conn) {
			dstConn, err := net.DialTimeout("tcp", f.dst.String(), time.Duration(f.dialTimeout)*time.Second)
			if err != nil {
				if strings.Contains(err.Error(), "i/o timeout") {
					println(fmt.Sprintf(`%s -- %s >-< %s %s`, time.Now().Format(time.RFC3339), srcConn.RemoteAddr().String(), f.dst.String(), "dial timed out"))
				} else {
					println(err.Error())
				}
				_ = srcConn.Close()
				return
			}
			println(fmt.Sprintf(`%s -- %s <-> %s`, time.Now().Format(time.RFC3339), srcConn.RemoteAddr().String(), f.dst.String()))
			go func(srcConn, dstConn net.Conn) {
				_, _ = io.Copy(srcConn, dstConn)
				_ = srcConn.Close()
				_ = dstConn.Close()
			}(srcConn, dstConn)
			go func(srcConn, dstConn net.Conn) {
				_, _ = io.Copy(dstConn, srcConn)
				_ = srcConn.Close()
				_ = dstConn.Close()
			}(srcConn, dstConn)
		}(srcConn)
	}
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
