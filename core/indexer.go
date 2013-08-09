package core

import (
	"github.com/huichen/wukong/types"
	"log"
	"math"
	"sync"
)

// 索引器
type Indexer struct {
	// 从搜索键到文档列表的反向索引
	// 加了读写锁以保证读写安全
	tableLock struct {
		sync.RWMutex
		table map[string]*KeywordIndices
	}

	initOptions types.IndexerInitOptions
	initialized bool

	// 这实际上是总文档数的一个近似
	numDocuments uint64

	// 所有被索引文本的总关键词数
	totalTokenLength float32

	// 每个文档的关键词长度
	docTokenLengths map[uint64]float32
}

// 反向索引表的一行，收集了一个搜索键出现的所有文档，按照DocId从小到大排序。
type KeywordIndices struct {
	// 下面的切片是否为空，取决于初始化时IndexType的值
	docIds      []uint64  // 全部类型都有
	frequencies []float32 // IndexType == FrequenciesIndex
	locations   [][]int   // IndexType == LocationsIndex
}

// 初始化索引器
func (indexer *Indexer) Init(options types.IndexerInitOptions) {
	if indexer.initialized == true {
		log.Fatal("索引器不能初始化两次")
	}
	indexer.initialized = true

	indexer.tableLock.table = make(map[string]*KeywordIndices)
	indexer.initOptions = options
	indexer.docTokenLengths = make(map[uint64]float32)
}

// 向反向索引表中加入一个文档
func (indexer *Indexer) AddDocument(document *types.DocumentIndex) {
	if indexer.initialized == false {
		log.Fatal("索引器尚未初始化")
	}

	indexer.tableLock.Lock()
	defer indexer.tableLock.Unlock()

	// 更新文档关键词总长度
	if document.TokenLength != 0 {
		originalLength, found := indexer.docTokenLengths[document.DocId]
		indexer.docTokenLengths[document.DocId] = float32(document.TokenLength)
		if found {
			indexer.totalTokenLength += document.TokenLength - originalLength
		} else {
			indexer.totalTokenLength += document.TokenLength
		}
	}

	docIdIsNew := true
	for _, keyword := range document.Keywords {
		indices, foundKeyword := indexer.tableLock.table[keyword.Text]
		if !foundKeyword {
			// 如果没找到该搜索键则加入
			ti := KeywordIndices{}
			switch indexer.initOptions.IndexType {
			case types.LocationsIndex:
				ti.locations = [][]int{keyword.Starts}
			case types.FrequenciesIndex:
				ti.frequencies = []float32{keyword.Frequency}
			}
			ti.docIds = []uint64{document.DocId}
			indexer.tableLock.table[keyword.Text] = &ti
			continue
		}

		// 查找应该插入的位置
		position, found := indexer.searchIndex(
			indices, 0, indexer.getIndexLength(indices)-1, document.DocId)
		if found {
			docIdIsNew = false

			// 覆盖已有的索引项
			switch indexer.initOptions.IndexType {
			case types.LocationsIndex:
				indices.locations[position] = keyword.Starts
			case types.FrequenciesIndex:
				indices.frequencies[position] = keyword.Frequency
			}
			continue
		}

		// 当索引不存在时，插入新索引项
		switch indexer.initOptions.IndexType {
		case types.LocationsIndex:
			indices.locations = append(indices.locations, []int{})
			copy(indices.locations[position+1:], indices.locations[position:])
			indices.locations[position] = keyword.Starts
		case types.FrequenciesIndex:
			indices.frequencies = append(indices.frequencies, float32(0))
			copy(indices.frequencies[position+1:], indices.frequencies[position:])
			indices.frequencies[position] = keyword.Frequency
		}
		indices.docIds = append(indices.docIds, 0)
		copy(indices.docIds[position+1:], indices.docIds[position:])
		indices.docIds[position] = document.DocId
	}

	// 更新文章总数
	if docIdIsNew {
		indexer.numDocuments++
	}
}

