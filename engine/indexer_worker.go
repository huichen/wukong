package engine

import (
	"github.com/huichen/wukong/types"
	"sync/atomic"
)

type indexerAddDocumentRequest struct {
	document *types.DocumentIndex
}

type indexerLookupRequest struct {
	countDocsOnly       bool
	tokens              []string
	labels              []string
	docIds              map[uint64]bool
	options             types.RankOptions
	rankerReturnChannel chan rankerReturnRequest
	orderless           bool
}

type indexerRemoveDocRequest struct {
	docId uint64
}

func (engine *Engine) indexerAddDocumentWorker(shard int) {
	for {
		request := <-engine.indexerAddDocumentChannels[shard]
		engine.indexers[shard].AddDocument(request.document)
		atomic.AddUint64(&engine.numTokenIndexAdded,
			uint64(len(request.document.Keywords)))
		atomic.AddUint64(&engine.numDocumentsIndexed, 1)
	}
}

func (engine *Engine) indexerLookupWorker(shard int) {
	for {
		request := <-engine.indexerLookupChannels[shard]

		var docs []types.IndexedDocument
		var numDocs int
		if request.docIds == nil {
			docs, numDocs = engine.indexers[shard].Lookup(request.tokens, request.labels, nil, request.countDocsOnly)
		} else {
			docs, numDocs = engine.indexers[shard].Lookup(request.tokens, request.labels, request.docIds, request.countDocsOnly)
		}

		if request.countDocsOnly {
			request.rankerReturnChannel <- rankerReturnRequest{numDocs: numDocs}
			continue
		}

		if len(docs) == 0 {
			request.rankerReturnChannel <- rankerReturnRequest{}
			continue
		}

		if request.orderless {
			var outputDocs []types.ScoredDocument
			for _, d := range docs {
				outputDocs = append(outputDocs, types.ScoredDocument{
					DocId: d.DocId,
					TokenSnippetLocations: d.TokenSnippetLocations,
					TokenLocations:        d.TokenLocations})
			}
			request.rankerReturnChannel <- rankerReturnRequest{
				docs:    outputDocs,
				numDocs: len(outputDocs),
			}
			continue
		}

		rankerRequest := rankerRankRequest{
			countDocsOnly:       request.countDocsOnly,
			docs:                docs,
			options:             request.options,
			rankerReturnChannel: request.rankerReturnChannel,
		}
		engine.rankerRankChannels[shard] <- rankerRequest
	}
}

func (engine *Engine) indexerRemoveDocWorker(shard int) {
	for {
		request := <-engine.indexerRemoveDocChannels[shard]
		engine.indexers[shard].RemoveDoc(request.docId)
	}
}
