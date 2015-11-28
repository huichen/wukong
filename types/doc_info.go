package types

import (
	"sync"
)

// 文档信息[id]info
type DocInfosShard struct {
	DocInfos     map[uint64]*DocInfo
	NumDocuments uint64 // 这实际上是总文档数的一个近似
	sync.RWMutex
}

type DocInfo struct {
	Fields       interface{}
	TokenLengths float32
}
