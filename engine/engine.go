package engine

import (
	"fmt"
	"github.com/huichen/murmur"
	"github.com/huichen/sego"
	"github.com/huichen/wukong/core"
	"github.com/huichen/wukong/storage"
	"github.com/huichen/wukong/types"
	"github.com/huichen/wukong/utils"
	"log"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	NumNanosecondsInAMillisecond = 1000000
	PersistentStorageFilePrefix  = "wukong"
)

type Engine struct {
	// 计数器，用来统计有多少文档被索引等信息
	numDocumentsIndexed uint64
	numIndexingRequests uint64
	numTokenIndexAdded  uint64
	numDocumentsStored  uint64

	// 记录初始化参数
	initOptions types.EngineInitOptions
	initialized bool

	indexers   []core.Indexer
	rankers    []core.Ranker
	segmenter  sego.Segmenter
	stopTokens StopTokens
	dbs        []storage.Storage

	// 建立索引器使用的通信通道
	segmenterChannel           chan segmenterRequest
	indexerAddDocumentChannels []chan indexerAddDocumentRequest
	indexerRemoveDocChannels   []chan indexerRemoveDocRequest
	rankerAddDocChannels       []chan rankerAddDocRequest

	// 建立排序器使用的通信通道
	indexerLookupChannels   []chan indexerLookupRequest
	rankerRankChannels      []chan rankerRankRequest
	rankerRemoveDocChannels []chan rankerRemoveDocRequest

	// 建立持久存储使用的通信通道
	persistentStorageIndexDocumentChannels []chan persistentStorageIndexDocumentRequest
	persistentStorageInitChannel           chan bool
}

func (engine *Engine) Init(options types.EngineInitOptions) {
	// 将线程数设置为CPU数
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 初始化初始参数
	if engine.initialized {
		log.Fatal("请勿重复初始化引擎")
	}
	options.Init()
	engine.initOptions = options
	engine.initialized = true

	if !options.NotUsingSegmenter {
		// 载入分词器词典
		engine.segmenter.LoadDictionary(options.SegmenterDictionaries)

		// 初始化停用词
		engine.stopTokens.Init(options.StopTokenFile)
	}

	// 初始化索引器和排序器
	for shard := 0; shard < options.NumShards; shard++ {
		engine.indexers = append(engine.indexers, core.Indexer{})
		engine.indexers[shard].Init(*options.IndexerInitOptions)

		engine.rankers = append(engine.rankers, core.Ranker{})
		engine.rankers[shard].Init()
	}

	// 初始化分词器通道
	engine.segmenterChannel = make(
		chan segmenterRequest, options.NumSegmenterThreads)

	// 初始化索引器通道
	engine.indexerAddDocumentChannels = make(
		[]chan indexerAddDocumentRequest, options.NumShards)
	engine.indexerRemoveDocChannels = make(
		[]chan indexerRemoveDocRequest, options.NumShards)
	engine.indexerLookupChannels = make(
		[]chan indexerLookupRequest, options.NumShards)
	for shard := 0; shard < options.NumShards; shard++ {
		engine.indexerAddDocumentChannels[shard] = make(
			chan indexerAddDocumentRequest,
			options.IndexerBufferLength)
		engine.indexerRemoveDocChannels[shard] = make(
			chan indexerRemoveDocRequest,
			options.IndexerBufferLength)
		engine.indexerLookupChannels[shard] = make(
			chan indexerLookupRequest,
			options.IndexerBufferLength)
	}

	// 初始化排序器通道
	engine.rankerAddDocChannels = make(
		[]chan rankerAddDocRequest, options.NumShards)
	engine.rankerRankChannels = make(
		[]chan rankerRankRequest, options.NumShards)
	engine.rankerRemoveDocChannels = make(
		[]chan rankerRemoveDocRequest, options.NumShards)
	for shard := 0; shard < options.NumShards; shard++ {
		engine.rankerAddDocChannels[shard] = make(
			chan rankerAddDocRequest,
			options.RankerBufferLength)
		engine.rankerRankChannels[shard] = make(
			chan rankerRankRequest,
			options.RankerBufferLength)
		engine.rankerRemoveDocChannels[shard] = make(
			chan rankerRemoveDocRequest,
			options.RankerBufferLength)
	}

	// 初始化持久化存储通道
	if engine.initOptions.UsePersistentStorage {
		engine.persistentStorageIndexDocumentChannels =
			make([]chan persistentStorageIndexDocumentRequest,
				engine.initOptions.PersistentStorageShards)
		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			engine.persistentStorageIndexDocumentChannels[shard] = make(
				chan persistentStorageIndexDocumentRequest)
		}
		engine.persistentStorageInitChannel = make(
			chan bool, engine.initOptions.PersistentStorageShards)
	}

	// 启动分词器
	for iThread := 0; iThread < options.NumSegmenterThreads; iThread++ {
		go engine.segmenterWorker()
	}

	// 启动索引器和排序器
	for shard := 0; shard < options.NumShards; shard++ {
		go engine.indexerAddDocumentWorker(shard)
		go engine.indexerRemoveDocWorker(shard)
		go engine.rankerAddDocWorker(shard)
		go engine.rankerRemoveDocWorker(shard)

		for i := 0; i < options.NumIndexerThreadsPerShard; i++ {
			go engine.indexerLookupWorker(shard)
		}
		for i := 0; i < options.NumRankerThreadsPerShard; i++ {
			go engine.rankerRankWorker(shard)
		}
	}

	// 启动持久化存储工作协程
	if engine.initOptions.UsePersistentStorage {
		err := os.MkdirAll(engine.initOptions.PersistentStorageFolder, 0700)
		if err != nil {
			log.Fatal("无法创建目录", engine.initOptions.PersistentStorageFolder)
		}

		// 打开或者创建数据库
		engine.dbs = make([]storage.Storage, engine.initOptions.PersistentStorageShards)
		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			dbPath := engine.initOptions.PersistentStorageFolder + "/" + PersistentStorageFilePrefix + "." + strconv.Itoa(shard)
			db, err := storage.OpenStorage(dbPath)
			if db == nil || err != nil {
				log.Fatal("无法打开数据库", dbPath, ": ", err)
			}
			engine.dbs[shard] = db
		}

		// 从数据库中恢复
		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			go engine.persistentStorageInitWorker(shard)
		}

		// 等待恢复完成
		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			<-engine.persistentStorageInitChannel
		}
		for {
			runtime.Gosched()
			if engine.numIndexingRequests == engine.numDocumentsIndexed {
				break
			}
		}

		// 关闭并重新打开数据库
		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			engine.dbs[shard].Close()
			dbPath := engine.initOptions.PersistentStorageFolder + "/" + PersistentStorageFilePrefix + "." + strconv.Itoa(shard)
			db, err := storage.OpenStorage(dbPath)
			if db == nil || err != nil {
				log.Fatal("无法打开数据库", dbPath, ": ", err)
			}
			engine.dbs[shard] = db
		}

		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			go engine.persistentStorageIndexDocumentWorker(shard)
		}
	}

	atomic.AddUint64(&engine.numDocumentsStored, engine.numIndexingRequests)
}

