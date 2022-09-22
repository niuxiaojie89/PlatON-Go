// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package trie

import (
	"fmt"
	"sync"

	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/crypto"
	"github.com/PlatONnetwork/PlatON-Go/rlp"
	"golang.org/x/crypto/sha3"
)

// leafChanSize is the size of the leafCh. It's a pretty arbitrary number, to allow
// some parallelism but not incur too much memory overhead.
const leafChanSize = 200

// leaf represents a trie leaf value
type leaf struct {
	size int         // size of the rlp data (estimate)
	hash common.Hash // hash of rlp data
	node node        // the node to commit
}

// committer is a type used for the trie Commit operation. A committer has some
// internal preallocated temp space, and also a callback that is invoked when
// leaves are committed. The leafs are passed through the `leafCh`,  to allow
// some level of parallelism.
// By 'some level' of parallelism, it's still the case that all leaves will be
// processed sequentially - onleaf will never be called in parallel or out of order.
type committer struct {
	sha crypto.KeccakState
	tmp sliceBuffer

	onleaf LeafCallback
	leafCh chan *leaf
}

// committers live in a global sync.Pool
var committerPool = sync.Pool{
	New: func() interface{} {
		return &committer{
			tmp: make([]byte, 0, 550), // cap is as large as a full fullNode.
			sha: sha3.NewLegacyKeccak256().(crypto.KeccakState),
		}
	},
}

// newCommitter creates a new committer or picks one from the pool.
func newCommitter() *committer {
	return committerPool.Get().(*committer)
}

func returnCommitterToPool(h *committer) {
	h.onleaf = nil
	h.leafCh = nil
	committerPool.Put(h)
}

// Commit collapses a node down into a hash node and inserts it into the database
func (c *committer) Commit(n node) (node, node, error) {
	// If we're not storing the node, just hashing, use available cached data
	if hash, dirty := n.cache(); len(hash) != 0 {
		if !dirty {
			switch n.(type) {
			case *fullNode, *shortNode:
				return hash, hash, nil
			default:
				return hash, n, nil
			}
		}
	}

	return c.commit(n, true)
}

// commit collapses a node down into a hash node and inserts it into the database
func (c *committer) commit(n node, force bool) (node, node, error) {
	// If we're not storing the node, just hashing, use available cached data
	if hash, dirty := n.cache(); len(hash) != 0 {
		if !dirty {
			switch n.(type) {
			case *fullNode, *shortNode:
				return hash, hash, nil
			default:
				return hash, n, nil
			}
		}
	}
	// Trie not processed yet or needs storage, walk the children
	collapsed, cached, err := c.commitChildren(n)
	if err != nil {
		return hashNode{}, n, err
	}
	hashed, err := c.store(collapsed, force)
	if err != nil {
		return hashNode{}, n, err
	}
	// Cache the hash of the node for later reuse and remove
	// the dirty flag in commit mode. It's fine to assign these values directly
	// without copying the node first because hashChildren copies it.
	cachedHash, _ := hashed.(hashNode)

	switch cn := cached.(type) {
	case *shortNode:
		*cn.flags.hash = cachedHash
		*cn.flags.dirty = false
	case *fullNode:
		*cn.flags.hash = cachedHash
		*cn.flags.dirty = false
	}
	return hashed, cached, nil
}

func (c *committer) commitChildren(original node) (node, node, error) {
	var err error

	switch n := original.(type) {
	case *shortNode:
		// Hash the short node's child, caching the newly hashed subtree
		collapsed, cached := n.copy(), n.copy()
		collapsed.Key = hexToCompact(n.Key)
		cached.Key = common.CopyBytes(n.Key)

		if _, ok := n.Val.(valueNode); !ok {
			collapsed.Val, cached.Val, err = c.commit(n.Val, false)
			if err != nil {
				return original, original, err
			}
		}
		return collapsed, cached, nil

	case *fullNode:
		// Hash the full node's children, caching the newly hashed subtrees
		collapsed, cached := n.copy(), n.copy()

		for i := 0; i < 16; i++ {
			if n.Children[i] != nil {
				collapsed.Children[i], cached.Children[i], err = c.commit(n.Children[i], false)
				if err != nil {
					return original, original, err
				}
			}
		}
		cached.Children[16] = n.Children[16]
		return collapsed, cached, nil

	default:
		// Value and hash nodes don't have children so they're left as were
		return n, original, nil
	}
}

// store hashes the node n and if we have a storage layer specified, it writes
// the key/value pair to it and tracks any node->child references as well as any
// node->external trie references.
func (c *committer) store(n node, force bool) (node, error) {
	// Don't store hashes or empty nodes.
	if _, isHash := n.(hashNode); n == nil || isHash {
		return n, nil
	}
	// Generate the RLP encoding of the node
	c.tmp.Reset()
	if err := rlp.Encode(&c.tmp, n); err != nil {
		panic("encode error: " + err.Error())
	}
	if len(c.tmp) < 32 && !force {
		return n, nil // Nodes smaller than 32 bytes are stored inside their parent
	}
	// Larger nodes are replaced by their hash and stored in the database.
	hash, _ := n.cache()
	if len(hash) == 0 {
		hash = c.hashData(c.tmp)
	}

	if c.leafCh != nil {
		c.leafCh <- &leaf{
			size: estimateSize(n),
			hash: common.BytesToHash(hash),
			node: n,
		}
	}

	return hash, nil
}

// commitLoop does the actual insert + leaf callback for nodes.
func (c *committer) commitLoop(db *Database) {
	for item := range c.leafCh {
		var (
			hash = item.hash
			size = item.size
			n    = item.node
		)
		// We are pooling the trie nodes into an intermediate memory cache
		db.lock.Lock()
		db.insert(hash, size, n)
		db.insertFreshNode(hash)
		db.lock.Unlock()

		if c.onleaf != nil {
			switch n := n.(type) {
			case *shortNode:
				if child, ok := n.Val.(valueNode); ok {
					c.onleaf(child, hash)
				}
			case *fullNode:
				for i := 0; i < 16; i++ {
					if child, ok := n.Children[i].(valueNode); ok {
						c.onleaf(child, hash)
					}
				}
			}
		}
	}
}

// hashData hashes the provided data
func (c *committer) hashData(data []byte) hashNode {
	n := make(hashNode, 32)
	c.sha.Reset()
	c.sha.Write(data)
	c.sha.Read(n)
	return n
}

// estimateSize estimates the size of an rlp-encoded node, without actually
// rlp-encoding it (zero allocs). This method has been experimentally tried, and with a trie
// with 1000 leafs, the only errors above 1% are on small shortnodes, where this
// method overestimates by 2 or 3 bytes (e.g. 37 instead of 35)
func estimateSize(n node) int {
	switch n := n.(type) {
	case *shortNode:
		// A short node contains a compacted key, and a value.
		return 3 + len(n.Key) + estimateSize(n.Val)
	case *fullNode:
		// A full node contains up to 16 hashes (some nils), and a key
		s := 3
		for i := 0; i < 16; i++ {
			if child := n.Children[i]; child != nil {
				s += estimateSize(child)
			} else {
				s++
			}
		}
		return s
	case valueNode:
		return 1 + len(n)
	case hashNode:
		return 1 + len(n)
	default:
		panic(fmt.Sprintf("node type %T", n))
	}
}
