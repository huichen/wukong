package engine

import (
	"github.com/huichen/wukong/types"
)

type rankerAddScoringFieldsRequest struct {
	docId  uint64
	fields interface{}
}

type rankerRankRequest struct {
	docs                []types.IndexedDocument
	options             types.RankOptions
	rankerReturnChannel chan rankerReturnRequest
}

type rankerReturnRequest struct {
	docs types.ScoredDocuments
}

type rankerRemoveScoringFieldsRequest struct {
	docId uint64
}

func (engine *Engine) rankerAddScoringFieldsWorker(shard int) {
	for {
		request := <-engine.rankerAddScoringFieldsChannels[shard]
		engine.rankers[shard].AddScoringFields(request.docId, request.fields)
	}
}

func (engine *Engine) rankerRankWorker(shard int) {
	for {
		request := <-engine.rankerRankChannels[shard]
		if request.options.MaxOutputs != 0 {
			request.options.MaxOutputs += request.options.OutputOffset
		}
		request.options.OutputOffset = 0
		outputDocs := engine.rankers[shard].Rank(request.docs, request.options)
		request.rankerReturnChannel <- rankerReturnRequest{docs: outputDocs}
	}
}

func (engine *Engine) rankerRemoveScoringFieldsWorker(shard int) {
	for {
		request := <-engine.rankerRemoveScoringFieldsChannels[shard]
		engine.rankers[shard].RemoveScoringFields(request.docId)
	}
}
