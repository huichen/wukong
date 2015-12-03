package core

import (
	"fmt"
	"github.com/huichen/wukong/types"
)

func indicesToString(indexer *Indexer, token string) (output string) {
	indices := indexer.tableLock.table[token]
	for i := 0; i < indexer.getIndexLength(indices); i++ {
		output += fmt.Sprintf("%d ",
			indexer.getDocId(indices, i))
	}
	return
}

func indexedDocsToString(docs []types.IndexedDocument, numDocs int) (output string) {
	for _, doc := range docs {
		output += fmt.Sprintf("[%d %d %v] ",
			doc.DocId, doc.TokenProximity, doc.TokenSnippetLocations)
	}
	return
}

func scoredDocsToString(docs []types.ScoredDocument) (output string) {
	for _, doc := range docs {
		output += fmt.Sprintf("[%d [", doc.DocId)
		for _, score := range doc.Scores {
			output += fmt.Sprintf("%d ", int(score*1000))
		}
		output += "]] "
	}
	return
}

func indexedDocIdsToString(docs []types.IndexedDocument, numDocs int) (output string) {
	for _, doc := range docs {
		output += fmt.Sprintf("[%d] ",
			doc.DocId)
	}
	return
}
