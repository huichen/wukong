package engine

import (
	"github.com/henrylee2cn/wukong/types"
	"sync/atomic"
)

type indexerAddDocumentRequest struct {
	document *types.DocumentIndex
}

type indexerLookupRequest struct {
	tokens              []string
	labels              []string
	docIds              []string
	options             types.RankOptions
	rankerReturnChannel chan rankerReturnRequest
}

func (engine *Engine) indexerAddDocumentWorker(shard int) {
	for {
		request := <-engine.indexerAddDocumentChannels[shard]
		// 加索引至内存
		engine.indexers[shard].AddDocument(request.document)
		atomic.AddUint64(&engine.numTokenIndexAdded, uint64(len(request.document.Keywords)))
		atomic.AddUint64(&engine.numDocumentsIndexed, 1)
	}
}

func (engine *Engine) indexerLookupWorker(shard int) {
	for {
		request := <-engine.indexerLookupChannels[shard]

		var docs []types.IndexedDocument
		if len(request.docIds) == 0 {
			docs = engine.indexers[shard].Lookup(request.tokens, request.labels, nil)
		} else {
			docIds := make(map[string]bool)
			for _, ids := range request.docIds {
				docIds[ids] = true
			}
			docs = engine.indexers[shard].Lookup(request.tokens, request.labels, &docIds)
		}

		if len(docs) == 0 {
			request.rankerReturnChannel <- rankerReturnRequest{}
			continue
		}

		engine.rankerRankChannels[shard] <- rankerRankRequest{
			docs:                docs,
			options:             request.options,
			rankerReturnChannel: request.rankerReturnChannel,
		}
	}
}
