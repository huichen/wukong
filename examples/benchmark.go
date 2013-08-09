// 悟空性能测试
package main

import (
	"bufio"
	"flag"
	"github.com/huichen/wukong/engine"
	"github.com/huichen/wukong/types"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

const (
	numRepeatQuery = 1000
)

var (
	weibo_data = flag.String(
		"weibo_data",
		"../testdata/weibo_data.txt",
		"微博数据")
	queries = flag.String(
		"queries",
		"女人母亲,你好中国,网络草根,热门微博,红十字会,"+
			"鳄鱼表演,星座歧视,chinajoy,高帅富,假期计划",
		"待搜索的关键词")
	dictionaries = flag.String(
		"dictionaries",
		"../data/dictionary.txt",
		"分词字典文件")
	stop_token_file = flag.String(
		"stop_token_file",
		"../data/stop_tokens.txt",
		"停用词文件")
	cpuprofile      = flag.String("cpuprofile", "", "处理器profile文件")
	memprofile      = flag.String("memprofile", "", "内存profile文件")
	num_repeat_text = flag.Int("num_repeat_text", 10, "文本重复加入多少次")
	index_type      = flag.Int("index_type", types.DocIdsIndex, "索引类型")

	searcher = engine.Engine{}
	options  = types.RankOptions{
		OutputOffset: 0,
		MaxOutputs:   100,
	}
	searchQueries = []string{}

	NumShards       = 2
	numQueryThreads = runtime.NumCPU() / NumShards
)

func main() {
	// 解析命令行参数
	flag.Parse()
	searchQueries = strings.Split(*queries, ",")
	log.Printf("待搜索的关键词为\"%s\"", searchQueries)

	// 初始化
	searcher.Init(types.EngineInitOptions{
		SegmenterDictionaries: *dictionaries,
		StopTokenFile:         *stop_token_file,
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: *index_type,
		},
		NumShards:          NumShards,
		DefaultRankOptions: &options,
	})

	// 打开将要搜索的文件
	file, err := os.Open(*weibo_data)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// 逐行读入
	log.Printf("读入文本 %s", *weibo_data)
	scanner := bufio.NewScanner(file)
	lines := []string{}
	size := 0
	for scanner.Scan() {
		var text string
		data := strings.Split(scanner.Text(), "||||")
		if len(data) != 10 {
			continue
		}
		text = data[9]
		if text != "" {
			size += len(text) * (*num_repeat_text)
			lines = append(lines, text)
		}
	}
	log.Print("文件行数", len(lines))

	// 记录时间
	t0 := time.Now()

	// 建索引
	log.Print("建索引 ... ")
	docId := uint64(1)
	for i := 0; i < *num_repeat_text; i++ {
		for _, line := range lines {
			searcher.IndexDocument(docId, types.DocumentIndexData{
				Content: line})
			docId++
			if docId-docId/1000000*1000000 == 0 {
				log.Printf("已索引%d百万文档", docId/1000000)
				runtime.GC()
			}
		}
	}
	searcher.FlushIndex()
	log.Print("加入的索引总数", searcher.NumTokenIndexAdded())

	// 记录时间
	t1 := time.Now()
	log.Printf("建立索引花费时间 %v", t1.Sub(t0))
	log.Printf("建立索引速度每秒添加 %f 百万个索引",
		float64(searcher.NumTokenIndexAdded())/t1.Sub(t0).Seconds()/(1000000))

	// 写入内存profile文件
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		defer f.Close()
	}

	// 记录时间
	t2 := time.Now()

	// 打开处理器profile文件
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	done := make(chan bool)
	for iThread := 0; iThread < numQueryThreads; iThread++ {
		go search(done)
	}
	for iThread := 0; iThread < numQueryThreads; iThread++ {
		<-done
	}

	// 停止处理器profile
	if *cpuprofile != "" {
		defer pprof.StopCPUProfile()
	}

	// 记录时间并计算分词速度
	t3 := time.Now()
	log.Printf("搜索平均响应时间 %v 毫秒",
		t3.Sub(t2).Seconds()*1000/float64(numRepeatQuery*len(searchQueries)))
	log.Printf("搜索吞吐量每秒 %v 次查询",
		float64(numRepeatQuery*numQueryThreads*len(searchQueries))/
			t3.Sub(t2).Seconds())
}

func search(ch chan bool) {
	for i := 0; i < numRepeatQuery; i++ {
		for _, query := range searchQueries {
			searcher.Search(types.SearchRequest{Text: query})
		}
	}
	ch <- true
}
