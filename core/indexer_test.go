package core

import (
	"github.com/huichen/wukong/types"
	"github.com/huichen/wukong/utils"
	"testing"
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

func TestLogicLookup(t *testing.T) {
	var indexer Indexer
	indexer.Init(types.IndexerInitOptions{IndexType: types.LocationsIndex})
	// doc0 = "label1, label2, label3, label4"(不在文本中)
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 0,
		Keywords: []types.KeywordIndex{
			{"label1", 0, []int{}},
			{"label2", 0, []int{}},
			{"label3", 0, []int{}},
			{"label4", 0, []int{}},
		},
	})

	// doc1 = "label1, label2, label5, label6"(不在文本中)
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 1,
		Keywords: []types.KeywordIndex{
			{"label1", 0, []int{}},
			{"label2", 0, []int{}},
			{"label5", 0, []int{}},
			{"label6", 0, []int{}},
		},
	})

	// test null LogicExpression
	empty := types.LogicExpression{}
	docs, _ := indexer.LogicLookup(nil, false, empty)
	if len(docs) != 0 {
		t.Error("error")
	}

	// test only and
	m := []string{"label1"}
	lm := types.LogicExpression{MustLabels: m}
	utils.Expect(t, "[1] [0] ", indexedDocIdsToString(indexer.LogicLookup(nil, false, lm)))

	// test A && (B ||) = A && B
	s := []string{"label2"}
	ls := types.LogicExpression{
		MustLabels:   m,
		ShouldLabels: s,
	}
	utils.Expect(t, "[1] [0] ", indexedDocIdsToString(indexer.LogicLookup(nil, false, ls)))

	// test A && (B||) && -C = A && B && -C : ("-" represent logic NOT)
	n := []string{"label5"}
	lmsn := types.LogicExpression{
		MustLabels:   m,
		ShouldLabels: s,
		NotInLabels:  n,
	}
	utils.Expect(t, "[0] ", indexedDocIdsToString(indexer.LogicLookup(nil, false, lmsn)))

	// test: (A || B)
	a := []string{"label3", "label5"}
	lor := types.LogicExpression{
		ShouldLabels: a,
	}
	utils.Expect(t, "[1] [0] ", indexedDocIdsToString(indexer.LogicLookup(nil, false, lor)))

	// test: -B
	b := []string{"label1"}
	lnot := types.LogicExpression{
		NotInLabels: b,
	}
	utils.Expect(t, "", indexedDocIdsToString(indexer.LogicLookup(nil, false, lnot)))

	// test: (A || B) && -C
	lornot := types.LogicExpression{
		ShouldLabels: a,
		NotInLabels:  b,
	}
	utils.Expect(t, "", indexedDocIdsToString(indexer.LogicLookup(nil, false, lornot)))
}

