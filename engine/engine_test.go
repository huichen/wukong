package engine

import (
	"encoding/gob"
	"github.com/huichen/wukong/types"
	"github.com/huichen/wukong/utils"
	"os"
	"reflect"
	"testing"
)

type ScoringFields struct {
	A, B, C float32
}

func AddDocs(engine *Engine) {
	docId := uint64(1)
	// 因为需要保证文档全部被加入到索引中，所以 forceUpdate 全部设置成 true
	engine.IndexDocument(docId, types.DocumentIndexData{
		Content: "中国有十三亿人口人口",
		Fields:  ScoringFields{1, 2, 3},
	}, true)
	docId++
	engine.IndexDocument(docId, types.DocumentIndexData{
		Content: "中国人口",
		Fields:  nil,
	}, true)
	docId++
	engine.IndexDocument(docId, types.DocumentIndexData{
		Content: "有人口",
		Fields:  ScoringFields{2, 3, 1},
	}, true)
	docId++
	engine.IndexDocument(docId, types.DocumentIndexData{
		Content: "有十三亿人口",
		Fields:  ScoringFields{2, 3, 3},
	}, true)
	docId++
	engine.IndexDocument(docId, types.DocumentIndexData{
		Content: "中国十三亿人口",
		Fields:  ScoringFields{0, 9, 1},
	}, true)

	engine.FlushIndex()
}

type RankByTokenProximity struct {
}

func (rule RankByTokenProximity) Score(
	doc types.IndexedDocument, fields interface{}) []float32 {
	if doc.TokenProximity < 0 {
		return []float32{}
	}
	return []float32{1.0 / (float32(doc.TokenProximity) + 1)}
}

func TestEngineIndexDocument(t *testing.T) {
	var engine Engine
	engine.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			OutputOffset:    0,
			MaxOutputs:      10,
			ScoringCriteria: &RankByTokenProximity{},
		},
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
	})

	AddDocs(&engine)

	outputs := engine.Search(types.SearchRequest{Text: "中国人口"})
	utils.Expect(t, "2", len(outputs.Tokens))
	utils.Expect(t, "中国", outputs.Tokens[0])
	utils.Expect(t, "人口", outputs.Tokens[1])
	utils.Expect(t, "3", len(outputs.Docs))

	utils.Expect(t, "2", outputs.Docs[0].DocId)
	utils.Expect(t, "1000", int(outputs.Docs[0].Scores[0]*1000))
	utils.Expect(t, "[0 6]", outputs.Docs[0].TokenSnippetLocations)

	utils.Expect(t, "5", outputs.Docs[1].DocId)
	utils.Expect(t, "100", int(outputs.Docs[1].Scores[0]*1000))
	utils.Expect(t, "[0 15]", outputs.Docs[1].TokenSnippetLocations)

	utils.Expect(t, "1", outputs.Docs[2].DocId)
	utils.Expect(t, "76", int(outputs.Docs[2].Scores[0]*1000))
	utils.Expect(t, "[0 18]", outputs.Docs[2].TokenSnippetLocations)
}

func TestReverseOrder(t *testing.T) {
	var engine Engine
	engine.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			ReverseOrder:    true,
			OutputOffset:    0,
			MaxOutputs:      10,
			ScoringCriteria: &RankByTokenProximity{},
		},
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
	})

	AddDocs(&engine)

	outputs := engine.Search(types.SearchRequest{Text: "中国人口"})
	utils.Expect(t, "3", len(outputs.Docs))

	utils.Expect(t, "1", outputs.Docs[0].DocId)
	utils.Expect(t, "5", outputs.Docs[1].DocId)
	utils.Expect(t, "2", outputs.Docs[2].DocId)
}

func TestOffsetAndMaxOutputs(t *testing.T) {
	var engine Engine
	engine.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			ReverseOrder:    true,
			OutputOffset:    1,
			MaxOutputs:      3,
			ScoringCriteria: &RankByTokenProximity{},
		},
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
	})

	AddDocs(&engine)

	outputs := engine.Search(types.SearchRequest{Text: "中国人口"})
	utils.Expect(t, "2", len(outputs.Docs))

	utils.Expect(t, "5", outputs.Docs[0].DocId)
	utils.Expect(t, "2", outputs.Docs[1].DocId)
}

type TestScoringCriteria struct {
}

func (criteria TestScoringCriteria) Score(
	doc types.IndexedDocument, fields interface{}) []float32 {
	if reflect.TypeOf(fields) != reflect.TypeOf(ScoringFields{}) {
		return []float32{}
	}
	fs := fields.(ScoringFields)
	return []float32{float32(doc.TokenProximity)*fs.A + fs.B*fs.C}
}

