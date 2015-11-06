package engine

import (
	"fmt"
	"github.com/cznic/kv"
	"github.com/henrylee2cn/wukong/core"
	"github.com/henrylee2cn/wukong/types"
	"github.com/henrylee2cn/wukong/utils"
	// "github.com/huichen/murmur"
	"github.com/huichen/sego"
	"log"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
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

	indexers   map[uint64]*core.Indexer
	rankers    map[uint64]*core.Ranker
	segmenter  sego.Segmenter
	stopTokens StopTokens
	dbs        map[uint64]*kv.DB

	// 建立索引器使用的通信通道
	segmenterChannel               chan segmenterRequest
	indexerAddDocumentChannels     map[uint64]chan indexerAddDocumentRequest
	rankerAddScoringFieldsChannels map[uint64]chan rankerAddScoringFieldsRequest

	// 建立排序器使用的通信通道
	indexerLookupChannels             map[uint64]chan indexerLookupRequest
	rankerRankChannels                map[uint64]chan rankerRankRequest
	rankerRemoveScoringFieldsChannels map[uint64]chan rankerRemoveScoringFieldsRequest

	// 建立持久存储使用的通信通道
	persistentStorageIndexDocumentChannels map[uint64]chan persistentStorageIndexDocumentRequest
	persistentStorageInitChannel           chan bool

	// 动态添加工作协程的锁
	sync.Mutex
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

	// 载入分词器词典
	engine.segmenter.LoadDictionary(options.SegmenterDictionaries)

	// 初始化停用词
	engine.stopTokens.Init(options.StopTokenFile)

	// 初始化分词器通道
	engine.segmenterChannel = make(chan segmenterRequest, options.NumSegmenterThreads)

	// 启动分词器
	for iThread := 0; iThread < options.NumSegmenterThreads; iThread++ {
		go engine.segmenterWorker()
	}

	// 分片数量
	numShards := len(options.Shards)

	// 初始化索引器和排序器
	engine.indexers = make(map[uint64]*core.Indexer, numShards)
	engine.rankers = make(map[uint64]*core.Ranker, numShards)

	// 初始化索引器通道
	engine.indexerAddDocumentChannels = make(map[uint64]chan indexerAddDocumentRequest, numShards)
	engine.indexerLookupChannels = make(map[uint64]chan indexerLookupRequest, numShards)

	// 初始化排序器通道
	engine.rankerAddScoringFieldsChannels = make(map[uint64]chan rankerAddScoringFieldsRequest, numShards)
	engine.rankerRankChannels = make(map[uint64]chan rankerRankRequest, numShards)
	engine.rankerRemoveScoringFieldsChannels = make(map[uint64]chan rankerRemoveScoringFieldsRequest, numShards)

	// 初始化持久化存储通道
	if options.UsePersistentStorage {
		engine.persistentStorageIndexDocumentChannels = make(map[uint64]chan persistentStorageIndexDocumentRequest, numShards)

		for _, shard := range options.Shards {
			engine.persistentStorageIndexDocumentChannels[shard] = make(chan persistentStorageIndexDocumentRequest)
		}
		engine.persistentStorageInitChannel = make(chan bool, numShards)

		err := os.MkdirAll(options.PersistentStorageFolder, 0700)
		if err != nil {
			log.Fatal("无法创建目录", options.PersistentStorageFolder)
		}

		// 打开或者创建数据库
		engine.dbs = make(map[uint64]*kv.DB, numShards)
	}

	if numShards == 0 {
		return
	}

	for _, shard := range options.Shards {
		// 索引器和排序器
		engine.indexers[shard] = new(core.Indexer)
		engine.indexers[shard].Init(*options.IndexerInitOptions)

		engine.rankers[shard] = new(core.Ranker)
		engine.rankers[shard].Init()

		// 排序器通道
		engine.indexerAddDocumentChannels[shard] = make(chan indexerAddDocumentRequest, options.IndexerBufferLength)
		engine.indexerLookupChannels[shard] = make(chan indexerLookupRequest, options.IndexerBufferLength)

		// 持久化存储通道
		engine.rankerAddScoringFieldsChannels[shard] = make(chan rankerAddScoringFieldsRequest, options.RankerBufferLength)
		engine.rankerRankChannels[shard] = make(chan rankerRankRequest, options.RankerBufferLength)
		engine.rankerRemoveScoringFieldsChannels[shard] = make(chan rankerRemoveScoringFieldsRequest, options.RankerBufferLength)

		// 启动索引器和排序器
		go engine.indexerAddDocumentWorker(shard)
		go engine.rankerAddScoringFieldsWorker(shard)
		go engine.rankerRemoveScoringFieldsWorker(shard)

		for i := 0; i < options.NumIndexerThreadsPerShard; i++ {
			go engine.indexerLookupWorker(shard)
		}
		for i := 0; i < options.NumRankerThreadsPerShard; i++ {
			go engine.rankerRankWorker(shard)
		}
	}

	// 启动索引信息持久化存储工作协程
	if options.UsePersistentStorage {
		for _, shard := range options.Shards {
			dbPath := options.PersistentStorageFolder + "/" + PersistentStorageFilePrefix + "." + strconv.FormatUint(shard, 10)
			db, err := utils.OpenOrCreateKv(dbPath, &kv.Options{})
			if db == nil || err != nil {
				log.Fatal("无法打开数据库", dbPath, ": ", err)
			}
			engine.dbs[shard] = db
		}

		// 从数据库中恢复
		for _, shard := range options.Shards {
			go engine.persistentStorageInitWorker(shard)
		}

		// 等待恢复完成
		for range options.Shards {
			<-engine.persistentStorageInitChannel
		}

		for {
			runtime.Gosched()
			if engine.numIndexingRequests == engine.numDocumentsIndexed {
				break
			}
		}

		for _, shard := range options.Shards {
			go engine.persistentStorageIndexDocumentWorker(shard)
		}
	}
}

