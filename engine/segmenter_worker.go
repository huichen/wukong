package engine

import (
	"github.com/henrylee2cn/wukong/types"
	"runtime"
)

type segmenterRequest struct {
	docId             string
	hash              uint32
	data              types.DocumentIndexData
	documentIndexChan chan<- *types.DocumentIndex
}

func (engine *Engine) segmenterWorker() {
	for {
		request := <-engine.segmenterChannel
		shard := engine.getShard(request.hash)

		tokensMap := make(map[string][]int)
		numTokens := 0
		if request.data.Content != "" {
			// 当文档正文不为空时，优先从内容分词中得到关键词
			segments := engine.segmenter.Segment([]byte(request.data.Content))
			for _, segment := range segments {
				token := segment.Token().Text()
				if !engine.stopTokens.IsStopToken(token) {
					tokensMap[token] = append(tokensMap[token], segment.Start())
				}
			}
			numTokens = len(segments)
		} else {
			// 否则载入用户输入的关键词
			for _, t := range request.data.Tokens {
				if !engine.stopTokens.IsStopToken(t.Text) {
					tokensMap[t.Text] = t.Locations
				}
			}
			numTokens = len(request.data.Tokens)
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
				TokenLength: float32(numTokens),
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

		if request.documentIndexChan != nil {
			// 返回DocumentIndex
			request.documentIndexChan <- indexerRequest.document
			// 统计被执行次数
			documentIndexChanCount[documentIndexChan]++
			// 等待通道被写满，即所有同类文档被分词完毕
			for documentIndexChanCount[request.documentIndexChan] < cap(request.documentIndexChan) {
				runtime.Gosched()
			}

			// 此时此处，外部插队执行关于documentIndexChan的处理...

			// 等待通道数据被外部程序全部读出，即确保外部程序对于documentIndexChan的相关处理在建立索引前完成
			for len(request.documentIndexChan) > 0 {
				runtime.Gosched()
			}

			// 清除记录
			if _, ok := documentIndexChanCount[request.documentIndexChan]; ok {
				delete(documentIndexChanCount[request.documentIndexChan], request.documentIndexChan)
			}
		}

		rankerRequest := rankerAddScoringFieldsRequest{
			docId: request.docId, fields: request.data.Fields}
		engine.rankerAddScoringFieldsChannels[shard] <- rankerRequest
	}
}
