package storage

import (
	"github.com/huichen/wukong/storage"
	"os"
	"testing"
)

func TestOpenOrCreateKv(t *testing.T) {
	db, err := storage.OpenStorage("test")
	Expect(t, "<nil>", err)
	db.Close()

	db, err := storage.OpenStorage("test")
	Expect(t, "<nil>", err)
	db.Buc
	err = db.Set([]byte("key1"), []byte("value1"))
	Expect(t, "<nil>", err)

	buffer := make([]byte, 100)
	buffer, err = db.Get(nil, []byte("key1"))
	Expect(t, "<nil>", err)
	Expect(t, "value1", string(buffer))

	walFile := db.WALName()
	db.Close()
	os.Remove(walFile)
	os.Remove("test")
}
