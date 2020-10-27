package trie

import (
	"encoding/json"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/ethdb/memorydb"
	"testing"
)

func createKv(num int) map[common.Hash]common.Hash {
	base := make(map[common.Hash]common.Hash)
	for i := 0; i < num; i++ {
		base[common.BytesToHash(randBytes(32))] = common.BytesToHash(randBytes(32))
	}
	//text , _ := json.MarshalIndent(base, "", "")
	//fmt.Println(string(text))
	return base
}
func updateTrie(trie *SecureTrie, kv map[common.Hash]common.Hash) {
	for k, v := range kv {
		trie.Update(k.Bytes(), v.Bytes())
	}
}
func commit(trie *SecureTrie, triedb *Database) common.Hash {
	root, _ := trie.Commit(nil)
	triedb.Commit(root, false, false)
	//triedb.reference(root, common.Hash{})
	return root
}

func reference(triedb *Database, root, storageRoot common.Hash, keys map[common.Hash]common.Hash) {
	trie, _ := NewSecure(root, triedb)
	iter := trie.NodeIterator(nil)
	for iter.Next(true) {
		if iter.Leaf() {
			hash := common.BytesToHash(iter.LeafKey())
			if _, ok := keys[hash]; ok {
				//fmt.Println("reference:", storageRoot.Hex(), "   ", iter.Parent().Hex())
				triedb.reference(storageRoot, iter.Parent())
			}
		}
	}
}

func diff(triedb *Database, lastRoot, newRoot common.Hash) map[common.Hash]struct{} {
	mapA := make(map[common.Hash]struct{})
	mapB := make(map[common.Hash]struct{})

	trieA, _ := NewSecure(lastRoot, triedb)
	trieB, _ := NewSecure(newRoot, triedb)
	iterA := trieA.NodeIterator(nil)
	iterB := trieB.NodeIterator(nil)

	for iterB.Next(true) {
		mapB[iterB.Hash()] = struct{}{}
	}

	for iterA.Next(true) {
		hash := iterA.Hash()
		if _, ok := mapB[hash]; !ok {
			mapA[hash] = struct{}{}
		}
	}
	return mapA
}

func TestDirtyRefNodeIterator(t *testing.T) {
	type Data struct {
		Base1     map[common.Hash]common.Hash
		Storage1  map[common.Hash]common.Hash
		Base2     map[common.Hash]common.Hash
		Storage2  map[common.Hash]common.Hash
		Accounts1 map[common.Hash]common.Hash
		Accounts2 map[common.Hash]common.Hash
	}
	lastRoot, lastStorageRoot, newRoot, newStorageRoot := common.Hash{}, common.Hash{}, common.Hash{}, common.Hash{}

	var data *Data
	createData := func(new bool) {
		if new {
			accounts2 := make(map[common.Hash]common.Hash)
			accounts1 := map[common.Hash]common.Hash{
				common.BytesToHash(randBytes(32)): common.BytesToHash(randBytes(32)),
			}
			for k, _ := range accounts1 {
				accounts2[k] = common.BytesToHash(randBytes(32))
			}
			data = &Data{
				Base1:     createKv(11),
				Storage1:  createKv(11),
				Base2:     createKv(11),
				Storage2:  createKv(11),
				Accounts1: accounts1,
				Accounts2: accounts1,
			}
		} else {
			var d Data
			json.Unmarshal([]byte(""), &d)
			data = &d
		}
	}
	createData(true)
	//tojson := func (m interface{}) {
	//	text , _ := json.MarshalIndent(m, "", "")
	//	fmt.Println(string(text))
	//}

	db := memorydb.New()
	triedb := NewDatabase(db)
	trie, _ := NewSecure(common.Hash{}, triedb)
	storageTrie, _ := NewSecure(common.Hash{}, triedb)
	updateTrie(trie, data.Base1)
	updateTrie(trie, data.Accounts1)
	updateTrie(storageTrie, data.Storage1)
	lastRoot = commit(trie, triedb)
	lastStorageRoot = commit(storageTrie, triedb)

	accountsHash := func() map[common.Hash]common.Hash {
		m := make(map[common.Hash]common.Hash)
		for k, v := range data.Accounts1 {
			m[common.BytesToHash(trie.hashKey(k[:]))] = v
		}
		return m
	}

	reference(triedb, lastRoot, lastStorageRoot, accountsHash())

	updateTrie(trie, data.Base2)
	updateTrie(trie, data.Accounts2)
	updateTrie(storageTrie, data.Storage2)

	newRoot = commit(trie, triedb)
	newStorageRoot = commit(storageTrie, triedb)
	reference(triedb, newRoot, newStorageRoot, accountsHash())

	accountDiff := diff(triedb, lastRoot, newRoot)
	storageDiff := diff(triedb, lastStorageRoot, newStorageRoot)

	count := 0
	iter := newDiffIterator(triedb, lastRoot, newRoot)
	for iter.Next() {
		count++
		if _, ok := accountDiff[iter.Hash()]; ok {
			delete(accountDiff, iter.Hash())
			continue
		}
		if _, ok := storageDiff[iter.Hash()]; ok {
			delete(storageDiff, iter.Hash())
			continue
		}

		t.Error("delete wrong node")
	}
}
