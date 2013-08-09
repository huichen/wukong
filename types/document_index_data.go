package types

type DocumentIndexData struct {
	// 文档全文，用于生成待索引的关键词
	Content string

	// 文档标签，比如文档的类别属性等，这些标签并不出现在文档文本中
	Labels []string

	// 文档的评分字段，可以接纳任何类型的结构体
	Fields interface{}
}
