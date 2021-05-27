package main

import (
	"github.com/PlatONnetwork/PlatON-Go/p2p"
	"net"
	"time"
)

const defaultDialTimeout = 10 * time.Second

func NewTCPDialer() p2p.TCPDialer {
	return p2p.TCPDialer{&net.Dialer{Timeout: defaultDialTimeout}}
}
