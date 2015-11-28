package core

import (
	"github.com/huichen/wukong/types"
	"github.com/huichen/wukong/utils"
	"log"
	"sort"
)

type Ranker struct {
	shard int
	// 文档信息
	*types.DocInfosShard

	initialized bool
}

func (ranker *Ranker) Init(shard int) {
	if ranker.initialized == true {
		log.Fatal("排序器不能初始化两次")
	}
	ranker.initialized = true

	ranker.shard = shard

	AddDocInfosShard(shard)
	ranker.DocInfosShard = DocInfoGroup[shard]
}

// 给某个文档添加评分字段
func (ranker *Ranker) AddDoc(docId uint64, fields interface{}, dealDocInfoChan <-chan bool) *types.DocInfo {
	if ranker.initialized == false {
		log.Fatal("排序器尚未初始化")
	}

	<-dealDocInfoChan // 等待索引器处理完成

	ranker.DocInfosShard.Lock()
	defer ranker.DocInfosShard.Unlock()
	if _, found := ranker.DocInfosShard.DocInfos[docId]; !found {
		ranker.DocInfosShard.DocInfos[docId] = new(types.DocInfo)
		ranker.DocInfosShard.NumDocuments++
	}
	ranker.DocInfosShard.DocInfos[docId].Fields = fields
	return ranker.DocInfosShard.DocInfos[docId]
}

// 删除某个文档的评分字段
func (ranker *Ranker) RemoveDoc(docId uint64) {
	if ranker.initialized == false {
		log.Fatal("排序器尚未初始化")
	}

	ranker.DocInfosShard.Lock()
	delete(ranker.DocInfosShard.DocInfos, docId)
	ranker.DocInfosShard.NumDocuments--
	ranker.DocInfosShard.Unlock()
}

// 给文档评分并排序
func (ranker *Ranker) Rank(
	docs []types.IndexedDocument, options types.RankOptions, countDocsOnly bool) (types.ScoredDocuments, int) {
	if ranker.initialized == false {
		log.Fatal("排序器尚未初始化")
	}
	// 对每个文档评分
	var outputDocs types.ScoredDocuments
	numDocs := 0
	for _, d := range docs {
		ranker.DocInfosShard.RLock()
		// 判断doc是否存在
		if _, ok := ranker.DocInfosShard.DocInfos[d.DocId]; ok {
			fs := ranker.DocInfosShard.DocInfos[d.DocId].Fields
			ranker.DocInfosShard.RUnlock()
			// 计算评分并剔除没有分值的文档
			scores := options.ScoringCriteria.Score(d, fs)
			if len(scores) > 0 {
				if !countDocsOnly {
					outputDocs = append(outputDocs, types.ScoredDocument{
						DocId:                 d.DocId,
						Scores:                scores,
						TokenSnippetLocations: d.TokenSnippetLocations,
						TokenLocations:        d.TokenLocations})
				}
				numDocs++
			}
		} else {
			ranker.DocInfosShard.RUnlock()
		}
	}

	// 排序
	if !countDocsOnly {
		if options.ReverseOrder {
			sort.Sort(sort.Reverse(outputDocs))
		} else {
			sort.Sort(outputDocs)
		}
		// 当用户要求只返回部分结果时返回部分结果
		var start, end int
		if options.MaxOutputs != 0 {
			start = utils.MinInt(options.OutputOffset, len(outputDocs))
			end = utils.MinInt(options.OutputOffset+options.MaxOutputs, len(outputDocs))
		} else {
			start = utils.MinInt(options.OutputOffset, len(outputDocs))
			end = len(outputDocs)
		}
		return outputDocs[start:end], numDocs
	}
	return outputDocs, numDocs
}