// 将文档加入索引
//
// 输入参数：
// 	docId	标识文档编号，必须唯一
//	data	见DocumentIndexData注释
//
// 注意：
//      1. 这个函数是线程安全的，请尽可能并发调用以提高索引速度
// 	2. 这个函数调用是非同步的，也就是说在函数返回时有可能文档还没有加入索引中，因此
//         如果立刻调用Search可能无法查询到这个文档。强制刷新索引请调用FlushIndex函数。
func (engine *Engine) IndexDocument(docId uint64, data types.DocumentIndexData) {
	engine.internalIndexDocument(docId, data)

	hash := murmur.Murmur3([]byte(fmt.Sprint("%d", docId))) % uint32(engine.initOptions.PersistentStorageShards)
	if engine.initOptions.UsePersistentStorage {
		engine.persistentStorageIndexDocumentChannels[hash] <- persistentStorageIndexDocumentRequest{docId: docId, data: data}
	}
}

func (engine *Engine) internalIndexDocument(docId uint64, data types.DocumentIndexData) {
	if !engine.initialized {
		log.Fatal("必须先初始化引擎")
	}

	atomic.AddUint64(&engine.numIndexingRequests, 1)
	hash := murmur.Murmur3([]byte(fmt.Sprint("%d%s", docId, data.Content)))
	engine.segmenterChannel <- segmenterRequest{
		docId: docId, hash: hash, data: data}
}

// 将文档从索引中删除
//
// 输入参数：
// 	docId	标识文档编号，必须唯一
//
// 注意：这个函数仅从排序器中删除文档，索引器不会发生变化。
func (engine *Engine) RemoveDocument(docId uint64) {
	if !engine.initialized {
		log.Fatal("必须先初始化引擎")
	}

	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		engine.indexerRemoveDocChannels[shard] <- indexerRemoveDocRequest{docId: docId}
		engine.rankerRemoveDocChannels[shard] <- rankerRemoveDocRequest{docId: docId}
	}

	if engine.initOptions.UsePersistentStorage {
		// 从数据库中删除
		hash := murmur.Murmur3([]byte(fmt.Sprint("%d", docId))) % uint32(engine.initOptions.PersistentStorageShards)
		go engine.persistentStorageRemoveDocumentWorker(docId, hash)
	}
}

