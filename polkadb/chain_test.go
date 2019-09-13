package polkadb

import (
	"fmt"
	"github.com/ChainSafe/gossamer/common"
	"testing"
)

func TestBlockDB_SetBestHash(t *testing.T) {
	dbSrv, err := NewDatabaseService("./test_data")
	if err != nil {
		t.Log("database was not created ", "error ", err)
	}
	h := "0x8550326cee1e1b768a254095b412e0db58523c2b5df9b7d2540b4513d475ce7f"
	byteRes, _ := common.HexToHash(h)
	r := [32]byte(byteRes)
	fmt.Println(r)
	dbSrv.BlockDB.SetBestHash(byteRes)


	hash := dbSrv.BlockDB.GetBestHash()
	fmt.Println(hash)
}
