package core

import (
	"testing"

	"github.com/huichen/wukong/engine"
	"github.com/huichen/wukong/types"
	"github.com/huichen/wukong/utils"
)

func TestAddKeywords(t *testing.T) {
	var indexer Indexer
	indexer.Init(types.IndexerInitOptions{IndexType: types.LocationsIndex})
	indexer.AddDocument(&types.DocumentIndex{
		DocId:    1,
		Keywords: []types.KeywordIndex{{"token1", 0, []int{}}},
	})
	indexer.AddDocument(&types.DocumentIndex{
		DocId:    7,
		Keywords: []types.KeywordIndex{{"token1", 0, []int{}}},
	})
	indexer.AddDocument(&types.DocumentIndex{
		DocId:    2,
		Keywords: []types.KeywordIndex{{"token1", 0, []int{}}},
	})
	indexer.AddDocument(&types.DocumentIndex{
		DocId:    3,
		Keywords: []types.KeywordIndex{{"token2", 0, []int{}}},
	})
	indexer.AddDocument(&types.DocumentIndex{
		DocId:    1,
		Keywords: []types.KeywordIndex{{"token1", 0, []int{}}},
	})
	indexer.AddDocument(&types.DocumentIndex{
		DocId:    1,
		Keywords: []types.KeywordIndex{{"token2", 0, []int{}}},
	})
	indexer.AddDocument(&types.DocumentIndex{
		DocId:    2,
		Keywords: []types.KeywordIndex{{"token2", 0, []int{}}},
	})
	indexer.AddDocument(&types.DocumentIndex{
		DocId:    0,
		Keywords: []types.KeywordIndex{{"token2", 0, []int{}}},
	})

	utils.Expect(t, "1 2 7 ", indicesToString(&indexer, "token1"))
	utils.Expect(t, "0 1 2 3 ", indicesToString(&indexer, "token2"))
}

func TestLookup(t *testing.T) {
	var indexer Indexer
	indexer.Init(types.IndexerInitOptions{IndexType: types.LocationsIndex})
	// doc0 = "token2 token3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 0,
		Keywords: []types.KeywordIndex{
			{"token2", 0, []int{0}},
			{"token3", 0, []int{7}},
		},
	})
	// doc1 = "token1 token2 token3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 1,
		Keywords: []types.KeywordIndex{
			{"token1", 0, []int{0}},
			{"token2", 0, []int{7}},
			{"token3", 0, []int{14}},
		},
	})
	// doc2 = "token1 token2"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 2,
		Keywords: []types.KeywordIndex{
			{"token1", 0, []int{0}},
			{"token2", 0, []int{7}},
		},
	})
	// doc3 = "token2"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 3,
		Keywords: []types.KeywordIndex{
			{"token2", 0, []int{0}},
		},
	})
	// doc7 = "token1 token3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 7,
		Keywords: []types.KeywordIndex{
			{"token1", 0, []int{0}},
			{"token3", 0, []int{7}},
		},
	})
	// doc9 = "token3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 9,
		Keywords: []types.KeywordIndex{
			{"token3", 0, []int{0}},
		},
	})

	utils.Expect(t, "1 2 7 ", indicesToString(&indexer, "token1"))
	utils.Expect(t, "0 1 2 3 ", indicesToString(&indexer, "token2"))
	utils.Expect(t, "0 1 7 9 ", indicesToString(&indexer, "token3"))

	utils.Expect(t, "", indexedDocsToString(indexer.Lookup([]string{"token4"}, []string{}, nil, false)))

	utils.Expect(t, "[7 0 [0]] [2 0 [0]] [1 0 [0]] ",
		indexedDocsToString(indexer.Lookup([]string{"token1"}, []string{}, nil, false)))
	utils.Expect(t, "", indexedDocsToString(indexer.Lookup([]string{"token1", "token4"}, []string{}, nil, false)))

	utils.Expect(t, "[2 1 [0 7]] [1 1 [0 7]] ",
		indexedDocsToString(indexer.Lookup([]string{"token1", "token2"}, []string{}, nil, false)))
	utils.Expect(t, "[2 13 [7 0]] [1 13 [7 0]] ",
		indexedDocsToString(indexer.Lookup([]string{"token2", "token1"}, []string{}, nil, false)))
	utils.Expect(t, "[7 1 [0 7]] [1 8 [0 14]] ",
		indexedDocsToString(indexer.Lookup([]string{"token1", "token3"}, []string{}, nil, false)))
	utils.Expect(t, "[7 13 [7 0]] [1 20 [14 0]] ",
		indexedDocsToString(indexer.Lookup([]string{"token3", "token1"}, []string{}, nil, false)))
	utils.Expect(t, "[1 1 [7 14]] [0 1 [0 7]] ",
		indexedDocsToString(indexer.Lookup([]string{"token2", "token3"}, []string{}, nil, false)))
	utils.Expect(t, "[1 13 [14 7]] [0 13 [7 0]] ",
		indexedDocsToString(indexer.Lookup([]string{"token3", "token2"}, []string{}, nil, false)))

	utils.Expect(t, "[1 2 [0 7 14]] ",
		indexedDocsToString(indexer.Lookup([]string{"token1", "token2", "token3"}, []string{}, nil, false)))
	utils.Expect(t, "[1 26 [14 7 0]] ",
		indexedDocsToString(indexer.Lookup([]string{"token3", "token2", "token1"}, []string{}, nil, false)))
}

