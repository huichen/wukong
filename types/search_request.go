package types

type SearchRequest struct {
	// 搜索的短语（必须是UTF-8格式），会被分词
	// 当值为空字符串时关键词会从下面的Tokens读入
	Text string

	// 关键词（必须是UTF-8格式），当Text不为空时优先使用Text
	// 通常你不需要自己指定关键词，除非你运行自己的分词程序
	Tokens []string

	// 文档标签（必须是UTF-8格式），标签不存在文档文本中，但也属于搜索键的一种
	Labels []string

	// 当不为空时，仅从这些文档中搜索
	DocIds []uint64

	// 排序选项
	RankOptions *RankOptions

	// 超时，单位毫秒（千分之一秒）。此值小于等于零时不设超时。
	// 搜索超时的情况下仍有可能返回部分排序结果。
	Timeout int
}

type RankOptions struct {
	// 文档的评分规则，值为nil时使用Engine初始化时设定的规则
	ScoringCriteria ScoringCriteria

	// 默认情况下（ReverseOrder=false）按照分数从大到小排序，否则从小到大排序
	ReverseOrder bool

	// 从第几条结果开始输出
	OutputOffset int

	// 最大输出的搜索结果数，为0时无限制
	MaxOutputs int
}