// 阻塞等待直到所有索引添加完毕
func (engine *Engine) FlushIndex() {
	for {
		runtime.Gosched()
		if engine.numIndexingRequests == engine.numDocumentsIndexed &&
			(!engine.initOptions.UsePersistentStorage ||
				engine.numIndexingRequests == engine.numDocumentsStored) {
			return
		}
	}
}

// 获取文本的分词结果
func (engine *Engine) Tokens(text []byte) (tokens []string) {
	querySegments := engine.segmenter.Segment(text)
	for _, s := range querySegments {
		token := s.Token().Text()
		if !engine.stopTokens.IsStopToken(token) {
			tokens = append(tokens, token)
		}
	}

	return tokens
}

// 查找满足搜索条件的文档，此函数线程安全
func (engine *Engine) Search(request types.SearchRequest) (output types.SearchResponse) {
	if !engine.initialized {
		log.Fatal("必须先初始化引擎")
	}

	var rankOptions types.RankOptions
	if request.RankOptions == nil {
		rankOptions = *engine.initOptions.DefaultRankOptions
	} else {
		rankOptions = *request.RankOptions
	}
	if rankOptions.ScoringCriteria == nil {
		rankOptions.ScoringCriteria = engine.initOptions.DefaultRankOptions.ScoringCriteria
	}

	// 收集关键词
	tokens := []string{}
	if request.Text != "" {
		tokens = engine.Tokens([]byte(request.Text))
	} else {
		for _, t := range request.Tokens {
			tokens = append(tokens, t)
		}
	}

	// 建立排序器返回的通信通道
	rankerReturnChannel := make(
		chan rankerReturnRequest, engine.initOptions.NumShards)

	// 生成查找请求
	lookupRequest := indexerLookupRequest{
		countDocsOnly:       request.CountDocsOnly,
		tokens:              tokens,
		labels:              request.Labels,
		docIds:              request.DocIds,
		options:             rankOptions,
		rankerReturnChannel: rankerReturnChannel,
		orderless:           request.Orderless,
	}

	// 向索引器发送查找请求
	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		engine.indexerLookupChannels[shard] <- lookupRequest
	}

	// 从通信通道读取排序器的输出
	numDocs := 0
	rankOutput := types.ScoredDocuments{}
	timeout := request.Timeout
	isTimeout := false
	if timeout <= 0 {
		// 不设置超时
		for shard := 0; shard < engine.initOptions.NumShards; shard++ {
			rankerOutput := <-rankerReturnChannel
			if !request.CountDocsOnly {
				for _, doc := range rankerOutput.docs {
					rankOutput = append(rankOutput, doc)
				}
			}
			numDocs += rankerOutput.numDocs
		}
	} else {
		// 设置超时
		deadline := time.Now().Add(time.Nanosecond * time.Duration(NumNanosecondsInAMillisecond*request.Timeout))
		for shard := 0; shard < engine.initOptions.NumShards; shard++ {
			select {
			case rankerOutput := <-rankerReturnChannel:
				if !request.CountDocsOnly {
					for _, doc := range rankerOutput.docs {
						rankOutput = append(rankOutput, doc)
					}
				}
				numDocs += rankerOutput.numDocs
			case <-time.After(deadline.Sub(time.Now())):
				isTimeout = true
				break
			}
		}
	}

	// 再排序
	if !request.CountDocsOnly && !request.Orderless {
		if rankOptions.ReverseOrder {
			sort.Sort(sort.Reverse(rankOutput))
		} else {
			sort.Sort(rankOutput)
		}
	}

	// 准备输出
	output.Tokens = tokens
	// 仅当CountDocsOnly为false时才充填output.Docs
	if !request.CountDocsOnly {
		if request.Orderless {
			// 无序状态无需对Offset截断
			output.Docs = rankOutput
		} else {
			var start, end int
			if rankOptions.MaxOutputs == 0 {
				start = utils.MinInt(rankOptions.OutputOffset, len(rankOutput))
				end = len(rankOutput)
			} else {
				start = utils.MinInt(rankOptions.OutputOffset, len(rankOutput))
				end = utils.MinInt(start+rankOptions.MaxOutputs, len(rankOutput))
			}
			output.Docs = rankOutput[start:end]
		}
	}
	output.NumDocs = numDocs
	output.Timeout = isTimeout
	return
}

// 关闭引擎
func (engine *Engine) Close() {
	engine.FlushIndex()
	if engine.initOptions.UsePersistentStorage {
		for _, db := range engine.dbs {
			db.Close()
		}
	}
}

// 从文本hash得到要分配到的shard
func (engine *Engine) getShard(hash uint32) int {
	return int(hash - hash/uint32(engine.initOptions.NumShards)*uint32(engine.initOptions.NumShards))
}
