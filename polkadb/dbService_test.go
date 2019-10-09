package polkadb

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func newTestDBService() (*DbService, func()) {
	dir, err := ioutil.TempDir(os.TempDir(), "test_data")
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}
	db, err := NewDatabaseService(dir)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}
	db.Start()
	return db, func() {
		db.Stop()
		if err := os.RemoveAll(dir); err != nil {
			fmt.Println("removal of temp directory test_data failed")
		}
	}
}

func TestDbService_Start(t *testing.T) {
	db, remove := newTestDBService()
	defer remove()

	err := db.Start()
	if e := <- err; e != nil {
		t.Fatal(e)
	}

	err = db.Stop()
	if e := <- err; e != nil {
		t.Fatal(e)
	}
}

func TestDb_Close(t *testing.T) {
	db, remove := newTestDBService()
	defer remove()

	err := db.StateDB.Db.Close()
	if err != nil {
		t.Fatalf("get returned wrong result, got %v", err)
	}
}
