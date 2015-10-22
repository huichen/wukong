package utils

import (
	//"github.com/cznic/kv"
	"github.com/huichen/wukong/storage"
)

// 打开或者创建KV数据库
// 当path指向的数据库存在时打开该数据库，否则尝试在该路径处创建新数据库
// func OpenOrCreateKv(path string, options *kv.Options) (*kv.DB, error) {
// 	db, errOpen := kv.Open(path, options)
// 	if errOpen != nil {
// 		var errCreate error
// 		db, errCreate = kv.Create(path, options)
// 		if errCreate != nil {
// 			return db, errCreate
// 		}
// 	}

// 	return db, nil
// }

func OpenOrCreateKv(path string, options *storage.Options) (storage.Storage, error) {
	return storage.OpenStorage(path)
}
