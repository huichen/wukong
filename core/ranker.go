package core

import (
	"github.com/henrylee2cn/wukong/types"
	// "github.com/henrylee2cn/wukong/utils"
	"log"
	"sort"
	"sync"
)

type Ranker struct {
	lock struct {
		sync.RWMutex
		fields map[string]interface{}
	}
	initialized bool
}

func (ranker *Ranker) Init() {
	if ranker.initialized == true {
		log.Fatal("排序器不能初始化两次")
	}
	ranker.initialized = true

	ranker.lock.fields = make(map[string]interface{})
}

// 给某个文档添加评分字段
func (ranker *Ranker) AddScoringFields(docId string, fields interface{}) {
	if ranker.initialized == false {
		log.Fatal("排序器尚未初始化")
	}

	ranker.lock.Lock()
	ranker.lock.fields[docId] = fields
	ranker.lock.Unlock()
}

// 删除某个文档的评分字段
func (ranker *Ranker) RemoveScoringFields(docId string) {
	if ranker.initialized == false {
		log.Fatal("排序器尚未初始化")
	}

	ranker.lock.Lock()
	delete(ranker.lock.fields, docId)
	ranker.lock.Unlock()
}

// 给文档评分并排序
func (ranker *Ranker) Rank(docs []types.IndexedDocument, options types.RankOptions) (outputDocs types.ScoredDocuments) {
	if ranker.initialized == false {
		log.Fatal("排序器尚未初始化")
	}

	// 对每个文档评分
	for _, d := range docs {
		ranker.lock.RLock()
		fs := ranker.lock.fields[d.DocId]
		ranker.lock.RUnlock()
		// 计算评分并剔除没有分值的文档
		scores := options.ScoringCriteria.Score(d, fs)
		if len(scores) > 0 {
			outputDocs = append(outputDocs, types.ScoredDocument{
				DocId:                 d.DocId,
				Scores:                scores,
				TokenSnippetLocations: d.TokenSnippetLocations,
				TokenLocations:        d.TokenLocations})
		}
	}

	// 排序
	if options.ReverseOrder {
		sort.Sort(sort.Reverse(outputDocs))
	} else {
		sort.Sort(outputDocs)
	}

	// 当用户要求只返回部分结果时返回部分结果
	// var start, end int
	// if options.MaxOutputs != 0 {
	// 	start = utils.MinInt(options.OutputOffset, len(outputDocs))
	// 	end = utils.MinInt(options.OutputOffset+options.MaxOutputs, len(outputDocs))
	// } else {
	// 	start = utils.MinInt(options.OutputOffset, len(outputDocs))
	// 	end = len(outputDocs)
	// }
	// return outputDocs[start:end]

	return outputDocs
}