func TestDocIdsIndex(t *testing.T) {
	var indexer Indexer
	indexer.Init(types.IndexerInitOptions{IndexType: types.DocIdsIndex})
	// doc0 = "token2 token3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 0,
		Keywords: []types.KeywordIndex{
			{"token2", 0, []int{0}},
			{"token3", 0, []int{7}},
		},
	})
	// doc1 = "token1 token2 token3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 1,
		Keywords: []types.KeywordIndex{
			{"token1", 0, []int{0}},
			{"token2", 0, []int{7}},
			{"token3", 0, []int{14}},
		},
	})
	// doc2 = "token1 token2"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 2,
		Keywords: []types.KeywordIndex{
			{"token1", 0, []int{0}},
			{"token2", 0, []int{7}},
		},
	})
	// doc3 = "token2"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 3,
		Keywords: []types.KeywordIndex{
			{"token2", 0, []int{0}},
		},
	})
	// doc7 = "token1 token3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 7,
		Keywords: []types.KeywordIndex{
			{"token1", 0, []int{0}},
			{"token3", 0, []int{7}},
		},
	})
	// doc9 = "token3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 9,
		Keywords: []types.KeywordIndex{
			{"token3", 0, []int{0}},
		},
	})

	utils.Expect(t, "1 2 7 ", indicesToString(&indexer, "token1"))
	utils.Expect(t, "0 1 2 3 ", indicesToString(&indexer, "token2"))
	utils.Expect(t, "0 1 7 9 ", indicesToString(&indexer, "token3"))

	utils.Expect(t, "", indexedDocsToString(indexer.Lookup([]string{"token4"}, []string{}, nil, false)))

	utils.Expect(t, "[7 0 []] [2 0 []] [1 0 []] ",
		indexedDocsToString(indexer.Lookup([]string{"token1"}, []string{}, nil, false)))
	utils.Expect(t, "", indexedDocsToString(indexer.Lookup([]string{"token1", "token4"}, []string{}, nil, false)))

	utils.Expect(t, "[2 0 []] [1 0 []] ",
		indexedDocsToString(indexer.Lookup([]string{"token1", "token2"}, []string{}, nil, false)))
	utils.Expect(t, "[2 0 []] [1 0 []] ",
		indexedDocsToString(indexer.Lookup([]string{"token2", "token1"}, []string{}, nil, false)))
	utils.Expect(t, "[7 0 []] [1 0 []] ",
		indexedDocsToString(indexer.Lookup([]string{"token1", "token3"}, []string{}, nil, false)))
	utils.Expect(t, "[7 0 []] [1 0 []] ",
		indexedDocsToString(indexer.Lookup([]string{"token3", "token1"}, []string{}, nil, false)))
	utils.Expect(t, "[1 0 []] [0 0 []] ",
		indexedDocsToString(indexer.Lookup([]string{"token2", "token3"}, []string{}, nil, false)))
	utils.Expect(t, "[1 0 []] [0 0 []] ",
		indexedDocsToString(indexer.Lookup([]string{"token3", "token2"}, []string{}, nil, false)))

	utils.Expect(t, "[1 0 []] ",
		indexedDocsToString(indexer.Lookup([]string{"token1", "token2", "token3"}, []string{}, nil, false)))
	utils.Expect(t, "[1 0 []] ",
		indexedDocsToString(indexer.Lookup([]string{"token3", "token2", "token1"}, []string{}, nil, false)))
}