func TestSearchWithCriteria(t *testing.T) {
	var engine Engine
	engine.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			ScoringCriteria: TestScoringCriteria{},
		},
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
	})

	AddDocs(&engine)

	outputs := engine.Search(types.SearchRequest{Text: "中国人口"})
	utils.Expect(t, "2", len(outputs.Docs))

	utils.Expect(t, "1", outputs.Docs[0].DocId)
	utils.Expect(t, "18000", int(outputs.Docs[0].Scores[0]*1000))

	utils.Expect(t, "5", outputs.Docs[1].DocId)
	utils.Expect(t, "9000", int(outputs.Docs[1].Scores[0]*1000))
}

func TestCompactIndex(t *testing.T) {
	var engine Engine
	engine.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			ScoringCriteria: TestScoringCriteria{},
		},
	})

	AddDocs(&engine)

	outputs := engine.Search(types.SearchRequest{Text: "中国人口"})
	utils.Expect(t, "2", len(outputs.Docs))

	utils.Expect(t, "5", outputs.Docs[0].DocId)
	utils.Expect(t, "9000", int(outputs.Docs[0].Scores[0]*1000))

	utils.Expect(t, "1", outputs.Docs[1].DocId)
	utils.Expect(t, "6000", int(outputs.Docs[1].Scores[0]*1000))
}

type BM25ScoringCriteria struct {
}

func (criteria BM25ScoringCriteria) Score(
	doc types.IndexedDocument, fields interface{}) []float32 {
	if reflect.TypeOf(fields) != reflect.TypeOf(ScoringFields{}) {
		return []float32{}
	}
	return []float32{doc.BM25}
}

func TestFrequenciesIndex(t *testing.T) {
	var engine Engine
	engine.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			ScoringCriteria: BM25ScoringCriteria{},
		},
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.FrequenciesIndex,
		},
	})

	AddDocs(&engine)

	outputs := engine.Search(types.SearchRequest{Text: "中国人口"})
	utils.Expect(t, "2", len(outputs.Docs))

	utils.Expect(t, "5", outputs.Docs[0].DocId)
	utils.Expect(t, "2349", int(outputs.Docs[0].Scores[0]*1000))

	utils.Expect(t, "1", outputs.Docs[1].DocId)
	utils.Expect(t, "2320", int(outputs.Docs[1].Scores[0]*1000))
}

func TestRemoveDocument(t *testing.T) {
	var engine Engine
	engine.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			ScoringCriteria: TestScoringCriteria{},
		},
	})

	AddDocs(&engine)
	engine.RemoveDocument(5, true)
	engine.FlushIndex()

	outputs := engine.Search(types.SearchRequest{Text: "中国人口"})
	utils.Expect(t, "1", len(outputs.Docs))

	utils.Expect(t, "1", outputs.Docs[0].DocId)
	utils.Expect(t, "6000", int(outputs.Docs[0].Scores[0]*1000))
}

func TestEngineIndexDocumentWithTokens(t *testing.T) {
	var engine Engine
	engine.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			OutputOffset:    0,
			MaxOutputs:      10,
			ScoringCriteria: &RankByTokenProximity{},
		},
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
	})

	docId := uint64(1)
	engine.IndexDocument(docId, types.DocumentIndexData{
		Content: "",
		Tokens: []types.TokenData{
			{"中国", []int{0}},
			{"人口", []int{18, 24}},
		},
		Fields: ScoringFields{1, 2, 3},
	}, true)
	docId++
	engine.IndexDocument(docId, types.DocumentIndexData{
		Content: "",
		Tokens: []types.TokenData{
			{"中国", []int{0}},
			{"人口", []int{6}},
		},
		Fields: ScoringFields{1, 2, 3},
	}, true)
	docId++
	engine.IndexDocument(docId, types.DocumentIndexData{
		Content: "中国十三亿人口",
		Fields:  ScoringFields{0, 9, 1},
	}, true)

	engine.FlushIndex()

	outputs := engine.Search(types.SearchRequest{Text: "中国人口"})
	utils.Expect(t, "2", len(outputs.Tokens))
	utils.Expect(t, "中国", outputs.Tokens[0])
	utils.Expect(t, "人口", outputs.Tokens[1])
	utils.Expect(t, "3", len(outputs.Docs))

	utils.Expect(t, "2", outputs.Docs[0].DocId)
	utils.Expect(t, "1000", int(outputs.Docs[0].Scores[0]*1000))
	utils.Expect(t, "[0 6]", outputs.Docs[0].TokenSnippetLocations)

	utils.Expect(t, "3", outputs.Docs[1].DocId)
	utils.Expect(t, "100", int(outputs.Docs[1].Scores[0]*1000))
	utils.Expect(t, "[0 15]", outputs.Docs[1].TokenSnippetLocations)

	utils.Expect(t, "1", outputs.Docs[2].DocId)
	utils.Expect(t, "76", int(outputs.Docs[2].Scores[0]*1000))
	utils.Expect(t, "[0 18]", outputs.Docs[2].TokenSnippetLocations)
}

