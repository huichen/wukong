package engine

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"github.com/huichen/wukong/core"
	"github.com/huichen/wukong/types"
	"sync"
	"sync/atomic"
)

type persistentStorageIndexDocumentRequest struct {
	typ string //"info"or"index"

	// typ=="info"时，以下两个字段有效
	docId   uint64
	docInfo *types.DocInfo

	// typ=="index"时，以下两个字段有效
	keyword        string
	keywordIndices *types.KeywordIndices
}

func (engine *Engine) persistentStorageIndexDocumentWorker(shard int) {
	for {
		request := <-engine.persistentStorageIndexDocumentChannels[shard]
		switch request.typ {
		case "info":
			// 得到key
			b := make([]byte, 10)
			length := binary.PutUvarint(b, request.docId)

			// 得到value
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			err := enc.Encode(request.docInfo)
			if err != nil {
				atomic.AddUint64(&engine.numDocumentsStored, 1)
				return
			}

			// 将key-value写入数据库
			engine.dbs[shard][getDB(request.typ)].Set(b[0:length], buf.Bytes())
			atomic.AddUint64(&engine.numDocumentsStored, 1)

		case "index":
			// 得到key
			b := []byte(request.keyword)

			// 得到value
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			err := enc.Encode(request.keywordIndices)
			if err != nil {
				return
			}

			// 将key-value写入数据库
			engine.dbs[shard][getDB(request.typ)].Set(b, buf.Bytes())
		}
	}
}

func (engine *Engine) persistentStorageRemoveDocumentWorker(docId uint64, shard int) {
	// 得到key
	b := make([]byte, 10)
	length := binary.PutUvarint(b, docId)

	// 从数据库删除该key
	engine.dbs[shard][getDB("info")].Delete(b[0:length])
}

func (engine *Engine) persistentStorageInitWorker(shard int) {
	var finish sync.WaitGroup
	finish.Add(2)
	// 恢复docInfo
	go func() {
		defer finish.Add(-1)
		engine.dbs[shard][getDB("info")].ForEach(func(k, v []byte) error {
			key, value := k, v
			// 得到docID
			docId, _ := binary.Uvarint(key)

			// 得到data
			buf := bytes.NewReader(value)
			dec := gob.NewDecoder(buf)
			var data types.DocInfo
			err := dec.Decode(&data)
			if err == nil {
				// 添加索引
				core.AddDocInfo(shard, docId, &data)
			}
			return nil
		})
	}()

	// 恢复invertedIndex
	go func() {
		defer finish.Add(-1)
		engine.dbs[shard][getDB("index")].ForEach(func(k, v []byte) error {
			key, value := k, v
			// 得到keyword
			keyword := string(key)

			// 得到data
			buf := bytes.NewReader(value)
			dec := gob.NewDecoder(buf)
			var data types.KeywordIndices
			err := dec.Decode(&data)
			if err == nil {
				// 添加索引
				core.AddKeywordIndices(shard, keyword, &data)
			}
			return nil
		})
	}()
	finish.Wait()
	engine.persistentStorageInitChannel <- true
}