func TestLookupWithProximity(t *testing.T) {
	var indexer Indexer
	indexer.Init(types.IndexerInitOptions{IndexType: types.LocationsIndex})

	// doc0 = "token2 token4 token4 token2 token3 token4"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 0,
		Keywords: []types.KeywordIndex{
			{"token2", 0, []int{0, 21}},
			{"token3", 0, []int{28}},
			{"token4", 0, []int{7, 14, 35}},
		},
	})
	utils.Expect(t, "[0 1 [21 28]] ",
		indexedDocsToString(indexer.Lookup([]string{"token2", "token3"}, []string{}, nil, false)))

	// doc0 = "t2 t1 . . . t2 t3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 0,
		Keywords: []types.KeywordIndex{
			{"t1", 0, []int{3}},
			{"t2", 0, []int{0, 12}},
			{"t3", 0, []int{15}},
		},
	})
	utils.Expect(t, "[0 8 [3 12 15]] ",
		indexedDocsToString(indexer.Lookup([]string{"t1", "t2", "t3"}, []string{}, nil, false)))

	// doc0 = "t3 t2 t1 . . . . . t2 t3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 0,
		Keywords: []types.KeywordIndex{
			{"t1", 0, []int{6}},
			{"t2", 0, []int{3, 19}},
			{"t3", 0, []int{0, 22}},
		},
	})
	utils.Expect(t, "[0 10 [6 3 0]] ",
		indexedDocsToString(indexer.Lookup([]string{"t1", "t2", "t3"}, []string{}, nil, false)))
}

func TestLookupWithPartialLocations(t *testing.T) {
	var indexer Indexer
	indexer.Init(types.IndexerInitOptions{IndexType: types.LocationsIndex})
	// doc0 = "token2 token4 token4 token2 token3 token4" + "label1"(不在文本中)
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 0,
		Keywords: []types.KeywordIndex{
			{"token2", 0, []int{0, 21}},
			{"token3", 0, []int{28}},
			{"label1", 0, []int{}},
			{"token4", 0, []int{7, 14, 35}},
		},
	})
	// doc1 = "token2 token4 token4 token2 token3 token4"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 1,
		Keywords: []types.KeywordIndex{
			{"token2", 0, []int{0, 21}},
			{"token3", 0, []int{28}},
			{"token4", 0, []int{7, 14, 35}},
		},
	})

	utils.Expect(t, "0 ", indicesToString(&indexer, "label1"))

	utils.Expect(t, "[0 1 [21 28]] ",
		indexedDocsToString(indexer.Lookup([]string{"token2", "token3"}, []string{"label1"}, nil, false)))
}

func TestLookupWithBM25(t *testing.T) {
	var indexer Indexer
	indexer.Init(types.IndexerInitOptions{
		IndexType: types.FrequenciesIndex,
		BM25Parameters: &types.BM25Parameters{
			K1: 1,
			B:  1,
		},
	})
	// doc0 = "token2 token4 token4 token2 token3 token4"
	indexer.AddDocument(&types.DocumentIndex{
		DocId:       0,
		TokenLength: 6,
		Keywords: []types.KeywordIndex{
			{"token2", 3, []int{0, 21}},
			{"token3", 7, []int{28}},
			{"token4", 15, []int{7, 14, 35}},
		},
	})
	// doc0 = "token6 token7"
	indexer.AddDocument(&types.DocumentIndex{
		DocId:       1,
		TokenLength: 2,
		Keywords: []types.KeywordIndex{
			{"token6", 3, []int{0}},
			{"token7", 15, []int{7}},
		},
	})

	outputs, _ := indexer.Lookup([]string{"token2", "token3", "token4"}, []string{}, nil, false)

	// BM25 = log2(3) * (12/9 + 28/17 + 60/33) = 6.3433
	utils.Expect(t, "76055", int(outputs[0].BM25*10000))
}