// 查找包含全部搜索键(AND操作)的文档
// 当docIds不为nil时仅从docIds指定的文档中查找
func (indexer *Indexer) Lookup(
	tokens []string, labels []string, docIds *map[uint64]bool) (docs []types.IndexedDocument) {
	if indexer.initialized == false {
		log.Fatal("索引器尚未初始化")
	}

	if indexer.numDocuments == 0 {
		return
	}

	// 合并关键词和标签为搜索键
	keywords := make([]string, len(tokens)+len(labels))
	copy(keywords, tokens)
	copy(keywords[len(tokens):], labels)

	indexer.tableLock.RLock()
	defer indexer.tableLock.RUnlock()
	table := make([]*KeywordIndices, len(keywords))
	for i, keyword := range keywords {
		indices, found := indexer.tableLock.table[keyword]
		if !found {
			// 当反向索引表中无此搜索键时直接返回
			return
		} else {
			// 否则加入反向表中
			table[i] = indices
		}
	}

	// 当没有找到时直接返回
	if len(table) == 0 {
		return
	}

	// 归并查找各个搜索键出现文档的交集
	// 从后向前查保证先输出DocId较大文档
	indexPointers := make([]int, len(table))
	for iTable := 0; iTable < len(table); iTable++ {
		indexPointers[iTable] = indexer.getIndexLength(table[iTable]) - 1
	}
	// 平均文本关键词长度，用于计算BM25
	avgDocLength := indexer.totalTokenLength / float32(indexer.numDocuments)
	for ; indexPointers[0] >= 0; indexPointers[0]-- {
		// 以第一个搜索键出现的文档作为基准，并遍历其他搜索键搜索同一文档
		baseDocId := indexer.getDocId(table[0], indexPointers[0])

		if docIds != nil {
			_, found := (*docIds)[baseDocId]
			if !found {
				continue
			}
		}
		iTable := 1
		found := true
		for ; iTable < len(table); iTable++ {
			// 二分法比简单的顺序归并效率高，也有更高效率的算法，
			// 但顺序归并也许是更好的选择，考虑到将来需要用链表重新实现
			// 以避免反向表添加新文档时的写锁。
			// TODO: 进一步研究不同求交集算法的速度和可扩展性。
			position, foundBaseDocId := indexer.searchIndex(table[iTable],
				0, indexPointers[iTable], baseDocId)
			if foundBaseDocId {
				indexPointers[iTable] = position
			} else {
				if position == 0 {
					// 该搜索键中所有的文档ID都比baseDocId大，因此已经没有
					// 继续查找的必要。
					return
				} else {
					// 继续下一indexPointers[0]的查找
					indexPointers[iTable] = position - 1
					found = false
					break
				}
			}
		}

		if found {
			indexedDoc := types.IndexedDocument{}

			// 当为LocationsIndex时计算关键词紧邻距离
			if indexer.initOptions.IndexType == types.LocationsIndex {
				// 计算有多少关键词是带有距离信息的
				numTokensWithLocations := 0
				for i, t := range table[:len(tokens)] {
					if len(t.locations[indexPointers[i]]) > 0 {
						numTokensWithLocations++
					}
				}
				if numTokensWithLocations != len(tokens) {
					docs = append(docs, types.IndexedDocument{
						DocId: baseDocId,
					})
					break
				}

				// 计算搜索键在文档中的紧邻距离
				tokenLocations := make([]int, len(tokens))
				tokenProximity := computeTokenProximity(
					table[:len(tokens)], indexPointers, tokens, &tokenLocations)
				indexedDoc.TokenProximity = int32(tokenProximity)
				indexedDoc.TokenSnippetLocations = tokenLocations

				// 添加TokenLocations
				indexedDoc.TokenLocations = make([][]int, len(tokens))
				for i, t := range table[:len(tokens)] {
					indexedDoc.TokenLocations[i] = t.locations[indexPointers[i]]
				}
			}

			// 当为LocationsIndex或者FrequenciesIndex时计算BM25
			if indexer.initOptions.IndexType == types.LocationsIndex ||
				indexer.initOptions.IndexType == types.FrequenciesIndex {
				bm25 := float32(0)
				d := indexer.docTokenLengths[baseDocId]
				for i, t := range table[:len(tokens)] {
					var frequency float32
					if indexer.initOptions.IndexType == types.LocationsIndex {
						frequency = float32(len(t.locations[indexPointers[i]]))
					} else {
						frequency = t.frequencies[indexPointers[i]]
					}

					// 计算BM25
					if len(t.docIds) > 0 && frequency > 0 && indexer.initOptions.BM25Parameters != nil && avgDocLength != 0 {
						// 带平滑的idf
						idf := float32(math.Log2(float64(indexer.numDocuments)/float64(len(t.docIds)) + 1))
						k1 := indexer.initOptions.BM25Parameters.K1
						b := indexer.initOptions.BM25Parameters.B
						bm25 += idf * frequency * (k1 + 1) / (frequency + k1*(1-b+b*d/avgDocLength))
					}
				}
				indexedDoc.BM25 = float32(bm25)
			}

			indexedDoc.DocId = baseDocId
			docs = append(docs, indexedDoc)
		}
	}
	return
}