// 在执行IndexDocument方法的过程中对documentIndexChan写入的数据进行计数
var documentIndexChanCount = map[chan *types.DocumentIndex]*Count{}

type Count struct {
	Num int
	sync.Mutex
}

// 将文档加入索引
//
// 输入参数：
// 	docId	标识文档编号，必须唯一
//	data	见DocumentIndexData注释
//
// 注意：
// 1. 这个函数是线程安全的，请尽可能并发调用以提高索引速度。
// 2. 这个函数调用是非同步的，也就是说在函数返回时有可能文档还没有加入索引中，因此
//    如果立刻调用Search可能无法查询到这个文档。强制刷新索引请调用FlushIndex函数。
// 3. 可选参数documentIndexChan不为空时，将在分词步骤写入关键词及其词频信息，此时程序并进入等待状态，
//    只有当其通道内所有数据被外部读出来，程序才能继续运行、建立索引。
//    这样设计的目的在于让外部程序可以在索引建立之前，处理一些与documentIndexChan相关的事务。
//    注意：documentIndexChan必须为异步通道，其容量设置为欲传入IndexDocument方法的总次数！
func (engine *Engine) IndexDocument(docId string, data types.DocumentIndexData, shard uint64, documentIndexChan ...chan *types.DocumentIndex) {
	// 检验指定的shard是否存在
	if err := engine.checkShard(shard); err != nil {
		// 动态添加工作协程及数据库
		engine.appendRoutine(shard)
	}

	if !engine.initialized {
		log.Fatal("必须先初始化引擎")
	}

	atomic.AddUint64(&engine.numIndexingRequests, 1)

	var dichan chan *types.DocumentIndex
	if len(documentIndexChan) > 0 {
		dichan = documentIndexChan[0]
		if _, ok := documentIndexChanCount[dichan]; !ok {
			documentIndexChanCount[dichan] = new(Count)
		}
	}

	engine.segmenterChannel <- segmenterRequest{docId: docId, shard: shard, data: data, documentIndexChan: dichan}
}

