package engine

import (
	"bytes"
	"encoding/gob"
	"github.com/henrylee2cn/wukong/types"
	"io"
	"log"
	"sync/atomic"
)

type persistentStorageIndexDocumentRequest struct {
	DocumentIndex *types.DocumentIndex
	Fields        interface{}
}

func (engine *Engine) persistentStorageIndexDocumentWorker(shard int) {
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

func (engine *Engine) persistentStorageRemoveDocumentWorker(shard int, docId string) {
	// 得到key
	b := []byte(docId)

	// 从数据库删除该key
	engine.dbs[shard].Delete(b)
}

func (engine *Engine) persistentStorageInitWorker(shard int) {
	iter, err := engine.dbs[shard].SeekFirst()
	if err == io.EOF {
		engine.persistentStorageInitChannel <- true
		return
	} else if err != nil {
		engine.persistentStorageInitChannel <- true
		log.Fatal("无法遍历数据库")
	}
	for {
		_, value, err := iter.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			continue
		}

		// 得到documentIndex
		buf := bytes.NewReader(value)
		dec := gob.NewDecoder(buf)
		var request = new(persistentStorageIndexDocumentRequest)
		err = dec.Decode(request)
		if err != nil {
			continue
		}

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
	engine.persistentStorageInitChannel <- true
}
