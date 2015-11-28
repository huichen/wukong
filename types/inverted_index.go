package types

import (
	"sync"
)

// 反向索引表([关键词]反向索引表)
type InvertedIndexShard struct {
	InvertedIndex    map[string]*KeywordIndices
	TotalTokenLength float32 //总关键词数
	sync.RWMutex
}

// 反向索引表的一行，收集了一个搜索键出现的所有文档，按照DocId从小到大排序。
type KeywordIndices struct {
	// 下面的切片是否为空，取决于初始化时IndexType的值
	DocIds      []uint64  // 全部类型都有
	Frequencies []float32 // IndexType == FrequenciesIndex
	Locations   [][]int   // IndexType == LocationsIndex
}