// 将文档从索引中删除
//
// 输入参数：
// 	docId	标识文档编号，必须唯一
//
// 注意：这个函数仅从排序器中删除文档的自定义评分字段，索引器不会发生变化。所以
// 你的自定义评分字段必须能够区别评分字段为nil的情况，并将其从排序结果中删除。
func (engine *Engine) RemoveDocument(shard uint64, docId string) {
	if !engine.initialized {
		log.Fatal("必须先初始化引擎")
	}

	engine.rankerRemoveScoringFieldsChannels[shard] <- rankerRemoveScoringFieldsRequest{docId: docId}

	if engine.initOptions.UsePersistentStorage {
		// 从数据库中删除
		go engine.persistentStorageRemoveDocumentWorker(shard, docId)
	}
}

// 阻塞等待直到所有索引添加完毕
func (engine *Engine) FlushIndex() {
	for {
		runtime.Gosched()
		if engine.numIndexingRequests == engine.numDocumentsIndexed &&
			(!engine.initOptions.UsePersistentStorage || engine.numIndexingRequests == engine.numDocumentsStored) {
			return
		}
	}
}

// 查找满足搜索条件的文档，此函数线程安全
func (engine *Engine) Search(request types.SearchRequest) (output types.SearchResponse) {
	if !engine.initialized {
		log.Fatal("必须先初始化引擎")
	}
	// 处理请求指定的shard
	if len(request.Shards) == 0 {
		request.Shards = engine.initOptions.Shards
	} else if err := engine.checkShard(request.Shards...); err != nil {
		log.Printf("%v", err)
		// 指定搜范围有误时返回
		return types.SearchResponse{}
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
		querySegments := engine.segmenter.Segment([]byte(request.Text))
		for _, s := range querySegments {
			token := s.Token().Text()
			if !engine.stopTokens.IsStopToken(token) {
				tokens = append(tokens, s.Token().Text())
			}
		}
	} else {
		for _, t := range request.Tokens {
			tokens = append(tokens, t)
		}
	}

	// 建立排序器返回的通信通道
	rankerReturnChannel := make(chan rankerReturnRequest, len(request.Shards))

	// 生成查找请求
	lookupRequest := indexerLookupRequest{
		tokens:              tokens,
		labels:              request.Labels,
		docIds:              request.DocIds,
		options:             rankOptions,
		rankerReturnChannel: rankerReturnChannel,
	}

	// 向索引器发送查找请求
	for _, shard := range request.Shards {
		engine.indexerLookupChannels[shard] <- lookupRequest
	}

	// 从通信通道读取排序器的输出
	rankOutput := types.ScoredDocuments{}
	timeout := request.Timeout
	isTimeout := false
	if timeout <= 0 {
		// 不设置超时
		for range request.Shards {
			rankerOutput := <-rankerReturnChannel
			for _, doc := range rankerOutput.docs {
				rankOutput = append(rankOutput, doc)
			}
		}
	} else {
		// 设置超时
		boom := time.After(time.Nanosecond * time.Duration(NumNanosecondsInAMillisecond*request.Timeout))
		for range request.Shards {
			select {
			case rankerOutput := <-rankerReturnChannel:
				for _, doc := range rankerOutput.docs {
					rankOutput = append(rankOutput, doc)
				}
			case <-boom:
				isTimeout = true
				break
			}
		}
	}

	// 再排序
	if rankOptions.ReverseOrder {
		sort.Sort(sort.Reverse(rankOutput))
	} else {
		sort.Sort(rankOutput)
	}

	// 准备输出
	output.Tokens = tokens
	output.Total = len(rankOutput)
	var start, end int
	if rankOptions.MaxOutputs == 0 {
		start = utils.MinInt(rankOptions.OutputOffset, output.Total)
		end = len(rankOutput)
	} else {
		start = utils.MinInt(rankOptions.OutputOffset, output.Total)
		end = utils.MinInt(start+rankOptions.MaxOutputs, output.Total)
	}
	output.Docs = rankOutput[start:end]
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

// 动态追加工作协程及数据库
func (engine *Engine) appendRoutine(shard uint64) {
	engine.Mutex.Lock()
	defer engine.Mutex.Unlock()
	// 检验指定的shard是否存在
	if err := engine.checkShard(shard); err == nil {
		return
	}

	engine.initOptions.Shards = append(engine.initOptions.Shards, shard)

	// 初始化索引器和排序器
	engine.indexers[shard] = new(core.Indexer)
	engine.indexers[shard].Init(*engine.initOptions.IndexerInitOptions)
	engine.rankers[shard] = new(core.Ranker)
	engine.rankers[shard].Init()

	// 初始化索引器通道
	engine.indexerAddDocumentChannels[shard] = make(chan indexerAddDocumentRequest, engine.initOptions.IndexerBufferLength)
	engine.indexerLookupChannels[shard] = make(chan indexerLookupRequest, engine.initOptions.IndexerBufferLength)

	// 初始化排序器通道
	engine.rankerAddScoringFieldsChannels[shard] = make(chan rankerAddScoringFieldsRequest, engine.initOptions.RankerBufferLength)
	engine.rankerRankChannels[shard] = make(chan rankerRankRequest, engine.initOptions.RankerBufferLength)
	engine.rankerRemoveScoringFieldsChannels[shard] = make(chan rankerRemoveScoringFieldsRequest, engine.initOptions.RankerBufferLength)

	// 初始化持久化存储通道
	if engine.initOptions.UsePersistentStorage {
		engine.persistentStorageIndexDocumentChannels[shard] = make(chan persistentStorageIndexDocumentRequest)
		engine.persistentStorageInitChannel = make(chan bool, 1)
	}

	// 启动索引器和排序器
	go engine.indexerAddDocumentWorker(shard)
	go engine.rankerAddScoringFieldsWorker(shard)
	go engine.rankerRemoveScoringFieldsWorker(shard)

	for i := 0; i < engine.initOptions.NumIndexerThreadsPerShard; i++ {
		go engine.indexerLookupWorker(shard)
	}
	for i := 0; i < engine.initOptions.NumRankerThreadsPerShard; i++ {
		go engine.rankerRankWorker(shard)
	}

	// 启动索引信息持久化存储工作协程
	if engine.initOptions.UsePersistentStorage {
		// 打开或者创建数据库
		dbPath := engine.initOptions.PersistentStorageFolder + "/" + PersistentStorageFilePrefix + "." + strconv.FormatUint(shard, 10)
		db, err := utils.OpenOrCreateKv(dbPath, &kv.Options{})
		if db == nil || err != nil {
			log.Fatal("无法打开数据库", dbPath, ": ", err)
		}
		engine.dbs[shard] = db

		// 从数据库中恢复
		go engine.persistentStorageInitWorker(shard)

		// 等待恢复完成
		<-engine.persistentStorageInitChannel

		for {
			runtime.Gosched()
			if engine.numIndexingRequests == engine.numDocumentsIndexed {
				break
			}
		}

		go engine.persistentStorageIndexDocumentWorker(shard)
	}
}

// 检查请求指定的shard是否已存在
func (engine *Engine) checkShard(shard ...uint64) error {
	if len(shard) == 0 {
		for _, i := range engine.initOptions.Shards {
			shard = append(shard, i)
		}
		return nil
	}
	var ok = false
	for _, m := range shard {
		ok = false
		for _, n := range engine.initOptions.Shards {
			if m == n {
				ok = true
				break
			}
		}
		if !ok {
			goto err
		}
	}

	return nil

err:
	return fmt.Errorf("指定shard %v 不存在，有效shard范围为 %v", shard, engine.initOptions.Shards)
}