func TestLookupWithLogicExpression(t *testing.T) {
	var indexer Indexer
	indexer.Init(types.IndexerInitOptions{IndexType: types.LocationsIndex})
	indexer.AddDocument(&types.DocumentIndex{
		DocId: 1,
		Keywords: []types.KeywordIndex{
			{"label1", 0, []int{}},
			{"label2", 0, []int{}},
		},
	})

	indexer.AddDocument(&types.DocumentIndex{
		DocId: 2,
		Keywords: []types.KeywordIndex{
			{"label1", 0, []int{}},
			{"label3", 0, []int{}},
		},
	})

	indexer.AddDocument(&types.DocumentIndex{
		DocId: 5,
		Keywords: []types.KeywordIndex{
			{"label1", 0, []int{}},
			{"label2", 0, []int{}},
		},
	})

	indexer.AddDocument(&types.DocumentIndex{
		DocId: 6,
		Keywords: []types.KeywordIndex{
			{"label3", 0, []int{}},
		},
	})

	indexer.AddDocument(&types.DocumentIndex{
		DocId: 7,
		Keywords: []types.KeywordIndex{
			{"label4", 0, []int{}},
		},
	})

	indexer.AddDocument(&types.DocumentIndex{
		DocId: 9,
		Keywords: []types.KeywordIndex{
			{"label1", 0, []int{}},
			{"label4", 0, []int{}},
		},
	})

	indexer.AddDocument(&types.DocumentIndex{
		DocId: 10,
		Keywords: []types.KeywordIndex{
			{"label2", 0, []int{}},
			{"label4", 0, []int{}},
		},
	})

	indexer.AddDocument(&types.DocumentIndex{
		DocId: 13,
		Keywords: []types.KeywordIndex{
			{"label3", 0, []int{}},
			{"label4", 0, []int{}},
		},
	})

	indexer.AddDocument(&types.DocumentIndex{
		DocId: 18,
		Keywords: []types.KeywordIndex{
			{"label4", 0, []int{}},
		},
	})

	// overview of inverted index:
	// label1: 1, 2, 5, 9
	// label2: 1, 5, 10
	// label3: 2, 6, 13
	// label4: 7, 9, 10, 13, 18

	// test not exists
	utils.Expect(t, "", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels: []string{"label9999"},
	})))

	// test without logicexpression
	utils.Expect(t, "", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false)))

	// test not permission: -A && -B
	utils.Expect(t, "", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		NotInLabels: []string{"label1", "label2"},
	})))

	// test exists: A
	utils.Expect(t, "[9] [5] [2] [1] ", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels: []string{"label1"},
	})))

	// test exists: A && B
	utils.Expect(t, "[9] ", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels: []string{"label1", "label4"},
	})))

	utils.Expect(t, "[5] [1] ", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels: []string{"label1", "label2"},
	})))

	utils.Expect(t, "", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels: []string{"label3", "label2"},
	})))

	// test exists: A && (B ||)
	utils.Expect(t, "[5] [1] ", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels:   []string{"label1"},
		ShouldLabels: []string{"label2"},
	})))

	// test exists: A && (B ||) && -C
	utils.Expect(t, "[5] [1] ", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels:   []string{"label1"},
		ShouldLabels: []string{"label2"},
		NotInLabels:  []string{"label3"},
	})))
	utils.Expect(t, "", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels:   []string{"label1"},
		ShouldLabels: []string{"label2"},
		NotInLabels:  []string{"label1"},
	})))

	// test exists: A && (B || C)
	utils.Expect(t, "[9] [5] [1] ", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels:   []string{"label1"},
		ShouldLabels: []string{"label2", "label4"},
	})))

	// test exists: A && (B || C) && D
	utils.Expect(t, "", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels:   []string{"label1", "label3"},
		ShouldLabels: []string{"label2", "label4"},
	})))
	utils.Expect(t, "", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels:   []string{"label1", "label2"},
		ShouldLabels: []string{"label3", "label4"},
	})))

	// test exists: A && (B || C || D)
	utils.Expect(t, "[9] [5] [2] [1] ", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels:   []string{"label1"},
		ShouldLabels: []string{"label2", "label4", "label3"},
	})))

	// test exists: (A || B || C || D)
	utils.Expect(t, "[18] [13] [10] [9] [7] [6] [5] [2] [1] ", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		ShouldLabels: []string{"label2", "label4", "label3", "label1"},
	})))

	// test exists: (B || C || D) && -A
	utils.Expect(t, "[18] [13] [10] [7] [6] ", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		ShouldLabels: []string{"label2", "label4", "label3"},
		NotInLabels:  []string{"label1"},
	})))

	// test exists: (B || C || D) && -A with not exists label9, label6
	utils.Expect(t, "[18] [13] [10] [7] [6] ", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		ShouldLabels: []string{"label2", "label4", "label3", "label6"},
		NotInLabels:  []string{"label1", "label9"},
	})))

	// test exists: (B || C || D) && -A with not exists label9, label6
	utils.Expect(t, "", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		ShouldLabels: []string{"label6"},
		NotInLabels:  []string{"label1", "label9"},
	})))

	// test exists: (B && C) && -A with not exists label9, label6
	utils.Expect(t, "[10] ", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels:   []string{"label4", "label2"},
		ShouldLabels: []string{"label2", "label6"},
		NotInLabels:  []string{"label1", "label9"},
	})))

	// test exists: (B && C) && -A with not exists label9, label6, label8
	utils.Expect(t, "", indexedDocIdsToString(indexer.Lookup([]string{}, []string{}, nil, false, types.LogicExpression{
		MustLabels:   []string{"label4", "label8"},
		ShouldLabels: []string{"label2", "label6"},
		NotInLabels:  []string{"label1", "label9"},
	})))

}
