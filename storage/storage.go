package storage

import (
	"fmt"
	"os"
	"time"
)

const DEFAULT_STORAGE_ENGIND = "bolt"

var _supported_storage = map[string]func(path string) (Storage, error){
	"kv":   openKVStorage,
	"bolt": openBoltStorage,
}

func RegisterStorageEngine(name string, fn func(path string) (Storage, error)) {
	_supported_storage[name] = fn
}

type Options struct {
	Timeout time.Duration
}

type Storage interface {
	Set(k, v []byte) error
	Get(k []byte) ([]byte, error)
	Delete(k []byte) error
	ForEach(fn func(k, v []byte) error) error
	Close() error
	WAlName() string
}

func OpenStorage(path string) (Storage, error) {
	wse := os.Getenv("WUKONG_STORAGE_ENGINE")
	if wse == "" {
		wse = DEFAULT_STORAGE_ENGIND
	}
	if has, fn := _supported_storage[wse]; has {
		return fn(path)
	}
	return nil, fmt.Errorf("unsupported storage engine %v", wse)
}
