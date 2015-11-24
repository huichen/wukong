package engine

import (
	"github.com/henrylee2cn/wukong/types"
	"runtime"
	"sync/atomic"
)

type segmenterRequest struct {
	docId             string
	shard             uint64
	data              types.DocumentIndexData
	documentIndexChan chan *types.DocumentIndex
}

// 只有IndexDocument时用到
func (engine *Engine) segmenterWorker() {
	for {
		request := <-engine.segmenterChannel

		tokensMap := make(map[string][]int)
		numTokens := 0
		if !engine.initOptions.NotUsingSegmenter && request.data.Content != "" {
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
			if !engine.initOptions.NotUsingSegmenter {
				if !engine.stopTokens.IsStopToken(label) {
					tokensMap[label] = []int{}
				}
			} else {
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
				Starts:    v,
			}
			iTokens++
		}

		if request.documentIndexChan != nil {
			go engine.segmenterWorkerExec(request, indexerRequest)
		} else {
			engine.segmenterWorkerExec(request, indexerRequest)
		}
	}
}

func (engine *Engine) segmenterWorkerExec(request segmenterRequest, indexerRequest indexerAddDocumentRequest) {
	if request.documentIndexChan != nil {
		// 返回DocumentIndex
		request.documentIndexChan <- indexerRequest.document

		// 统计被执行次数
		documentIndexChanCount[request.documentIndexChan].Mutex.Lock()
		documentIndexChanCount[request.documentIndexChan].Num++
		documentIndexChanCount[request.documentIndexChan].Mutex.Unlock()

		// 此时此处，外部插队执行关于documentIndexChan的处理...

		// 等待通道被写满，即所有同类文档被分词完毕
		for documentIndexChanCount[request.documentIndexChan].Num < cap(request.documentIndexChan) {
			runtime.Gosched()
		}

		// 等待通道数据被外部程序全部读出，即确保外部程序对于documentIndexChan的相关处理在建立索引前完成
		for len(request.documentIndexChan) > 0 {
			runtime.Gosched()
		}

		// 等待通道关闭
		for range request.documentIndexChan {
			// 获取任意一个信号，终止当前索引任务
			// 清除记录
			if _, ok := documentIndexChanCount[request.documentIndexChan]; ok {
				delete(documentIndexChanCount, request.documentIndexChan)
			}
			atomic.AddUint64(&engine.numDocumentsIndexed, 1)
			atomic.AddUint64(&engine.numDocumentsStored, 1)
			return
		}
		// 清除记录
		if _, ok := documentIndexChanCount[request.documentIndexChan]; ok {
			delete(documentIndexChanCount, request.documentIndexChan)
		}
	}
	// 发送至索引器处理
	engine.indexerAddDocumentChannels[request.shard] <- indexerRequest
	// 发送至排序器处理
	engine.rankerAddDocChannels[request.shard] <- rankerAddDocRequest{
		docId:  request.docId,
		fields: request.data.Fields,
	}

	// 存索引信息至数据库
	if engine.initOptions.UsePersistentStorage {
		// for range indexerRequest.document.Keywords {
		// if engine.indexers[request.shard].FoundKeyword(keyword.Text) {
		// 	continue
		// }
		engine.persistentStorageIndexDocumentChannels[request.shard] <- persistentStorageIndexDocumentRequest{
			DocumentIndex: indexerRequest.document,
			Fields:        request.data.Fields,
		}
		// }
	}
}