// 二分法查找indices中某文档的索引项
// 第一个返回参数为找到的位置或需要插入的位置
// 第二个返回参数标明是否找到
func (indexer *Indexer) searchIndex(
	indices *KeywordIndices, start int, end int, docId uint64) (int, bool) {
	// 特殊情况
	if indexer.getIndexLength(indices) == start {
		return start, false
	}
	if docId < indexer.getDocId(indices, start) {
		return start, false
	} else if docId == indexer.getDocId(indices, start) {
		return start, true
	}
	if docId > indexer.getDocId(indices, end) {
		return end + 1, false
	} else if docId == indexer.getDocId(indices, end) {
		return end, true
	}

	// 二分
	var middle int
	for end-start > 1 {
		middle = (start + end) / 2
		if docId == indexer.getDocId(indices, middle) {
			return middle, true
		} else if docId > indexer.getDocId(indices, middle) {
			start = middle
		} else {
			end = middle
		}
	}
	return end, false
}

// 计算搜索键在文本中的紧邻距离
//
// 假定第i个搜索键首字节出现在文本中的位置为P_i，长度L_i
// 紧邻距离计算公式为
//
// 	ArgMin(Sum(Abs(P_(i+1) - P_i - L_i)))
//
// 具体计算过程为先取定一个P_1，计算所有P_2的可能值中令Abs(P_2 - P_1 - L1)最小,
// 然后固定P2后依照同样的方法选择P3，P4，等等。遍历所有可能的P_1得到最小的紧邻距离。
//
// 选定的P_i通过tokenLocations参数传回。
func computeTokenProximity(
	table []*KeywordIndices,
	indexPointers []int,
	tokens []string,
	tokenLocations *[]int) int {
	minTokenProximity := -1
	currentLocations := make([]int, len(tokens))
	for _, primaryLocation := range table[0].locations[indexPointers[0]] {
		tokenProximity := 0
		previousLocation := primaryLocation + len(tokens[0]) // P_1 + L_1
		for iToken := 1; iToken < len(tokens); iToken++ {
			locations := table[iToken].locations[indexPointers[iToken]]

			// 寻找 P_i + L_i 后面最近的那个 P_(i+1)
			for currentLocations[iToken] = 0; currentLocations[iToken] < len(locations) &&
				locations[currentLocations[iToken]] < previousLocation; currentLocations[iToken]++ {
			}

			if currentLocations[iToken] == 0 {
				// 找到的P_(i+1)是搜索键i+1出现的第一个位置
				tokenProximity += locations[currentLocations[iToken]] -
					previousLocation
			} else if currentLocations[iToken] == len(locations) {
				// 否则当搜索键i+1出现的最后一个位置仍然小于P_i + L_i
				tokenProximity += previousLocation -
					locations[currentLocations[iToken]-1]
				currentLocations[iToken]--
			} else {
				rightProximity := locations[currentLocations[iToken]] - previousLocation
				leftProximity := previousLocation - locations[currentLocations[iToken]-1]
				if rightProximity > leftProximity {
					// 左侧更接近
					tokenProximity += leftProximity
					currentLocations[iToken]--
				} else {
					// 右侧更接近
					tokenProximity += rightProximity
				}
			}

			// 更新 P_(i+1) + L_(i+1)
			previousLocation = locations[currentLocations[iToken]] + len(tokens[iToken])
		}

		// 更新搜索键紧邻距离
		if minTokenProximity < 0 || minTokenProximity > tokenProximity {
			minTokenProximity = tokenProximity
			(*tokenLocations)[0] = primaryLocation
			for iToken := 1; iToken < len(tokens); iToken++ {
				(*tokenLocations)[iToken] = table[iToken].locations[indexPointers[iToken]][currentLocations[iToken]]
			}
		}
	}

	return minTokenProximity
}

// 从KeywordIndices中得到第i个文档的DocId
func (indexer *Indexer) getDocId(ti *KeywordIndices, i int) uint64 {
	return ti.docIds[i]
}

// 得到KeywordIndices中文档总数
func (indexer *Indexer) getIndexLength(ti *KeywordIndices) int {
	return len(ti.docIds)
}
