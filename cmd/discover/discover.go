package main

import (
	"flag"
	"fmt"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/common/hexutil"
	"github.com/PlatONnetwork/PlatON-Go/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"net"
	"time"
)

const NodeIDBits = 512

var (
	nodeDBItemPrefix = []byte("n:") // Identifier to prefix node entries with
)

type NodeID [NodeIDBits / 8]byte

// Node represents a host on the network.
// The fields of Node may not be modified.
type Node struct {
	IP       net.IP // len 4 for IPv4 or 16 for IPv6
	UDP, TCP uint16 // port numbers
	ID       NodeID // the node's public key

	// This is a cached copy of sha3(ID) which is used for node
	// distance calculations. This is part of Node in order to make it
	// possible to write tests that need a node at a certain distance.
	// In those tests, the content of sha will not actually correspond
	// with ID.
	sha common.Hash

	// Time when the node was added to the table.
	addedAt time.Time
}

func main() {
	// node数据库目录
	nodeData := flag.String("nodeData", "D:/data/alaya/nodes", "node data directory")
	flag.Parse()

	opts := &opt.Options{OpenFilesCacheCapacity: 5}
	db, err := leveldb.OpenFile(*nodeData, opts)
	if err != nil {
		fmt.Println("Open leveldb error", "nodeData", nodeData, "error", err.Error())
		return
	}
	// 遍历查找节点
	it := db.NewIterator(util.BytesPrefix(nodeDBItemPrefix), nil)
	for it.Next() {
		value := it.Value()
		var n Node
		if err := rlp.DecodeBytes(value, &n); err == nil {
			fmt.Println("find node", "nodeId", hexutil.Encode(n.ID[:]), "nodeIP", n.IP)
		} else {
			//fmt.Println("find node error", "error", err.Error())
		}
	}
	fmt.Println("find node complete")
	it.Release()
}
