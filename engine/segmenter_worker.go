package engine

import (
	"github.com/huichen/wukong/types"
)

type segmenterRequest struct {
	docId uint64
	hash  uint32
	data  types.DocumentIndexData
}

func (engine *Engine) segmenterWorker() {
	for {
		request := <-engine.segmenterChannel
		shard := engine.getShard(request.hash)
		segments := engine.segmenter.Segment([]byte(request.data.Content))
		tokensMap := make(map[string][]int)

		// 加入分词得到的关键词
		for _, segment := range segments {
			token := segment.Token().Text()
			if !engine.stopTokens.IsStopToken(token) {
				tokensMap[token] = append(tokensMap[token], segment.Start())
			}
		}

		// 加入非分词的文档标签
		for _, label := range request.data.Labels {
			if !engine.stopTokens.IsStopToken(label) {
				tokensMap[label] = []int{}
			}
		}

		indexerRequest := indexerAddDocumentRequest{
			document: &types.DocumentIndex{
				DocId:       request.docId,
				TokenLength: float32(len(segments)),
				Keywords:    make([]types.KeywordIndex, len(tokensMap)),
			},
		}
		iTokens := 0
		for k, v := range tokensMap {
			indexerRequest.document.Keywords[iTokens] = types.KeywordIndex{
				Text: k,
				// 非分词标注的词频设置为0，不参与tf-idf计算
				Frequency: float32(len(v)),
				Starts:    v}
			iTokens++
		}
		engine.indexerAddDocumentChannels[shard] <- indexerRequest
		rankerRequest := rankerAddScoringFieldsRequest{
			docId: request.docId, fields: request.data.Fields}
		engine.rankerAddScoringFieldsChannels[shard] <- rankerRequest
	}
}
