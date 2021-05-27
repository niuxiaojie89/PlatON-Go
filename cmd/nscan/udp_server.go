package main

import (
	"fmt"
	"github.com/PlatONnetwork/PlatON-Go/crypto"
	"github.com/PlatONnetwork/PlatON-Go/p2p/discover"
	"math/big"
	"net"
)

const ListenAddr = ":16789"

var ecdsaKey, _ = crypto.GenerateKey()

func NewUDPTable(chainID uint64) (*discover.Table, error) {
	addr, err := net.ResolveUDPAddr("udp", ListenAddr)
	if err != nil {
		fmt.Println(err.Error())
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println(err.Error())
	}
	realaddr := conn.LocalAddr().(*net.UDPAddr)
	cfg := discover.Config{
		PrivateKey:   ecdsaKey,
		ChainID:      new(big.Int).SetUint64(chainID),
		AnnounceAddr: realaddr,
		NodeDBPath:   "",
		NetRestrict:  nil,
		Bootnodes:    nil,
		Unhandled:    nil,
	}
	return discover.ListenUDP(conn, cfg)
}
