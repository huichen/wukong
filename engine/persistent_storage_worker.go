package engine

import (
	"bytes"
	"encoding/gob"
	"github.com/henrylee2cn/wukong/types"
	"sync/atomic"
)

type persistentStorageIndexDocumentRequest struct {
	DocumentIndex *types.DocumentIndex
	Fields        interface{}
}

// 写入数据的协程
func (engine *Engine) persistentStorageIndexDocumentWorker(shard uint64) {
	for {
		request := <-engine.persistentStorageIndexDocumentChannels[shard]

		// 得到key
		b := []byte(request.DocumentIndex.DocId)

		// 得到value
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		err := enc.Encode(request)
		if err != nil {
			atomic.AddUint64(&engine.numDocumentsStored, 1)
			continue
		}

		// 将key-value写入数据库
		engine.dbs[shard].Set(b, buf.Bytes())
		atomic.AddUint64(&engine.numDocumentsStored, 1)
	}
}

// 删除数据
func (engine *Engine) persistentStorageRemoveDocumentWorker(shard uint64, docId string) {
	// 得到key
	b := []byte(docId)

	// 从数据库删除该key
	engine.dbs[shard].Delete(b)
}

// 恢复数据
func (engine *Engine) persistentStorageInitWorker(shard uint64) {
	engine.dbs[shard].ForEach(func(key, value []byte) error {
		// 得到documentIndex
		buf := bytes.NewReader(value)
		dec := gob.NewDecoder(buf)
		var request = new(persistentStorageIndexDocumentRequest)
		err := dec.Decode(request)

		// 添加索引文档
		if err == nil {
			// 发送至索引器处理
			engine.indexerAddDocumentChannels[shard] <- indexerAddDocumentRequest{
				document: request.DocumentIndex,
			}
			// 发送至排序器处理
			engine.rankerAddScoringFieldsChannels[shard] <- rankerAddScoringFieldsRequest{
				docId:  request.DocumentIndex.DocId,
				fields: request.Fields,
			}

			atomic.AddUint64(&engine.numIndexingRequests, 1)
			atomic.AddUint64(&engine.numDocumentsStored, 1)
		}
		return nil
	})

	engine.persistentStorageInitChannel <- true
}