func TestEngineIndexDocumentWithPersistentStorage(t *testing.T) {
	gob.Register(ScoringFields{})
	var engine Engine
	engine.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			OutputOffset:    0,
			MaxOutputs:      10,
			ScoringCriteria: &RankByTokenProximity{},
		},
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
		UsePersistentStorage:    true,
		PersistentStorageFolder: "wukong.persistent",
		PersistentStorageShards: 2,
	})
	AddDocs(&engine)
	engine.RemoveDocument(5, true)
	engine.Close()

	var engine1 Engine
	engine1.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			OutputOffset:    0,
			MaxOutputs:      10,
			ScoringCriteria: &RankByTokenProximity{},
		},
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
		UsePersistentStorage:    true,
		PersistentStorageFolder: "wukong.persistent",
		PersistentStorageShards: 2,
	})
	engine1.FlushIndex()

	outputs := engine1.Search(types.SearchRequest{Text: "中国人口"})
	utils.Expect(t, "2", len(outputs.Tokens))
	utils.Expect(t, "中国", outputs.Tokens[0])
	utils.Expect(t, "人口", outputs.Tokens[1])
	utils.Expect(t, "2", len(outputs.Docs))

	utils.Expect(t, "2", outputs.Docs[0].DocId)
	utils.Expect(t, "1000", int(outputs.Docs[0].Scores[0]*1000))
	utils.Expect(t, "[0 6]", outputs.Docs[0].TokenSnippetLocations)

	utils.Expect(t, "1", outputs.Docs[1].DocId)
	utils.Expect(t, "76", int(outputs.Docs[1].Scores[0]*1000))
	utils.Expect(t, "[0 18]", outputs.Docs[1].TokenSnippetLocations)

	engine1.Close()
	os.RemoveAll("wukong.persistent")
}

func TestCountDocsOnly(t *testing.T) {
	var engine Engine
	engine.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			ReverseOrder:    true,
			OutputOffset:    0,
			MaxOutputs:      1,
			ScoringCriteria: &RankByTokenProximity{},
		},
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
	})

	AddDocs(&engine)
	engine.RemoveDocument(5, true)
	engine.FlushIndex()

	outputs := engine.Search(types.SearchRequest{Text: "中国人口", CountDocsOnly: true})
	utils.Expect(t, "0", len(outputs.Docs))
	utils.Expect(t, "2", len(outputs.Tokens))
	utils.Expect(t, "2", outputs.NumDocs)
}

func TestSearchWithin(t *testing.T) {
	var engine Engine
	engine.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../testdata/test_dict.txt",
		DefaultRankOptions: &types.RankOptions{
			ReverseOrder:    true,
			OutputOffset:    0,
			MaxOutputs:      10,
			ScoringCriteria: &RankByTokenProximity{},
		},
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
	})

	AddDocs(&engine)

	docIds := make(map[uint64]bool)
	docIds[5] = true
	docIds[1] = true
	outputs := engine.Search(types.SearchRequest{
		Text:   "中国人口",
		DocIds: docIds,
	})
	utils.Expect(t, "2", len(outputs.Tokens))
	utils.Expect(t, "中国", outputs.Tokens[0])
	utils.Expect(t, "人口", outputs.Tokens[1])
	utils.Expect(t, "2", len(outputs.Docs))

	utils.Expect(t, "1", outputs.Docs[0].DocId)
	utils.Expect(t, "76", int(outputs.Docs[0].Scores[0]*1000))
	utils.Expect(t, "[0 18]", outputs.Docs[0].TokenSnippetLocations)

	utils.Expect(t, "5", outputs.Docs[1].DocId)
	utils.Expect(t, "100", int(outputs.Docs[1].Scores[0]*1000))
	utils.Expect(t, "[0 15]", outputs.Docs[1].TokenSnippetLocations)
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
	searcher_locations := Engine{}
	searcher_locations.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../data/dictionary.txt",
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
	})
	defer searcher_locations.Close()
	for _, data := range datas {
		searcher_locations.IndexDocument(uint64(data.Id), types.DocumentIndexData{Content: data.Content, Labels: data.Labels}, true)
	}
	searcher_locations.FlushIndex()
	res_locations := searcher_locations.Search(types.SearchRequest{Text: "百度"})

	searcher_docids := Engine{}
	searcher_docids.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../data/dictionary.txt",
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.DocIdsIndex,
		},
	})
	defer searcher_docids.Close()
	for _, data := range datas {
		searcher_docids.IndexDocument(uint64(data.Id), types.DocumentIndexData{Content: data.Content, Labels: data.Labels}, true)
	}
	searcher_docids.FlushIndex()
	res_docids := searcher_docids.Search(types.SearchRequest{Text: "百度"})
	if res_docids.NumDocs != res_locations.NumDocs {
		t.Errorf("期待的搜索结果个数=\"%d\", 实际=\"%d\"", res_docids.NumDocs, res_locations.NumDocs)
	}
}
