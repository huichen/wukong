package storage

import (
	"fmt"
)

const DEFAULT_STORAGE_ENGINE = "bolt"

// const DEFAULT_STORAGE_ENGINE = "kv"

var supportedStorage = map[string]func(path string) (Storage, error){
	"kv":   openKVStorage,
	"bolt": openBoltStorage,
}

func RegisterStorageEngine(name string, fn func(path string) (Storage, error)) {
	supportedStorage[name] = fn
}

type Storage interface {
	Set(k, v []byte) error
	Get(k []byte) ([]byte, error)
	Delete(k []byte) error
	ForEach(fn func(k, v []byte) error) error
	Close() error
	WALName() string
}

func OpenStorage(path, wse string) (Storage, error) {
	switch wse {
	case "bolt", "kv":
	default:
		wse = DEFAULT_STORAGE_ENGINE
	}
	if fn, has := supportedStorage[wse]; has {
		return fn(path)
	}
	return nil, fmt.Errorf("unsupported storage engine %v", wse)
}
