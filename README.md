悟空全文搜索引擎
======

* [高效索引和搜索](/docs/benchmarking.md)（10M条微博3.6G数据7分钟索引完，2.5毫秒搜索响应时间，每秒可处理1.6K次请求）
* 支持中文分词（使用[sego分词包](https://github.com/huichen/sego)并发分词，速度13MB/秒）
* 支持计算关键词在文本中的[紧邻距离](/docs/token_proximity.md)（token proximity）
* 支持计算[BM25相关度](/docs/bm25.md)
* 支持[自定义评分字段和评分规则](/docs/custom_scoring_criteria.md)
* 支持[在线添加、删除索引](/docs/realtime_indexing.md)
* 可实现[分布式索引和搜索](/docs/distributed_indexing_and_search.md)
* 采用对商业应用友好的[Apache License v2](/license.txt)发布

微博搜索演示 http://soooweibo.com

# 安装/更新

先安装依赖包
```
go get -u github.com/huichen/sego
go get -u github.com/huichen/murmur
```

然后安装悟空引擎
```
go get -u github.com/huichen/wukong
```

需要Go版本至少1.1.1

# 使用

先看一个例子（来自[examples/simplest_example.go](/examples/simplest_example.go)）
```go
package main

import (
	"github.com/huichen/wukong/engine"
	"github.com/huichen/wukong/types"
	"log"
)

var (
	// searcher是协程安全的
	searcher = engine.Engine{}
)

func main() {
	// 初始化
	searcher.Init(types.EngineInitOptions{
		SegmenterDictionaries: "github.com/huichen/wukong/data/dictionary.txt"})

	// 将文档加入索引
	searcher.IndexDocument(0, types.DocumentIndexData{Content: "此次百度收购将成中国互联网最大并购"})
	searcher.IndexDocument(1, types.DocumentIndexData{Content: "百度宣布拟全资收购91无线业务"})
	searcher.IndexDocument(2, types.DocumentIndexData{Content: "百度是中国最大的搜索引擎"})

	// 等待索引刷新完毕
	searcher.FlushIndex()

	// 搜索输出格式见types.SearchResponse结构体
	log.Print(searcher.Search(types.SearchRequest{Text:"百度中国"}))
}
```

是不是很简单！

然后看看一个[入门教程](/docs/codelab.md)，教你用不到200行Go代码实现一个微博搜索网站。

# 其它

* [为什么要有悟空引擎](/docs/why_wukong.md)
* [联系方式](/docs/feedback.md)
