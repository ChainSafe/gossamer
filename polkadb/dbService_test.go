package polkadb

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

// Returns started dbService
func newTestDBService(t *testing.T) (*DbService, func()) {
	dir, err := ioutil.TempDir(os.TempDir(), "test_data")
	if err != nil {
		t.Fatal("failed to create test file: " + err.Error())
	}
	db, err := NewDatabaseService(dir)
	if err != nil {
		t.Fatal("failed to create test database: " + err.Error())
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
	dir, err := ioutil.TempDir(os.TempDir(), "test_data")
	if err != nil {
		t.Fatal("failed to create test file: " + err.Error())
	}
	db, err := NewDatabaseService(dir)
	if err != nil {
		t.Fatal("failed to create test database: " + err.Error())
	}

	err = db.Start()
	if err != nil {
		t.Fatal(err)
	}
}
