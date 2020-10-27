package trie

import (
	"bytes"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"os"
	"sort"
)

type dirtyNodeState struct {
	cache    *cachedNode
	hash     common.Hash
	children []common.Hash
	index    uint8
	pathlen  uint8
}
type dirtyNodeIterator struct {
	db    *Database
	root  common.Hash
	stack []*dirtyNodeState
	path  []byte
}

func newDirtyNodeIterator(db *Database, root common.Hash) *dirtyNodeIterator {
	return &dirtyNodeIterator{
		db:   db,
		root: root,
	}
}
func (d *dirtyNodeIterator) Next() bool {
	d.seek()
	return len(d.stack) != 0
}
func (d *dirtyNodeIterator) seek() {
	if len(d.stack) == 0 {
		s := d.findNode(d.root)
		if s != nil {
			d.stack = append(d.stack, s)
		}
		return
	}

	for len(d.stack) > 0 {
		s := d.stack[len(d.stack)-1]
		if len(s.children) > 0 {
			for i, k := range s.children {
				if c := d.findNode(k); c != nil {
					if len(s.children) == i+1 {
						s.children = []common.Hash{}
					} else {
						s.children = s.children[i+1:]
					}
					c.pathlen = uint8(len(d.path))
					d.stack = append(d.stack, c)
					d.path = append(d.path, byte(0xff))
					return
				}
			}
		}

		if child, path := d.findChild(s); child != nil {
			child.pathlen = uint8(len(d.path))
			d.stack = append(d.stack, child)
			d.path = append(d.path, path...)
			return
		}
		d.pop()
	}
}

func (d *dirtyNodeIterator) pop() {
	pop := func() {
		s := d.stack[len(d.stack)-1]
		d.path = d.path[:s.pathlen]
		d.stack = d.stack[:len(d.stack)-1]
	}
	pop()
	for len(d.stack) > 0 {
		s := d.stack[len(d.stack)-1]
		if len(s.cache.children) > 0 {
			break
		}
		if _, ok := s.cache.node.(rawFullNode); ok {
			break
		}
		pop()
	}
}

func (d *dirtyNodeIterator) findChild(s *dirtyNodeState) (*dirtyNodeState, []byte) {
	var path []byte
	switch n := s.cache.node.(type) {
	case rawFullNode:
		for i := s.index; i < 17; i++ {
			s.index = i + 1
			if child, path := d.nextCachedNode(n[i], append(path, i)); child != nil {
				return child, path
			}
		}
	default:
		if child, path := d.nextCachedNode(s.cache.node, path); child != nil {
			return child, path
		}
	}
	return nil, path
}

func (d *dirtyNodeIterator) nextCachedNode(n node, path []byte) (*dirtyNodeState, []byte) {
	switch n := n.(type) {
	case *rawShortNode:
		return d.nextCachedNode(n.Val, append(path, compactToHex(n.Key)...))
	case rawFullNode:
		for i := 0; i < 17; i++ {
			if node, path := d.nextCachedNode(n[i], append(path, byte(i))); node != nil {
				return node, path
			}
		}
	case hashNode:
		if node := d.findNode(common.BytesToHash(n)); node != nil {
			if c, ok := node.cache.node.(*rawShortNode); ok {
				return node, append(path, compactToHex(c.Key)...)
			}
			return node, path
		} else {
			return nil, path
		}

	case valueNode:
		return nil, path
	}
	return nil, path
}

func (d *dirtyNodeIterator) findNode(hash common.Hash) *dirtyNodeState {
	sort := func(m map[common.Hash]uint16) []common.Hash {
		array := make([]common.Hash, 0, len(m))
		for k, _ := range m {
			array = append(array, k)
		}
		sort.Slice(array, func(i, j int) bool {
			return array[i].Big().Cmp(array[j].Big()) < 0
		})
		return array
	}
	if n, ok := d.db.dirties[hash]; ok {
		return &dirtyNodeState{
			cache:    n,
			hash:     hash,
			index:    0,
			children: sort(n.children),
		}
	}
	return nil
}

func (d *dirtyNodeIterator) Hash() common.Hash {
	if len(d.stack) == 0 {
		return common.Hash{}
	}
	return d.stack[len(d.stack)-1].hash

}

func (d *dirtyNodeIterator) Path() []byte {
	return d.path
}

func (d *dirtyNodeIterator) Error() error {
	return nil
}

type diffIterator struct {
	db *Database
	a  *dirtyNodeIterator
	b  *dirtyNodeIterator
}

func newDiffIterator(db *Database, a, b common.Hash) *diffIterator {
	iter := &diffIterator{
		db: db,
		a: &dirtyNodeIterator{
			db:   db,
			root: a,
		},
		b: &dirtyNodeIterator{
			db:   db,
			root: b,
		},
	}
	return iter
}

func (d *diffIterator) Next() bool {
	nextA := d.a.Next()
	nextB := d.b.Next()

	for nextA && nextB {
		//fmt.Println("A", hexutil.Encode(d.a.Path()), d.a.Hash().Hex())
		//fmt.Println("B", hexutil.Encode(d.b.Path()), d.b.Hash().Hex())
		switch d.compare(d.a.Path(), d.b.Path()) {
		case 0:
			if d.a.Hash() == d.b.Hash() {
				nextA = d.a.Next()
				nextB = d.b.Next()
			} else {
				return true
			}
		case 1:
			nextB = d.b.Next()
		case -1:
			nextA = d.a.Next()
		}
	}

	return false
}

func (d *diffIterator) compare(pathA, pathB []byte) int {
	os.Stdout.Sync()
	return bytes.Compare(pathA, pathB)
}

func (d *diffIterator) Error() error {
	if d.a.Error() != nil {
		return d.a.Error()
	}
	if d.b.Error() != nil {
		return d.a.Error()
	}
	return nil
}

func (d *diffIterator) Hash() common.Hash {
	return d.a.Hash()
}
