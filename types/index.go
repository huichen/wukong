package types

type DocumentIndex struct {
	// 文本的DocId
	DocId uint64

	// 文本的关键词长
	TokenLength float32

	// 加入的索引键
	Keywords []KeywordIndex
}

// 反向索引项，这实际上标注了一个（搜索键，文档）对。
type KeywordIndex struct {
	// 搜索键的UTF-8文本
	Text string

	// 搜索键词频
	Frequency float32

	// 搜索键在文档中的起始字节位置，按照升序排列
	Starts []int
}

// 索引器返回结果
type IndexedDocument struct {
	DocId uint64

	// BM25，仅当索引类型为FrequenciesIndex或者LocationsIndex时返回有效值
	BM25 float32

	// 关键词在文档中的紧邻距离，紧邻距离的含义见computeTokenProximity的注释。
	// 仅当索引类型为LocationsIndex时返回有效值。
	TokenProximity int32

	// 紧邻距离计算得到的关键词位置，和Lookup函数输入tokens的长度一样且一一对应。
	// 仅当索引类型为LocationsIndex时返回有效值。
	TokenSnippetLocations []int

	// 关键词在文本中的具体位置。
	// 仅当索引类型为LocationsIndex时返回有效值。
	TokenLocations [][]int
}
