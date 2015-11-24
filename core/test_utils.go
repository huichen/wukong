package core

import (
	"fmt"
	"github.com/henrylee2cn/wukong/types"
)

func indicesToString(indexer *Indexer, token string) (output string) {
	indices := indexer.tableLock.table[token]
	for i := 0; i < indexer.getIndexLength(indices); i++ {
		output += fmt.Sprintf("%v ", indexer.getDocId(indices, i))
	}
	return
}

func indexedDocsToString(docs []types.IndexedDocument, numDocs int) (output string) {
	for _, doc := range docs {
		output += fmt.Sprintf("[%v %v %v] ",
			doc.DocId, doc.TokenProximity, doc.TokenSnippetLocations)
	}
	return
}

func scoredDocsToString(docs []types.ScoredDocument) (output string) {
	for _, doc := range docs {
		output += fmt.Sprintf("[%v [", doc.DocId)
		for _, score := range doc.Scores {
			output += fmt.Sprintf("%v ", int(score*1000))
		}
		output += "]] "
	}
	return
}
