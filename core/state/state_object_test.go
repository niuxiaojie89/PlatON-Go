package state

import (
	"bytes"
	"encoding/binary"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"testing"
)

func TestStateObject(t *testing.T) {
	x := Account{
		Root: common.HexToHash("0x1000000000000000000000000000000000000001"),
	}
	x2 := newObject(nil, common.HexToAddress("0x1000000000000000000000000000000000000001"), x)
	x3 := x2.deepCopy(nil)
	x2.data.Root = common.HexToHash("0x1000000000000000000000000000000000000012")
	t.Log(x2.data.Root.String())
	t.Log(x3.data.Root.String())
}

func TestStateObjectValuePrefix(t *testing.T) {
	hash := common.HexToHash("0x1000000000000000000000000000000000000001")
	addr := common.HexToAddress("0x1000000000000000000000000000000000000001")
	key := []byte("key")
	value := []byte("value")
	x2 := newObject(nil, addr, Account{
		Root:             hash,
		StorageKeyPrefix: addr.Bytes(),
	})

	pack := make([]byte, 8)
	binary.BigEndian.PutUint64(pack[0:8], uint64(999999))
	dbValue := x2.getPrefixValue(key, pack, value)
	if !bytes.Equal(value, x2.removePrefixValue(dbValue)) {
		t.Fatal("prefix error")
	}
}