func TestLookupWithinDocIds(t *testing.T) {
	var indexer Indexer
	indexer.Init(types.IndexerInitOptions{IndexType: types.LocationsIndex})
	// doc0 = "token2 token3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 0,
		Keywords: []types.KeywordIndex{
			{"token2", 0, []int{0}},
			{"token3", 0, []int{7}},
		},
	})
	// doc1 = "token1 token2 token3"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 1,
		Keywords: []types.KeywordIndex{
			{"token1", 0, []int{0}},
			{"token2", 0, []int{7}},
			{"token3", 0, []int{14}},
		},
	})
	// doc2 = "token1 token2"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 2,
		Keywords: []types.KeywordIndex{
			{"token1", 0, []int{0}},
			{"token2", 0, []int{7}},
		},
	})
	// doc3 = "token2"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 3,
		Keywords: []types.KeywordIndex{
			{"token2", 0, []int{0}},
		},
	})

	docIds := make(map[uint64]bool)
	docIds[0] = true
	docIds[2] = true
	utils.Expect(t, "[2 0 [7]] [0 0 [0]] ",
		indexedDocsToString(indexer.Lookup([]string{"token2"}, []string{}, docIds, false)))
}

func TestLookupWithLocations(t *testing.T) {
	var indexer Indexer
	indexer.Init(types.IndexerInitOptions{IndexType: types.LocationsIndex})
	// doc0 = "token2 token4 token4 token2 token3 token4"
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 0,
		Keywords: []types.KeywordIndex{
			{"token2", 0, []int{0, 21}},
			{"token3", 0, []int{28}},
			{"token4", 0, []int{7, 14, 35}},
		},
	})

	docs, _ := indexer.Lookup([]string{"token2", "token3"}, []string{}, nil, false)
	utils.Expect(t, "[[0 21] [28]]", docs[0].TokenLocations)
}

func TestLookupWithLocations1(t *testing.T) {

	type Data struct {
		Id      int
		Content string
		Labels  []string
	}

	datas := make([]Data, 0)

	data0 := Data{Id: 0, Content: "此次百度收购将成中国互联网最大并购", Labels: []string{"百度", "中国"}}
	datas = append(datas, data0)

	data1 := Data{Id: 1, Content: "百度宣布拟全资收购91无线业务", Labels: []string{"百度"}}
	datas = append(datas, data1)

	data2 := Data{Id: 2, Content: "百度是中国最大的搜索引擎", Labels: []string{"百度"}}
	datas = append(datas, data2)

	data3 := Data{Id: 3, Content: "百度在研制无人汽车", Labels: []string{"百度"}}
	datas = append(datas, data3)

	data4 := Data{Id: 4, Content: "BAT是中国互联网三巨头", Labels: []string{"百度"}}
	datas = append(datas, data4)

	// 初始化
	searcher_locations := engine.Engine{}
	searcher_locations.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../data/dictionary.txt",
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
	})
	defer searcher_locations.Close()
	for _, data := range datas {
		searcher_locations.IndexDocument(uint64(data.Id), types.DocumentIndexData{Content: data.Content, Labels: data.Labels})
	}
	searcher_locations.FlushIndex()
	res_locations := searcher_locations.Search(types.SearchRequest{Text: "百度"})

	searcher_docids := engine.Engine{}
	searcher_docids.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../data/dictionary.txt",
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.DocIdsIndex,
		},
	})
	defer searcher_docids.Close()
	for _, data := range datas {
		searcher_docids.IndexDocument(uint64(data.Id), types.DocumentIndexData{Content: data.Content, Labels: data.Labels})
	}
	searcher_docids.FlushIndex()
	res_docids := searcher_docids.Search(types.SearchRequest{Text: "百度"})
	if res_docids.NumDocs != res_locations.NumDocs {
		t.Errorf("期待的搜索结果个数=\"%d\", 实际=\"%d\"", res_docids.NumDocs, res_locations.NumDocs)
	}
}
