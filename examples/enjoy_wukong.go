package main

import (
	"log"

	"github.com/huichen/wukong/engine"
	"github.com/huichen/wukong/types"
)

var (
	searcher = engine.Engine{}
)

type Data struct {
	Id      int
	Content string
	Labels  []string
}

func (d *Data) Print() {
	log.Println(d.Id, d.Content, d.Labels)
}

func main() {
	datas := []Data{}

	data0 := Data{Id: 1, Content: "此次百度收购将成中国互联网最大并购", Labels: []string{"百度", "中国"}}
	datas = append(datas, data0)

	data1 := Data{Id: 2, Content: "百度宣布拟全资收购91无线业务", Labels: []string{"百度"}}
	datas = append(datas, data1)

	data2 := Data{Id: 3, Content: "百度是中国最大的搜索引擎", Labels: []string{"百度"}}
	datas = append(datas, data2)

	data3 := Data{Id: 4, Content: "百度在研制无人汽车", Labels: []string{"百度"}}
	datas = append(datas, data3)

	data4 := Data{Id: 5, Content: "BAT是中国互联网三巨头", Labels: []string{"百度"}}
	datas = append(datas, data4)

	// 初始化
	searcher.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../data/dictionary.txt",
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
			//IndexType: types.FrequenciesIndex,
			//IndexType: types.DocIdsIndex,
		},
	})
	defer searcher.Close()

	// 将文档加入索引
	for _, data := range datas {
		searcher.IndexDocument(uint64(data.Id), types.DocumentIndexData{Content: data.Content, Labels: data.Labels}, false)
	}

	// 等待索引刷新完毕
	searcher.FlushIndex()

	// 搜索输出格式见types.SearchResponse结构体
	res := searcher.Search(types.SearchRequest{Text: "百度"})
	log.Println("关键字", res.Tokens, "共有", res.NumDocs, "条搜索结果")
	for i := range res.Docs {
		datas[res.Docs[i].DocId-1].Print()
	}
}
